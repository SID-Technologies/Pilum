package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/sid-technologies/centurion/lib/errors"
	"github.com/sid-technologies/centurion/lib/recepie"
	serviceinfo "github.com/sid-technologies/centurion/lib/service_info"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dry-run [services...]",
		Short: "Perform a dry run of one or more services",
		Long:  "Preform a dry run one or more services or all services if none specified for build, publish, push and deploy operations",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			err := viper.BindPFlag("tag", cmd.Flags().Lookup("tag"))
			if err != nil {
				return errors.Wrap(err, "error binding tag flag: %v", err)
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			tag := viper.GetString("tag")
			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}
			// load all recepies
			_, err = recepie.LoadRecipesFromDirectory("./recepies")
			if err != nil {
				return errors.Wrap(err, "error loading recipes")
			}

			log.Info().Msgf("Dry Run Executing")
			for _, service := range services {
				// Logic here for each service
				log.Info().Msgf("Dry run for service %s with tag %s\n", service.Name, tag)
			}

			return nil
		},
	}

	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")

	return cmd
}

//nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DryRunCmd())
}
