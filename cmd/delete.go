package cmd

import (
	"os"
	"path/filepath"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
)

func DeleteBuildsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete-builds [services...]",
		Aliases: []string{"clean"},
		Short:   "Delete builds for services",
		Long:    "Delete dist/ directories for one or more services, or all services if none specified.",
		RunE: func(_ *cobra.Command, args []string) error {
			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}

			if len(services) == 0 {
				output.Warning("No services found")
				return nil
			}

			output.Info("Deleting builds for %d service(s)...", len(services))
			if err := deleteBuildsForServices(services); err != nil {
				return errors.Wrap(err, "error deleting builds")
			}

			output.Success("Builds deleted successfully.")
			return nil
		},
	}

	return cmd
}

func deleteBuildsForServices(services []serviceinfo.ServiceInfo) error {
	for _, svc := range services {
		distPath := filepath.Join(svc.Path, "dist")
		if _, err := os.Stat(distPath); os.IsNotExist(err) {
			output.Dimmed("No dist/ found for: %s", svc.Name)
			continue
		}
		output.Dimmed("Removing: %s", distPath)
		if err := os.RemoveAll(distPath); err != nil {
			return errors.Wrap(err, "error removing %s", distPath)
		}
	}
	return nil
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DeleteBuildsCmd())
}
