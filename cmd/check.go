package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
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
		RunE: withJSON(func(cmd *cobra.Command, args []string) (any, error) {
			output.Info("Checking configuration of the services")

			env, _ := cmd.Flags().GetString("env")

			// Find services
			filterOpts := serviceinfo.FilterOptions{
				Names:       args,
				NoGitIgnore: NoGitIgnore(),
				Env:         env,
			}
			services, err := serviceinfo.FindAndFilterServicesWithOptions(".", filterOpts)
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.NoServices, errors.Wrap(err, "error finding services"))
			}

			if len(services) == 0 {
				output.Warning("No services found")
				return nil, nil
			}

			// Load recipes
			recipes, err := recepie.LoadEmbeddedRecipes()
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.Config, errors.Wrap(err, "error loading recipes"))
			}

			if len(recipes) == 0 {
				output.Warning("No recipes found")
				return nil, nil
			}

			// Index recipes by composite key (e.g., "gcp-cloud-run")
			recipeMap := make(map[string]*recepie.Recipe)
			for i := range recipes {
				recipeMap[recipes[i].RecipeKey()] = &recipes[i].Recipe
			}

			// Validate each service, collecting results for JSON
			type checkResult struct {
				Service string `json:"service"`
				Recipe  string `json:"recipe"`
				Valid   bool   `json:"valid"`
				Error   string `json:"error,omitempty"`
			}
			var results []checkResult
			allValid := true
			var firstErr error

			for _, service := range services {
				recipeKey := service.RecipeKey()
				output.Dimmed("  Checking service %s (recipe: %s)", service.Name, recipeKey)

				// Base validation
				if err := service.Validate(); err != nil {
					allValid = false
					validationErr := exitcodes.WithCode(exitcodes.Config,
						errors.Wrap(err, "error checking service %s", service.Name))
					results = append(results, checkResult{
						Service: service.Name,
						Recipe:  recipeKey,
						Valid:   false,
						Error:   err.Error(),
					})
					if firstErr == nil {
						firstErr = validationErr
					}
					if !output.IsJSON() {
						return nil, validationErr
					}
					continue
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
					results = append(results, checkResult{
						Service: service.Name,
						Recipe:  recipeKey,
						Valid:   true,
					})
					continue
				}

				if err := recipe.ValidateService(&service); err != nil {
					allValid = false
					validationErr := exitcodes.WithCode(exitcodes.Config,
						errors.Wrap(err, "error checking service %s", service.Name))
					results = append(results, checkResult{
						Service: service.Name,
						Recipe:  recipeKey,
						Valid:   false,
						Error:   err.Error(),
					})
					if firstErr == nil {
						firstErr = validationErr
					}
					if !output.IsJSON() {
						return nil, validationErr
					}
					continue
				}

				output.Success("    %s: valid", service.Name)
				results = append(results, checkResult{
					Service: service.Name,
					Recipe:  recipeKey,
					Valid:   true,
				})
			}

			output.Success("All services are valid")

			jsonResult := struct {
				Success bool          `json:"success"`
				Results []checkResult `json:"results"`
			}{allValid, results}

			return jsonResult, firstErr
		}),
	}

	cmd.Flags().StringP("env", "e", "", "Environment to apply (merges overrides from environments block)")

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(CheckCmd())
}
