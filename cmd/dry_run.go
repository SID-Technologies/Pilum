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

func DryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dry-run [services...]",
		Aliases: []string{"dr"},
		Short:   "Preview commands without executing",
		Long:    "Show what commands would be executed based on each service's recipe.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return bindFlagsForDeploymentCommands(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			tag := viper.GetString("tag")
			debug := viper.GetBool("debug")
			timeout := viper.GetInt("timeout")
			retries := viper.GetInt("retries")
			recipePath := viper.GetString("recipe-path")
			maxWorkers := viper.GetInt("max-workers")
			onlyTags := parseCommaSeparated(viper.GetString("only-tags"))
			excludeTags := parseCommaSeparated(viper.GetString("exclude-tags"))

			// Find services
			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}

			if len(services) == 0 {
				output.Warning("No services found")
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

			// Create and run the orchestrator with DryRun enabled
			runner := orchestrator.NewRunner(services, recipes, orchestrator.RunnerOptions{
				Tag:         tag,
				Debug:       debug,
				Timeout:     timeout,
				Retries:     retries,
				DryRun:      true, // Always dry-run for this command
				MaxWorkers:  maxWorkers,
				OnlyTags:    onlyTags,
				ExcludeTags: excludeTags,
			})

			return runner.Run()
		},
	}

	cmdFlagStringsNoDryRun(cmd)

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DryRunCmd())
}
