package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/baseline"
	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/store"
)

func newBaselineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Create and manage sanitized known-good baselines",
	}
	cmd.AddCommand(newBaselineCreateCmd())
	return cmd
}

func newBaselineCreateCmd() *cobra.Command {
	var out string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Capture the current machine as a sanitized, shareable baseline",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("capture failed: %v", err)}
			}

			sanitized := baseline.Sanitize(snap)

			if dir := filepath.Dir(out); dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return &exitError{code: 2, msg: fmt.Sprintf("create %s: %v", dir, err)}
				}
			}
			if err := store.Save(out, sanitized); err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Baseline saved to %s\n", out)
			fmt.Fprintln(cmd.OutOrStdout(), "Hostname and absolute paths were removed. Safe to commit or share.")
			return nil
		},
	}

	cmd.Flags().StringVar(&out, "out", ".snagify/baseline.json", "output path for the baseline JSON")
	return cmd
}
