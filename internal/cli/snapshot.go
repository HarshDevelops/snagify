package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/check"
	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/probe"
	"github.com/harshdevelops/snagify/internal/store"
)

func newSnapshotCmd() *cobra.Command {
	var quiet bool
	var probes bool

	cmd := &cobra.Command{
		Use:   "snapshot [output.json]",
		Short: "Capture the current machine and project state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return fmt.Errorf("capture failed: %w", err)
			}

			// Active probes are opt-in and require a config with probe sections.
			if probes {
				if path, ok := check.FindConfig("", snap.Project.Root); ok {
					if cfg, cfgErr := config.Load(path); cfgErr == nil && cfg.HasActiveProbes() {
						results := probe.Run(cfg, probe.Options{})
						snap.Probes = &results
					}
				} else if flagVerbose {
					fmt.Fprintln(cmd.ErrOrStderr(), "--probes set but no .snagify.yaml with probe sections found")
				}
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
	cmd.Flags().BoolVar(&probes, "probes", false, "also run active probes declared in .snagify.yaml")
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
