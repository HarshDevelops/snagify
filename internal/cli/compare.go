package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/baseline"
	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/lan"
	"github.com/harshdevelops/snagify/internal/model"
	"github.com/harshdevelops/snagify/internal/report"
	"github.com/harshdevelops/snagify/internal/share"
)

func newCompareCmd() *cobra.Command {
	var (
		lanMode      bool
		code         string
		discTimeout  time.Duration
		format       string
		saveBaseline string
	)

	cmd := &cobra.Command{
		Use:   "compare [repo-name]",
		Short: "Compare this machine against a LAN-shared baseline",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoFilter := ""
			if len(args) == 1 {
				repoFilter = args[0]
			}

			if !lanMode {
				return &exitError{code: 2, msg: "currently --lan is required for compare. Use `snagify check --against <file>` for file-based comparison"}
			}

			// Discover LAN shares.
			ctx := context.Background()
			shares, err := lan.Discover(ctx, repoFilter, discTimeout)
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("LAN discovery failed: %v", err)}
			}

			// Filter out expired shares.
			var active []lan.DiscoveredShare
			for _, s := range shares {
				if !s.IsExpired() {
					active = append(active, s)
				}
			}

			if len(active) == 0 {
				return &exitError{code: 2, msg: `No Snagify LAN baselines found.
Make sure the known-good machine is running ` + "`snagify share current --lan`" + ` on the same LAN/VPN.`}
			}

			var chosen lan.DiscoveredShare
			if len(active) == 1 {
				chosen = active[0]
			} else {
				if repoFilter == "" {
					msg := "Multiple LAN shares found. Pass a repo name:\n"
					for _, s := range active {
						msg += fmt.Sprintf("  snagify compare --lan %s\n", s.Repo)
					}
					return &exitError{code: 2, msg: msg}
				}
				// repoFilter already applied in Discover; use first remaining.
				chosen = active[0]
			}

			// Pairing code check.
			if chosen.Mode == "code" && code == "" {
				return &exitError{code: 2, msg: `This share requires a pairing code.
Re-run with --code <code>:
  snagify compare --lan ` + chosen.Repo + ` --code <code>`}
			}

			// Discover the baseline URL. The server serves /baseline/<token>;
			// for LAN discovery mode the compare client needs to request the
			// token from a discovery endpoint. For simplicity we expose a
			// /resolve endpoint on the server that returns the token URL when
			// the pairing code is correct.
			baseURL := fmt.Sprintf("https://%s:%d", chosen.Host, chosen.Port)
			resolvedURL, err := resolveBaselineURL(baseURL, chosen.Fingerprint, code, 10*time.Second)
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("resolve baseline URL: %v", err)}
			}

			data, err := share.Fetch(share.FetchOptions{
				URL:         resolvedURL,
				Pin:         chosen.Fingerprint,
				Timeout:     15 * time.Second,
				PairingCode: code,
			})
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("fetch baseline: %v", err)}
			}

			var base model.Snapshot
			if err := json.Unmarshal(data, &base); err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("parse baseline: %v", err)}
			}

			// Capture local snapshot — stays on this machine, never sent.
			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("capture failed: %v", err)}
			}

			rep := baseline.Compare(base, snap)

			if saveBaseline != "" {
				if err := writeFile(saveBaseline, data); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "save baseline: %v\n", err)
				}
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

	cmd.Flags().BoolVar(&lanMode, "lan", false, "discover baseline on LAN/VPN via mDNS")
	cmd.Flags().StringVar(&code, "code", "", "pairing code for code-mode shares")
	cmd.Flags().DurationVar(&discTimeout, "timeout", 5*time.Second, "mDNS discovery timeout")
	cmd.Flags().StringVar(&format, "format", "terminal", "output format: terminal, markdown, or json")
	cmd.Flags().StringVar(&saveBaseline, "save-baseline", "", "save fetched baseline to this path")
	return cmd
}
