package cmd

import (
	"slices"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/providers"
	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/suggest"
	"github.com/sid-technologies/pilum/schemas"

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
			recipePath, _ := cmd.Flags().GetString("recipe-path")
			recipes, err := loadRecipes(recipePath)
			if err != nil {
				return nil, err
			}

			if len(recipes) == 0 {
				output.Warning("No recipes found")
				return nil, nil
			}

			// The same lookup the pipeline uses, so check passes only what
			// deploy can run.
			recipeIndex := recipe.NewIndex(recipes)

			topLevelKeys, err := schemas.TopLevelKeys()
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.Config, err)
			}

			// Validate each service, collecting results for JSON
			type checkResult struct {
				Service  string   `json:"service"`
				Recipe   string   `json:"recipe"`
				Valid    bool     `json:"valid"`
				Error    string   `json:"error,omitempty"`
				Warnings []string `json:"warnings,omitempty"`
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
				recipe, lookupErr := recipeIndex.Lookup(&service)
				if lookupErr != nil {
					allValid = false
					message := lookupErr.Error()
					suggestion := suggest.FormatSuggestion(recipeKey, providers.GetAllRecipeKeys())
					if suggestion != "" && strings.HasPrefix(message, "no recipe") {
						message += " - " + suggestion
					}
					validationErr := exitcodes.WithCode(exitcodes.Config,
						errors.New("error checking service %s: %s", service.Name, message))
					results = append(results, checkResult{
						Service: service.Name,
						Recipe:  recipeKey,
						Valid:   false,
						Error:   message,
					})
					if firstErr == nil {
						firstErr = validationErr
					}
					if !output.IsJSON() {
						return nil, validationErr
					}
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

				schemaErrors, warnings := schemaFindings(&service, recipe, topLevelKeys)
				for _, warning := range warnings {
					output.Warning("    %s: %s", service.Name, warning)
				}
				if len(schemaErrors) > 0 {
					allValid = false
					message := "pilum.yaml doesn't match the schema: " + strings.Join(schemaErrors, "; ")
					validationErr := exitcodes.WithCode(exitcodes.Config,
						errors.New("error checking service %s: %s", service.Name, message))
					results = append(results, checkResult{
						Service:  service.Name,
						Recipe:   recipeKey,
						Valid:    false,
						Error:    message,
						Warnings: warnings,
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
					Service:  service.Name,
					Recipe:   recipeKey,
					Valid:    true,
					Warnings: warnings,
				})
			}

			if allValid {
				output.Success("All services are valid")
			}

			jsonResult := struct {
				Success bool          `json:"success"`
				Results []checkResult `json:"results"`
			}{allValid, results}

			return jsonResult, firstErr
		}),
	}

	cmd.Flags().StringP("env", "e", "", "Environment to apply (merges overrides from environments block)")
	cmd.Flags().String("recipe-path", "", "Load recipes from a directory instead of embedded recipes")

	return cmd
}

// schemaFindings splits a service's schema problems into errors and
// warnings. Top-level keys the schema doesn't define are warnings, not
// errors, because custom recipes read their own; keys the service's recipe
// names are not reported.
func schemaFindings(svc *serviceinfo.ServiceInfo, rcp recipe.Recipe, topLevelKeys []string) (schemaErrors, warnings []string) {
	for _, problem := range svc.SchemaProblems {
		if problem.Warning {
			warnings = append(warnings, problem.String())
			continue
		}
		schemaErrors = append(schemaErrors, problem.String())
	}

	known := map[string]bool{}
	for _, key := range topLevelKeys {
		known[key] = true
	}
	for _, field := range slices.Concat(rcp.GetRequiredFields(), rcp.GetOptionalFields()) {
		section, _, _ := strings.Cut(field.Name, ".")
		known[section] = true
	}

	var unknown []string
	for key := range svc.Config {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	slices.Sort(unknown)
	for _, key := range unknown {
		warning := key + ": unknown key"
		hint := schemas.SuggestKey(key, topLevelKeys)
		if hint != "" {
			warning += " - " + hint
		}
		warnings = append(warnings, warning)
	}

	return schemaErrors, warnings
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(CheckCmd())
}
