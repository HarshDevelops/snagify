package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/check"
	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/fixplan"
)

func newFixCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Apply or suggest fixes for detected blockers",
	}
	cmd.AddCommand(newFixSafeCmd())
	cmd.AddCommand(newFixGuidedCmd())
	return cmd
}

func newFixSafeCmd() *cobra.Command {
	var (
		dryRun bool
		yes    bool
		force  bool
		format string
	)

	cmd := &cobra.Command{
		Use:   "safe",
		Short: "Apply only safe, reversible fixes (create .env, append missing keys, pull Docker images)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Require explicit consent in non-interactive mode.
			fi, _ := os.Stdin.Stat()
			isInteractive := (fi.Mode() & os.ModeCharDevice) != 0
			if !isInteractive && !yes && !dryRun {
				return &exitError{code: 2, msg: "non-interactive mode: pass --yes to apply fixes, or --dry-run to preview"}
			}

			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("capture failed: %v", err)}
			}

			cfgPath, exists := check.FindConfig("", snap.Project.Root)
			if !exists {
				return &exitError{code: 2, msg: fmt.Sprintf(
					"no %s found in %s (run `snagify init` to create one)", config.FileName, snap.Project.Root)}
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}

			rep := check.Run(snap, cfg, snap.Project.Root, check.Options{})

			examplePath := resolveEnvPath(cfg.Env.ExampleFile, snap.Project.Root)
			actualPath := resolveEnvPath(cfg.Env.ActualFile, snap.Project.Root)
			if actualPath == snap.Project.Root { // ActualFile was empty
				actualPath = filepath.Join(snap.Project.Root, ".env")
			}

			plan := fixplan.Build(rep, snap.Project.Root, examplePath, actualPath)

			w := cmd.OutOrStdout()
			if dryRun {
				fmt.Fprintln(w, "Dry-run mode: no changes will be made.")
			}

			result := fixplan.ApplySafe(plan, fixplan.ApplyOptions{
				DryRun:      dryRun,
				Yes:         yes,
				Force:       force,
				Stdout:      w,
				ExamplePath: examplePath,
				ActualPath:  actualPath,
				ProjectRoot: snap.Project.Root,
			})

			fmt.Fprintln(w)
			fixplan.RenderResult(w, result)

			if len(result.Errors) > 0 {
				return &exitError{code: 2, msg: "some fixes failed"}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview fixes without applying them")
	cmd.Flags().BoolVar(&yes, "yes", false, "apply without confirmation prompts")
	cmd.Flags().BoolVar(&force, "force", false, "force overwrite of existing files")
	cmd.Flags().StringVar(&format, "format", "terminal", "output format")
	return cmd
}

func newFixGuidedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guided",
		Short: "Print version-manager commands to fix runtime mismatches (does not execute anything)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, err := capture.Capture(capture.Options{ProjectRoot: flagProjectRoot})
			if err != nil {
				return &exitError{code: 2, msg: fmt.Sprintf("capture failed: %v", err)}
			}

			cfgPath, exists := check.FindConfig("", snap.Project.Root)
			if !exists {
				return &exitError{code: 2, msg: fmt.Sprintf(
					"no %s found in %s (run `snagify init` to create one)", config.FileName, snap.Project.Root)}
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}

			rep := check.Run(snap, cfg, snap.Project.Root, check.Options{})
			plan := fixplan.Build(rep, snap.Project.Root, "", "")

			w := cmd.OutOrStdout()
			fmt.Fprintln(w, "Guided fix suggestions (nothing is executed):")
			fmt.Fprintln(w)
			if len(plan.Guided) == 0 {
				fmt.Fprintln(w, "  No guided fixes available.")
				return nil
			}
			for _, a := range plan.Guided {
				fmt.Fprintf(w, "  %s\n", a.Title)
				for _, c := range a.Commands {
					fmt.Fprintf(w, "    $ %s\n", c)
				}
				fmt.Fprintln(w)
			}
			return nil
		},
	}

	return cmd
}

func resolveEnvPath(path, projectRoot string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(projectRoot, path)
}
