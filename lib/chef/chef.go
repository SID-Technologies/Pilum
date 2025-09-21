package chef

import (
	// "github.com/sid-technologies/centurion/lib/errors".
	"github.com/sid-technologies/centurion/lib/recepie"
	"github.com/sid-technologies/centurion/lib/registry"
)

// ExecuteRecipe runs all steps in a recipe with the provided context.
func ExecuteRecipe(recipe recepie.Recipe, context map[string]any, registry *registry.CommandRegistry) error {
	// for _, step := range recipe.Steps {
	// 	handler, exists := registry.Get(recipe.Provider, recipe.Service, step.Name)
	// 	if !exists {
	// 		return errors.New("no handler found for %s:%s:%s", recipe.Provider, recipe.Service, step.Name)
	// 	}

	// 	err := handler()
	// 	if err != nil {
	// 		return errors.Wrap(err, "failed to execute step %s", step.Name)
	// 	}
	// }

	return nil
}
