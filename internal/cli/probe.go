package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/check"
	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/probe"
	"github.com/harshdevelops/snagify/internal/report"
)

func newProbeCmd() *cobra.Command {
	var (
		format     string
		configPath string
		pf         probeFlags
	)

	cmd := &cobra.Command{
		Use:   "probe",
		Short: "Run only the active connectivity probes (services, network, TLS) from .snagify.yaml",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("capture failed: %v", err)}
			}

			path, exists := check.FindConfig(configPath, snap.Project.Root)
			if !exists {
				return &exitError{code: 2, msg: fmt.Sprintf(
					"no %s found in %s", config.FileName, snap.Project.Root)}
			}
			cfg, err := config.Load(path)
			if err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}

			if !cfg.HasActiveProbes() {
				return &exitError{code: 2, msg: "no services/network/tls probes declared in config"}
			}

			if pf.insecureProbe {
				fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: --insecure-probe disables TLS verification for probes. Results are not trustworthy.")
			}

			results := probe.Run(cfg, probe.Options{
				DefaultTimeout: pf.timeout,
				SkipNetwork:    pf.noNetwork,
				SkipTLS:        pf.noTLS,
				InsecureProbe:  pf.insecureProbe,
			})

			// Build a probe-only report by checking just the probe results.
			snap.Probes = &results
			rep := check.ProbeReport(snap, cfg)

			if err := report.Render(cmd.OutOrStdout(), rep, format); err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}
			if rep.Failed() {
				return &exitError{code: 1}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "terminal", "output format: terminal, markdown, or json")
	cmd.Flags().StringVar(&configPath, "config", "", "path to .snagify.yaml (default: project root)")
	addProbeFlags(cmd, &pf)
	return cmd
}
