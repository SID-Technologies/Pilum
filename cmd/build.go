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

func BuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [services...]",
		Short: "Build services",
		Long:  "Build one or more services or all services if none specified. Runs all recipe steps that don't have the 'deploy' tag.",
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
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			tag := viper.GetString("tag")
			debug := viper.GetBool("debug")
			timeout := viper.GetInt("timeout")
			retries := viper.GetInt("retries")
			dryRun := viper.GetBool("dry-run")
			registry := viper.GetString("registry")
			templatePath := viper.GetString("template-path")
			recipePath := viper.GetString("recipe-path")
			maxWorkers := viper.GetInt("max-workers")

			// Find services
			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services: %v", err.Error())
			}

			if len(services) == 0 {
				fmt.Println("No services found to build")
				return nil
			}

			// Load recipes
			recipes, err := recepie.LoadRecipesFromDirectory(recipePath)
			if err != nil {
				return errors.Wrap(err, "error loading recipes: %v", err.Error())
			}

			if len(recipes) == 0 {
				fmt.Println("No recipes found")
				return nil
			}

			// Create and run the orchestrator
			// Build excludes steps tagged with "deploy"
			runner := orchestrator.NewRunner(services, recipes, orchestrator.RunnerOptions{
				Tag:          tag,
				Registry:     registry,
				TemplatePath: templatePath,
				Debug:        debug,
				Timeout:      timeout,
				Retries:      retries,
				DryRun:       dryRun,
				MaxWorkers:   maxWorkers,
				ExcludeTags:  []string{"deploy"},
			})

			return runner.Run()
		},
	}

	cmdFlagStrings(cmd)
	cmd.Flags().String("registry", "", "Docker registry prefix (overrides service.yaml)")
	cmd.Flags().String("template-path", "./_templates", "Path to Dockerfile templates")
	cmd.Flags().String("recipe-path", "./recepies", "Path to recipe definitions")
	cmd.Flags().Int("max-workers", 0, "Maximum parallel workers (0 = auto)")

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(BuildCmd())
}
