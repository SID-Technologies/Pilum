package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
)

func BuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "build [services...]",
		Aliases: []string{"b", "make"},
		Short:   "Build services",
		Long:    "Build one or more services or all services if none specified. Runs all recipe steps tagged with 'build'.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return bindFlagsForDeploymentCommands(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts := getDeploymentOptions()

			// Default to "build" tag if no tags specified
			if len(opts.OnlyTags) == 0 {
				opts.OnlyTags = []string{"build"}
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
				output.Warning("No services found to build")
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
	rootCmd.AddCommand(BuildCmd())
}
