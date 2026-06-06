package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/initcfg"
)

func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a starter .snagify.yaml from detected project manifests",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("capture failed: %v", err)}
			}

			root := snap.Project.Root
			out := filepath.Join(root, config.FileName)

			if _, statErr := os.Stat(out); statErr == nil && !force {
				return &exitError{code: 2, msg: fmt.Sprintf(
					"%s already exists (use --force to overwrite)", out)}
			}

			doc := initcfg.Generate(snap, root)
			if err := os.WriteFile(out, []byte(doc), 0o644); err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("write %s: %v", out, err)}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", out)
			fmt.Fprintln(cmd.OutOrStdout(), "Review the generated requirements, then run `snagify check`.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing .snagify.yaml")
	return cmd
}
