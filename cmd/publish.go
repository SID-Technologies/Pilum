package cmd

import (
	"log"

	"github.com/sid-technologies/centurion/lib/errors"
	serviceinfo "github.com/sid-technologies/centurion/lib/service_info"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func PublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish [services...]",
		Short: "Publish services (build, push, publish)",
		Long:  "Publish one or more services or all services if none specified. This command will build, push and publish the services to the specified environment.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			err := bindFlagsForDeploymentCommands(cmd)
			if err != nil {
				return errors.Wrap(err, "error binding flags for deployment commands: %v")
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			tag := viper.GetString("tag")
			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services: %v", err)
			}
			for _, service := range services {
				// Logic here for each service
				log.Printf("  Deploying service %s %s\n", service.Name, tag)
			}

			return nil
		},
	}

	cmdFlagStrings(cmd)

	return cmd
}

//nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(PublishCmd())
}
