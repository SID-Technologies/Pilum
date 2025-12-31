package cmd

import (
	"github.com/spf13/cobra"
)

func PublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "publish [services...]",
		Aliases: []string{"p"},
		Short:   "Publish services (build, docker build, push)",
		Long:    "Publish one or more services or all services if none specified. This command will build, create Docker images, and push them to the registry.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return bindFlagsForDeploymentCommands(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts := getDeploymentOptions()

			// Default to excluding "deploy" tag if no tags specified
			if len(opts.ExcludeTags) == 0 && len(opts.OnlyTags) == 0 {
				opts.ExcludeTags = []string{"deploy"}
			}

			return runPipeline(args, opts, "No services found to publish")
		},
	}

	addCommandFlags(cmd, true)

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(PublishCmd())
}
