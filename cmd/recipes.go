package cmd

import (
	"sort"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recipe"
	"github.com/sid-technologies/pilum/lib/suggest"

	"github.com/spf13/cobra"
)

func RecipesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "recipes [name]",
		Aliases: []string{"providers"},
		Short:   "List available recipes or describe a specific one",
		Long:    "List all available deployment recipes, or pass a recipe name to see its required fields, optional fields, and deployment steps.",
		Args:    cobra.MaximumNArgs(1),
		RunE: withJSON(func(_ *cobra.Command, args []string) (any, error) {
			recipes, err := recipe.LoadEmbeddedRecipes()
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.Config,
					errors.Wrap(err, "failed to load recipes"))
			}

			// Sort recipes by name for consistent output
			sort.Slice(recipes, func(i, j int) bool {
				return recipes[i].Recipe.Name < recipes[j].Recipe.Name
			})

			// No args: list all recipes
			if len(args) == 0 {
				return listRecipes(recipes)
			}

			// With arg: describe a specific recipe
			return describeRecipe(recipes, args[0])
		}),
	}

	return cmd
}

func listRecipes(recipes []recipe.Info) (any, error) {
	items := make([]output.RecipeListItem, len(recipes))
	for i, r := range recipes {
		items[i] = output.RecipeListItem{
			Key:         r.Recipe.Name,
			Description: r.Recipe.Description,
		}
	}

	// Text output (suppressed in JSON mode)
	output.PrintRecipeList(items)

	// JSON output
	type jsonRecipe struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Provider    string `json:"provider"`
		Service     string `json:"service,omitempty"`
	}
	result := make([]jsonRecipe, len(recipes))
	for i, r := range recipes {
		result[i] = jsonRecipe{
			Name:        r.Recipe.Name,
			Description: r.Recipe.Description,
			Provider:    r.Provider,
			Service:     r.Service,
		}
	}
	return result, nil
}

func describeRecipe(recipes []recipe.Info, name string) (any, error) {
	// Find the recipe by name or key
	var found *recipe.Info
	var allNames []string
	for i, r := range recipes {
		allNames = append(allNames, r.Recipe.Name)
		if r.Recipe.Name == name || r.RecipeKey() == name {
			found = &recipes[i]
			break
		}
	}

	if found == nil {
		suggestion := suggest.FormatSuggestion(name, allNames)
		if suggestion != "" {
			return nil, exitcodes.WithCode(exitcodes.InvalidArgs,
				errors.New("unknown recipe '%s'. %s", name, suggestion))
		}
		return nil, exitcodes.WithCode(exitcodes.InvalidArgs,
			errors.New("unknown recipe '%s'. Run 'pilum recipes' to see available recipes", name))
	}

	r := found.Recipe

	// Build detail view
	detail := output.RecipeDetail{
		Name:        r.Name,
		Description: r.Description,
		Provider:    found.Provider,
		Service:     found.Service,
	}

	for _, f := range r.RequiredFields {
		detail.RequiredFields = append(detail.RequiredFields, output.RecipeField{
			Name:        f.Name,
			Description: f.Description,
			Type:        f.Type,
			Default:     f.Default,
		})
	}

	for _, f := range r.OptionalFields {
		detail.OptionalFields = append(detail.OptionalFields, output.RecipeField{
			Name:        f.Name,
			Description: f.Description,
			Type:        f.Type,
			Default:     f.Default,
		})
	}

	for _, s := range r.Steps {
		detail.Steps = append(detail.Steps, output.RecipeStep{
			Name:    s.Name,
			Tags:    s.Tags,
			Timeout: s.Timeout,
		})
	}

	// Text output (suppressed in JSON mode)
	output.PrintRecipeDetail(detail)

	// JSON output
	return detail, nil
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(RecipesCmd())
}
