package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/baseline"
	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/share"
	"github.com/harshdevelops/snagify/internal/store"
)

func newShareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "share",
		Short: "Securely share a baseline over the local network",
	}
	cmd.AddCommand(newShareBaselineCmd())
	cmd.AddCommand(newShareCurrentCmd())
	return cmd
}

func newShareBaselineCmd() *cobra.Command {
	var (
		file         string
		host         string
		port         int
		ttl          time.Duration
		maxDownloads int
		unsafeHTTP   bool
	)

	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Start a temporary TLS server that serves a baseline file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("read %s: %v", file, err)}
			}

			srv, err := share.StartServer(data, share.ServerOptions{
				Host:         host,
				Port:         port,
				TTL:          ttl,
				MaxDownloads: maxDownloads,
				UnsafeHTTP:   unsafeHTTP,
			})
			if err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}

			w := cmd.OutOrStdout()
			fmt.Fprintln(w, "Snagify baseline sharing started")
			fmt.Fprintf(w, "Expires in: %s\n", srv.TTL)
			fmt.Fprintf(w, "Downloads allowed: %d\n", srv.MaxDownloads)
			fmt.Fprintln(w)
			if unsafeHTTP {
				fmt.Fprintln(w, "WARNING: serving over plain HTTP (--unsafe-http). Traffic is not encrypted and not pinned.")
				fmt.Fprintln(w, "Ask teammates on the same LAN/VPN to run:")
				fmt.Fprintf(w, "  snagify check --from %s --unsafe-http\n", srv.URL)
			} else {
				fmt.Fprintln(w, "Ask teammates on the same LAN/VPN to run:")
				fmt.Fprintf(w, "  snagify check --from %s \\\n    --pin %s\n", srv.URL, srv.Pin)
			}
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Press Ctrl-C to stop early.")

			// Stop cleanly on Ctrl-C as well as TTL/max-downloads.
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sig
				srv.Close()
			}()

			srv.Wait()
			fmt.Fprintf(w, "Sharing ended (%d download(s) served).\n", srv.Downloads())
			return nil
		},
	}

	cmd.Flags().StringVar(&file, "file", ".snagify/baseline.json", "baseline file to serve")
	cmd.Flags().StringVar(&host, "host", "0.0.0.0", "bind host")
	cmd.Flags().IntVar(&port, "port", 0, "bind port (0 = random free port)")
	cmd.Flags().DurationVar(&ttl, "ttl", 10*time.Minute, "how long sharing stays open")
	cmd.Flags().IntVar(&maxDownloads, "max-downloads", 20, "shut down after this many downloads")
	cmd.Flags().BoolVar(&unsafeHTTP, "unsafe-http", false, "serve plain HTTP instead of TLS (insecure)")
	return cmd
}

func newShareCurrentCmd() *cobra.Command {
	var (
		repo         string
		lanMode      bool
		openMode     bool
		codeMode     bool
		host         string
		port         int
		ttl          time.Duration
		maxDownloads int
		unsafeHTTP   bool
	)

	cmd := &cobra.Command{
		Use:   "current",
		Short: "Capture and share the current machine setup over the local network",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("capture failed: %v", err)}
			}

			sanitized := baseline.Sanitize(snap)
			data, err := store.MarshalJSON(sanitized)
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("serialize baseline: %v", err)}
			}

			if repo == "" {
				repo = sanitized.Project.Name
			}
			if repo == "" {
				repo = "snagify-share"
			}

			// Determine pairing code.
			var pairingCode string
			if !openMode { // default to code mode
				pc, err := share.PairingCode()
				if err != nil {
					return &exitError{code: 2, msg: err.Error()}
				}
				pairingCode = pc
			}

			mode := "code"
			if openMode {
				mode = "open"
			}

			srv, err := share.StartServer(data, share.ServerOptions{
				Host:         host,
				Port:         port,
				TTL:          ttl,
				MaxDownloads: maxDownloads,
				UnsafeHTTP:   unsafeHTTP,
				PairingCode:  pairingCode,
			})
			if err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}

			w := cmd.OutOrStdout()
			fmt.Fprintln(w, "Snagify LAN share started")
			fmt.Fprintf(w, "Repo:           %s\n", repo)
			fmt.Fprintf(w, "Mode:           %s\n", mode)
			fmt.Fprintf(w, "Expires in:     %s\n", srv.TTL)
			fmt.Fprintf(w, "Max downloads:  %d\n", srv.MaxDownloads)

			if openMode {
				fmt.Fprintln(w)
				fmt.Fprintln(w, "⚠️  Open LAN mode: anyone on this network can download this sanitized baseline until it expires.")
			}

			var advertiser *lanAdvertiser
			if lanMode {
				fingerprint := ""
				if srv.Fingerprint != "" {
					fingerprint = share.PinString(srv.Fingerprint)
				}
				_, serverPort := parseHostPort(srv.URL)
				adv, advErr := advertiseLAN(repo, snap.Project.Name, version, mode, fingerprint, serverPort, ttl)
				if advErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "mDNS advertise: %v (LAN discovery unavailable, use URL+pin)\n", advErr)
				} else {
					advertiser = adv
				}

				fmt.Fprintln(w)
				fmt.Fprintln(w, "Teammates on the same LAN/VPN can run:")
				if openMode {
					fmt.Fprintf(w, "  snagify compare --lan %s\n", repo)
				} else {
					fmt.Fprintf(w, "  snagify compare --lan %s --code %s\n", repo, pairingCode)
				}
			} else {
				fmt.Fprintln(w)
				fmt.Fprintln(w, "Teammates can run:")
				if unsafeHTTP {
					fmt.Fprintf(w, "  snagify check --from %s --unsafe-http\n", srv.URL)
				} else {
					fmt.Fprintf(w, "  snagify check --from %s \\\n    --pin %s\n", srv.URL, srv.Pin)
				}
			}
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Press Ctrl-C to stop early.")

			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sig
				srv.Close()
			}()

			srv.Wait()
			if advertiser != nil {
				advertiser.stop()
			}
			fmt.Fprintf(w, "Sharing ended (%d download(s) served).\n", srv.Downloads())
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "repo/project name for LAN discovery (default: project name)")
	cmd.Flags().BoolVar(&lanMode, "lan", false, "advertise on LAN/VPN via mDNS")
	cmd.Flags().BoolVar(&openMode, "open", false, "open LAN mode (no pairing code; anyone on LAN can download)")
	cmd.Flags().BoolVar(&codeMode, "code", false, "pairing-code mode (default when --lan is set)")
	cmd.Flags().StringVar(&host, "host", "0.0.0.0", "bind host")
	cmd.Flags().IntVar(&port, "port", 0, "bind port (0 = random)")
	cmd.Flags().DurationVar(&ttl, "ttl", 15*time.Minute, "sharing TTL")
	cmd.Flags().IntVar(&maxDownloads, "max-downloads", 20, "shut down after N downloads")
	cmd.Flags().BoolVar(&unsafeHTTP, "unsafe-http", false, "serve plain HTTP (insecure)")
	_ = codeMode // alias for pairing-code mode; already the default
	return cmd
}
