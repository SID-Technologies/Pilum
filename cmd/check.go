package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func CheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "check [services...]",
		Aliases: []string{"validate"},
		Short:   "Check the configuration of the services",
		Long:    "Check the configuration of the services against their recipe requirements. Optionally specify service names to check only those services.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := viper.BindPFlag("recipe-path", cmd.Flags().Lookup("recipe-path")); err != nil {
				return errors.Wrap(err, "error binding recipe-path flag")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			recipePath := viper.GetString("recipe-path")

			output.Info("Checking configuration of the services")

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

			// Index recipes by provider
			recipeMap := make(map[string]*recepie.Recipe)
			for i := range recipes {
				recipeMap[recipes[i].Provider] = &recipes[i].Recipe
			}

			// Validate each service
			for _, service := range services {
				output.Dimmed("  Checking service %s (provider: %s)", service.Name, service.Provider)

				// Base validation
				if err := service.Validate(); err != nil {
					return errors.Wrap(err, "error checking service %s", service.Name)
				}

				// Recipe-specific validation
				recipe, exists := recipeMap[service.Provider]
				if !exists {
					output.Warning("    No recipe found for provider '%s'", service.Provider)
					continue
				}

				if err := recipe.ValidateService(&service); err != nil {
					return errors.Wrap(err, "error checking service %s", service.Name)
				}

				output.Success("    %s: valid", service.Name)
			}

			output.Success("All services are valid")
			return nil
		},
	}

	cmd.Flags().String("recipe-path", "./recepies", "Path to recipe definitions")

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(CheckCmd())
}
