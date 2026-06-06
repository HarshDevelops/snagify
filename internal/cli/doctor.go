package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/check"
	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/fixplan"
)

func newDoctorCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Detect likely blockers and suggest safe and guided fixes",
		Long: `doctor runs the same checks as snagify check and then produces a fix plan.
It never modifies files. Use snagify fix --safe to apply safe fixes.`,
		Args: cobra.NoArgs,
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

			examplePath := cfg.Env.ExampleFile
			if examplePath != "" && !filepath.IsAbs(examplePath) {
				examplePath = filepath.Join(snap.Project.Root, examplePath)
			}
			actualPath := cfg.Env.ActualFile
			if actualPath == "" {
				actualPath = ".env"
			}
			if !filepath.IsAbs(actualPath) {
				actualPath = filepath.Join(snap.Project.Root, actualPath)
			}

			plan := fixplan.Build(rep, snap.Project.Root, examplePath, actualPath)

			if err := fixplan.Render(cmd.OutOrStdout(), plan, format); err != nil {
				return &exitError{code: 2, msg: err.Error()}
			}
			if rep.Failed() {
				return &exitError{code: 1}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "terminal", "output format: terminal or json")
	return cmd
}
