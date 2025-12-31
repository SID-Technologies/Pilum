package cmd

import (
	"github.com/spf13/cobra"
)

func BuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "build [services...]",
		Aliases: []string{"b", "make"},
		Short:   "Build services",
		Long:    "Build one or more services or all services if none specified. Runs all recipe steps tagged with 'build'.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return bindFlagsForDeploymentCommands(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts := getDeploymentOptions()

			// Default to "build" tag if no tags specified
			if len(opts.OnlyTags) == 0 {
				opts.OnlyTags = []string{"build"}
			}

			return runPipeline(args, opts, "No services found to build")
		},
	}

	addCommandFlags(cmd, true)

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(BuildCmd())
}
