package cmd

import (
	"fmt"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dry-run [services...]",
		Short: "Preview commands without executing",
		Long:  "Show what commands would be executed based on each service's recipe.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := bindFlagsForDeploymentCommands(cmd); err != nil {
				return errors.Wrap(err, "error binding flags for deployment commands: %v")
			}
			if err := viper.BindPFlag("recipe-path", cmd.Flags().Lookup("recipe-path")); err != nil {
				return errors.Wrap(err, "error binding recipe-path flag")
			}
			if err := viper.BindPFlag("registry", cmd.Flags().Lookup("registry")); err != nil {
				return errors.Wrap(err, "error binding registry flag")
			}
			if err := viper.BindPFlag("template-path", cmd.Flags().Lookup("template-path")); err != nil {
				return errors.Wrap(err, "error binding template-path flag")
			}
			if err := viper.BindPFlag("max-workers", cmd.Flags().Lookup("max-workers")); err != nil {
				return errors.Wrap(err, "error binding max-workers flag")
			}
			if err := viper.BindPFlag("only-tags", cmd.Flags().Lookup("only-tags")); err != nil {
				return errors.Wrap(err, "error binding only-tags flag")
			}
			if err := viper.BindPFlag("exclude-tags", cmd.Flags().Lookup("exclude-tags")); err != nil {
				return errors.Wrap(err, "error binding exclude-tags flag")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			tag := viper.GetString("tag")
			debug := viper.GetBool("debug")
			timeout := viper.GetInt("timeout")
			retries := viper.GetInt("retries")
			registry := viper.GetString("registry")
			templatePath := viper.GetString("template-path")
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
				fmt.Println("No services found")
				return nil
			}

			// Load recipes
			recipes, err := recepie.LoadRecipesFromDirectory(recipePath)
			if err != nil {
				return errors.Wrap(err, "error loading recipes")
			}

			if len(recipes) == 0 {
				fmt.Println("No recipes found")
				return nil
			}

			// Create and run the orchestrator with DryRun enabled
			runner := orchestrator.NewRunner(services, recipes, orchestrator.RunnerOptions{
				Tag:          tag,
				Registry:     registry,
				TemplatePath: templatePath,
				Debug:        debug,
				Timeout:      timeout,
				Retries:      retries,
				DryRun:       true, // Always dry-run for this command
				MaxWorkers:   maxWorkers,
				OnlyTags:     onlyTags,
				ExcludeTags:  excludeTags,
			})

			return runner.Run()
		},
	}

	cmdFlagStrings(cmd)
	cmd.Flags().String("registry", "", "Docker registry prefix")
	cmd.Flags().String("template-path", "./_templates", "Path to Dockerfile templates")
	cmd.Flags().String("recipe-path", "./recepies", "Path to recipe definitions")
	cmd.Flags().Int("max-workers", 0, "Maximum parallel workers (0 = auto)")
	cmd.Flags().String("only-tags", "", "Only run steps with these tags (comma-separated)")
	cmd.Flags().String("exclude-tags", "", "Exclude steps with these tags (comma-separated)")

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DryRunCmd())
}
