package cmd

import (
	"os"
	"path/filepath"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/output"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
)

// deleteResult holds the result of deleting a single service's build artifacts.
type deleteResult struct {
	Service string `json:"service"`
	Path    string `json:"path"`
	Deleted bool   `json:"deleted"`
}

func DeleteBuildsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete-builds [services...]",
		Aliases: []string{"clean"},
		Short:   "Delete builds for services",
		Long:    "Delete dist/ directories for one or more services, or all services if none specified.",
		RunE: withJSON(func(_ *cobra.Command, args []string) (any, error) {
			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.NoServices,
					errors.Wrap(err, "error finding services"))
			}

			if len(services) == 0 {
				output.Warning("No services found")
				return nil, nil
			}

			output.Info("Deleting builds for %d service(s)...", len(services))
			results, err := deleteBuildsForServices(services)
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.IO, errors.Wrap(err, "error deleting builds"))
			}

			output.Success("Builds deleted successfully.")
			return results, nil
		}),
	}

	return cmd
}

func deleteBuildsForServices(services []serviceinfo.ServiceInfo) ([]deleteResult, error) {
	var results []deleteResult
	for _, svc := range services {
		distPath := filepath.Join(svc.Path, "dist")
		if _, err := os.Stat(distPath); os.IsNotExist(err) {
			output.Dimmed("No dist/ found for: %s", svc.Name)
			results = append(results, deleteResult{
				Service: svc.Name,
				Path:    distPath,
				Deleted: false,
			})
			continue
		}
		output.Dimmed("Removing: %s", distPath)
		if err := os.RemoveAll(distPath); err != nil {
			return results, exitcodes.WithCode(exitcodes.IO, errors.Wrap(err, "error removing %s", distPath))
		}
		results = append(results, deleteResult{
			Service: svc.Name,
			Path:    distPath,
			Deleted: true,
		})
	}
	return results, nil
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DeleteBuildsCmd())
}
