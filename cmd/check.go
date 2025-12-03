package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
)

func CheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check servvice.yaml",
		Short: "Check the configuration of the services",
		Long:  "Check the configuration of the services",
		RunE: func(_ *cobra.Command, _ []string) error {
			output.Info("Checking configuration of the services")
			services, err := serviceinfo.FindServices(".")
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}
			for _, service := range services {
				output.Dimmed("  Checking service %s", service.Name)
				err := service.Validate()
				if err != nil {
					return errors.Wrap(err, "error checking service %s", service.Name)
				}
			}
			output.Success("All services are valid")

			return nil
		},
	}

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(CheckCmd())
}
