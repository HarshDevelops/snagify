package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/diff"
	"github.com/harshdevelops/snagify/internal/render"
	"github.com/harshdevelops/snagify/internal/store"
)

func newDiffCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "diff <snapshotA.json> <snapshotB.json>",
		Short: "Compare two snapshots and show ranked differences",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := store.Load(args[0])
			if err != nil {
				return err
			}
			b, err := store.Load(args[1])
			if err != nil {
				return err
			}

			res := diff.Compare(a, b, label(args[0], a.Hostname), label(args[1], b.Hostname))
			diff.ApplyProjectContext(&res, a, b)

			w := cmd.OutOrStdout()
			switch strings.ToLower(format) {
			case "markdown", "md":
				render.Markdown(w, res)
			case "json":
				if err := render.JSON(w, res); err != nil {
					return err
				}
			case "", "text", "terminal":
				render.Terminal(w, res)
			default:
				return fmt.Errorf("unknown format %q (use text, markdown, or json)", format)
			}

			// Exit code: 0 = no differences, 1 = differences found.
			if res.HasDifferences() {
				cmd.SilenceUsage = true
				return &exitError{code: 1}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format: text, markdown, or json")
	return cmd
}

// label derives a short machine label from the snapshot filename, falling back
// to the hostname.
func label(path, hostname string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if base != "" {
		return base
	}
	if hostname != "" {
		return hostname
	}
	return path
}

// exitError carries a desired process exit code up to main, optionally with a
// message to print to stderr.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

// Code returns the intended process exit code.
func (e *exitError) Code() int { return e.code }
