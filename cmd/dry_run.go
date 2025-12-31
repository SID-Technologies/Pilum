package cmd

import (
	"github.com/spf13/cobra"
)

func DryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dry-run [services...]",
		Aliases: []string{"dr"},
		Short:   "Preview commands without executing",
		Long:    "Show what commands would be executed based on each service's recipe.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return bindFlagsForDeploymentCommands(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts := getDeploymentOptions()
			opts.DryRun = true // Always dry-run for this command

			return runPipeline(args, opts, "No services found")
		},
	}

	addCommandFlags(cmd, false) // No --dry-run flag needed

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DryRunCmd())
}
