// Package cli wires the cobra command tree for snagify.
package cli

import (
	"github.com/spf13/cobra"
)

// Flags shared across commands.
var (
	flagProjectRoot string
	flagVerbose     bool
)

// NewRootCmd builds the root command with all subcommands attached.
func NewRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "snagify",
		Short: "Snag the differences. Kill 'works on my machine'.",
		Long: "Snagify captures environment and project configuration snapshots\n" +
			"and produces clear, ranked diffs highlighting setup blockers.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagProjectRoot, "project-root", "", "override project auto-detection")
	root.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output to stderr")

	root.AddCommand(newSnapshotCmd())
	root.AddCommand(newDiffCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newProbeCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newBaselineCmd())
	root.AddCommand(newDiffManyCmd())
	root.AddCommand(newShareCmd())
	return root
}
