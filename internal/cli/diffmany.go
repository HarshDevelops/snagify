package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/store"
	"github.com/harshdevelops/snagify/internal/team"
)

func newDiffManyCmd() *cobra.Command {
	var (
		against string
		format  string
	)

	cmd := &cobra.Command{
		Use:   "diff-many <snapshots-dir>",
		Short: "Compare many snapshots against a baseline and summarize team drift",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if against == "" {
				return &exitError{code: 2, msg: "--against <baseline.json> is required"}
			}
			base, err := store.Load(against)
			if err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}

			dir := args[0]
			entries, err := os.ReadDir(dir)
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("read dir %s: %v", dir, err)}
			}

			baseAbs, _ := filepath.Abs(against)

			var inputs []team.Input
			var names []string
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
					continue
				}
				p := filepath.Join(dir, e.Name())
				if abs, _ := filepath.Abs(p); abs == baseAbs {
					continue // don't compare the baseline against itself
				}
				snap, loadErr := store.Load(p)
				if loadErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "skipping %s: %v\n", p, loadErr)
					continue
				}
				name := snap.Hostname
				if name == "" {
					name = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
				}
				inputs = append(inputs, team.Input{Name: name, Snapshot: snap})
				names = append(names, name)
			}

			if len(inputs) == 0 {
				return &exitError{code: 2, msg: fmt.Sprintf("no snapshot .json files found in %s", dir)}
			}
			sort.Strings(names)

			summary := team.Build(base, inputs)
			if err := team.Render(cmd.OutOrStdout(), summary, format); err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&against, "against", "", "baseline JSON to compare each snapshot against")
	cmd.Flags().StringVar(&format, "format", "terminal", "output format: terminal, markdown, or json")
	return cmd
}
