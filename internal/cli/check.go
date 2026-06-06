package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/baseline"
	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/check"
	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
	"github.com/harshdevelops/snagify/internal/probe"
	"github.com/harshdevelops/snagify/internal/report"
	"github.com/harshdevelops/snagify/internal/share"
	"github.com/harshdevelops/snagify/internal/store"
)

// probeFlags bundles the active-probe control flags shared by check and probe.
type probeFlags struct {
	timeout       time.Duration
	noNetwork     bool
	noTLS         bool
	noDocker      bool
	insecureProbe bool
}

func newCheckCmd() *cobra.Command {
	var (
		format       string
		configPath   string
		against      string
		from         string
		pin          string
		saveBaseline string
		unsafeHTTP   bool
		pf           probeFlags
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check this machine against a repo config (.snagify.yaml) or a baseline",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("capture failed: %v", err)}
			}

			var rep report.Report
			switch {
			case from != "":
				rep, err = checkFromPeer(snap, from, pin, saveBaseline, unsafeHTTP)
			case against != "":
				rep, err = checkAgainstBaseline(snap, against)
			default:
				rep, err = checkAgainstConfig(cmd, snap, configPath, pf)
			}
			if err != nil {
				return err
			}

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
	cmd.Flags().StringVar(&against, "against", "", "compare against a baseline JSON file")
	cmd.Flags().StringVar(&from, "from", "", "fetch a baseline from a sharing URL (requires --pin)")
	cmd.Flags().StringVar(&pin, "pin", "", "expected TLS certificate fingerprint for --from")
	cmd.Flags().StringVar(&saveBaseline, "save-baseline", "", "save a baseline fetched via --from to this path")
	cmd.Flags().BoolVar(&unsafeHTTP, "unsafe-http", false, "allow plain HTTP for --from (insecure)")
	addProbeFlags(cmd, &pf)
	return cmd
}

func addProbeFlags(cmd *cobra.Command, pf *probeFlags) {
	cmd.Flags().DurationVar(&pf.timeout, "timeout", 2*time.Second, "per-probe timeout")
	cmd.Flags().BoolVar(&pf.noNetwork, "no-network", false, "skip DNS and HTTP probes")
	cmd.Flags().BoolVar(&pf.noTLS, "no-tls", false, "skip TLS probes")
	cmd.Flags().BoolVar(&pf.noDocker, "no-docker", false, "skip Docker checks")
	cmd.Flags().BoolVar(&pf.insecureProbe, "insecure-probe", false, "allow TLS verification skip (INSECURE)")
}

func checkAgainstConfig(cmd *cobra.Command, snap model.Snapshot, configPath string, pf probeFlags) (report.Report, error) {
	path, exists := check.FindConfig(configPath, snap.Project.Root)
	if !exists {
		return report.Report{}, &exitError{code: 2, msg: fmt.Sprintf(
			"no %s found in %s (run `snagify init` to create one)", config.FileName, snap.Project.Root)}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return report.Report{}, &exitError{code: 2, msg: err.Error()}
	}

	if pf.noDocker {
		cfg.Docker = config.Docker{}
	}

	// Run active probes only if the config declares them.
	if cfg.HasActiveProbes() {
		if pf.insecureProbe {
			fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: --insecure-probe disables TLS verification for probes. Results are not trustworthy.")
		}
		results := probe.Run(cfg, probe.Options{
			DefaultTimeout: pf.timeout,
			SkipNetwork:    pf.noNetwork,
			SkipTLS:        pf.noTLS,
			InsecureProbe:  pf.insecureProbe,
		})
		snap.Probes = &results
	}

	return check.Run(snap, cfg, snap.Project.Root, check.Options{}), nil
}

func checkAgainstBaseline(snap model.Snapshot, path string) (report.Report, error) {
	base, err := store.Load(path)
	if err != nil {
		return report.Report{}, &exitError{code: 2, msg: err.Error()}
	}
	return baseline.Compare(base, snap), nil
}

func checkFromPeer(snap model.Snapshot, url, pin, savePath string, unsafeHTTP bool) (report.Report, error) {
	data, err := share.Fetch(share.FetchOptions{
		URL:        url,
		Pin:        pin,
		UnsafeHTTP: unsafeHTTP,
		Timeout:    15 * time.Second,
	})
	if err != nil {
		return report.Report{}, &exitError{code: 2, msg: fmt.Sprintf(
			"%v\nDirect sharing requires same LAN/VPN. Use baseline file sharing or future relay mode.", err)}
	}

	var base model.Snapshot
	if err := json.Unmarshal(data, &base); err != nil {
		return report.Report{}, &exitError{code: 2, msg: fmt.Sprintf("downloaded baseline is not valid JSON: %v", err)}
	}

	if savePath != "" {
		if err := os.MkdirAll(filepath.Dir(savePath), 0o755); err == nil {
			_ = os.WriteFile(savePath, data, 0o644)
		}
	}

	return baseline.Compare(base, snap), nil
}
