package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/store"
)

func newSnapshotCmd() *cobra.Command {
	var quiet bool

	cmd := &cobra.Command{
		Use:   "snapshot [output.json]",
		Short: "Capture the current machine and project state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return fmt.Errorf("capture failed: %w", err)
			}

			out := defaultName(snap.Hostname)
			if len(args) == 1 {
				out = args[0]
			}

			if err := store.Save(out, snap); err != nil {
				return err
			}

			if !quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "Snapshot saved to %s\n", out)
				if flagVerbose {
					fmt.Fprintf(cmd.ErrOrStderr(), "project: %s (%s)\n", snap.Project.Name, snap.Project.Root)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")
	return cmd
}

// defaultName builds ./snagify-snapshot-<host>-<date>.json with a filesystem
// safe hostname.
func defaultName(host string) string {
	host = strings.NewReplacer("/", "-", " ", "-", ".", "-").Replace(host)
	if host == "" {
		host = "host"
	}
	date := time.Now().Format("20060102")
	return fmt.Sprintf("snagify-snapshot-%s-%s.json", host, date)
}
