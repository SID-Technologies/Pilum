package cmd

import (
	"github.com/spf13/cobra"
)

func PushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "push [services...]",
		Aliases: []string{"ps"},
		Short:   "Push Docker images to registry",
		Long:    "Push Docker images for one or more services to the container registry. Runs recipe steps tagged with 'push'.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return bindFlagsForDeploymentCommands(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts := getDeploymentOptions()

			// Default to "push" tag if no tags specified
			if len(opts.OnlyTags) == 0 {
				opts.OnlyTags = []string{"push"}
			}

			return runPipeline(args, opts, "No services found to push")
		},
	}

	addCommandFlags(cmd, true)

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(PushCmd())
}
