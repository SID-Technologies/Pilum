package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/providers"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/suggest"

	"github.com/spf13/cobra"
)

func CheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "check [services...]",
		Aliases: []string{"validate"},
		Short:   "Check the configuration of the services",
		Long:    "Check the configuration of the services against their recipe requirements. Optionally specify service names to check only those services.",
		RunE: func(_ *cobra.Command, args []string) error {
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
			recipes, err := recepie.LoadEmbeddedRecipes()
			if err != nil {
				return errors.Wrap(err, "error loading recipes")
			}

			if len(recipes) == 0 {
				output.Warning("No recipes found")
				return nil
			}

			// Index recipes by provider-service key (e.g., "gcp-cloud-run")
			recipeMap := make(map[string]*recepie.Recipe)
			for i := range recipes {
				key := recipes[i].Provider
				if recipes[i].Service != "" {
					key = recipes[i].Provider + "-" + recipes[i].Service
				}
				recipeMap[key] = &recipes[i].Recipe
			}

			// Validate each service
			for _, service := range services {
				recipeKey := service.RecipeKey()
				output.Dimmed("  Checking service %s (recipe: %s)", service.Name, recipeKey)

				// Base validation
				if err := service.Validate(); err != nil {
					return errors.Wrap(err, "error checking service %s", service.Name)
				}

				// Recipe-specific validation
				recipe, exists := recipeMap[recipeKey]
				if !exists {
					// Use providers registry for suggestions
					suggestion := suggest.FormatSuggestion(recipeKey, providers.GetAllRecipeKeys())
					if suggestion != "" {
						output.Warning("    No recipe found for '%s' - %s", recipeKey, suggestion)
					} else {
						output.Warning("    No recipe found for '%s'", recipeKey)
					}
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

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(CheckCmd())
}
