package cmd

import (
	"github.com/spf13/cobra"
)

func DeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy [services...]",
		Aliases: []string{"up"},
		Short:   "Deploy services (build, publish, push, deploy)",
		Long:    "Deploy one or more services or all services if none specified. This command will build, publish, push and deploy the services to the specified environment.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return bindFlagsForDeploymentCommands(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts := getDeploymentOptions()
			return runPipeline(args, opts, "No services found to deploy")
		},
	}

	addCommandFlags(cmd, true)

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DeployCmd())
}
