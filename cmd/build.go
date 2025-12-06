package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
			tag := viper.GetString("tag")
			debug := viper.GetBool("debug")
			timeout := viper.GetInt("timeout")
			retries := viper.GetInt("retries")
			dryRun := viper.GetBool("dry-run")
			recipePath := viper.GetString("recipe-path")
			maxWorkers := viper.GetInt("max-workers")
			onlyTags := parseCommaSeparated(viper.GetString("only-tags"))
			excludeTags := parseCommaSeparated(viper.GetString("exclude-tags"))

			// Default to "build" tag if no tags specified
			if len(onlyTags) == 0 {
				onlyTags = []string{"build"}
			}

			// Find services
			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}

			if len(services) == 0 {
				output.Warning("No services found to build")
				return nil
			}

			// Load recipes
			recipes, err := recepie.LoadRecipesFromDirectory(recipePath)
			if err != nil {
				return errors.Wrap(err, "error loading recipes")
			}

			if len(recipes) == 0 {
				output.Warning("No recipes found")
				return nil
			}

			// Create and run the orchestrator
			runner := orchestrator.NewRunner(services, recipes, orchestrator.RunnerOptions{
				Tag:         tag,
				Debug:       debug,
				Timeout:     timeout,
				Retries:     retries,
				DryRun:      dryRun,
				MaxWorkers:  maxWorkers,
				OnlyTags:    onlyTags,
				ExcludeTags: excludeTags,
			})

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
