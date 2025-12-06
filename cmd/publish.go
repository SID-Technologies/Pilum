package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

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

			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}
			if len(services) == 0 {
				output.Warning("No services found to publish")
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
	rootCmd.AddCommand(PublishCmd())
}
