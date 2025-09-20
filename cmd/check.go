package cmd

import (
	"log"

	"github.com/sid-technologies/centurion/lib/errors"
	serviceinfo "github.com/sid-technologies/centurion/lib/service_info"
	"github.com/spf13/cobra"
)

func CheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check servvice.yaml",
		Short: "Check the configuration of the services",
		Long:  "Check the configuration of the services",
		RunE: func(_ *cobra.Command, _ []string) error {
			file := ""
			log.Printf("Checking configuration of the services in %s\n", file)
			services, err := serviceinfo.FindServices(".")
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}
			for _, service := range services {
				// Logic here for each service
				log.Printf("  Checking service %s\n", service.Name)
				err := service.Validate()
				if err != nil {
					return errors.Wrap(err, "error checking service %s", service.Name)
				}
			}
			log.Println("All services are valid")

			return nil
		},
	}

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(CheckCmd())
}
