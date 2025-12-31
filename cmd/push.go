package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

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

			filterOpts := serviceinfo.FilterOptions{
				Names:       args,
				OnlyChanged: opts.OnlyChanged,
				Since:       opts.Since,
			}
			services, err := serviceinfo.FindAndFilterServicesWithOptions(".", filterOpts)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}
			if len(services) == 0 {
				output.Warning("No services found to push")
				return nil
			}

			recipes, err := recepie.LoadRecipesFromDirectory(opts.RecipePath)
			if err != nil {
				return errors.Wrap(err, "error loading recipes")
			}
			if len(recipes) == 0 {
				output.Warning("No recipes found")
				return nil
			}

			runner := orchestrator.NewRunner(services, recipes, opts.toRunnerOptions())
			return runner.Run()
		},
	}

	cmdFlagStrings(cmd)

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(PushCmd())
}
