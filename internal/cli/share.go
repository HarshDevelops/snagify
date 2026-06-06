package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/share"
)

func newShareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "share",
		Short: "Securely share a baseline over the local network",
	}
	cmd.AddCommand(newShareBaselineCmd())
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
