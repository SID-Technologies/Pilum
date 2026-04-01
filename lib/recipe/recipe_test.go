package recipe

import (
	"path/filepath"
	"runtime"
	"testing"
)

// getRecipesPath returns the path to the recipes directory relative to this test file.
func getRecipesPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not get current file path")
	}
	// Go from lib/recipe/ up to repo root, then into recipes/
	return filepath.Join(filepath.Dir(filename), "..", "..", "recipes")
}

func TestAllStepsHaveTags(t *testing.T) {
	recipesPath := getRecipesPath(t)

	recipeInfos, err := LoadRecipesFromDirectory(recipesPath)
	if err != nil {
		t.Fatalf("failed to load recipes: %v", err)
	}

	if len(recipeInfos) == 0 {
		t.Fatal("no recipes found")
	}

	for _, info := range recipeInfos {
		for _, step := range info.Recipe.Steps {
			if len(step.Tags) == 0 {
				t.Errorf("recipe %q step %q has no tags", info.Recipe.Name, step.Name)
			}
		}
	}
}

func TestStepsHaveValidTags(t *testing.T) {
	recipesPath := getRecipesPath(t)

	recipeInfos, err := LoadRecipesFromDirectory(recipesPath)
	if err != nil {
		t.Fatalf("failed to load recipes: %v", err)
	}

	validTags := map[string]bool{
		"build":   true,
		"push":    true,
		"deploy":  true,
		"execute": true,
	}

	for _, info := range recipeInfos {
		for _, step := range info.Recipe.Steps {
			for _, tag := range step.Tags {
				if !validTags[tag] {
					t.Errorf("recipe %q step %q has invalid tag %q (valid: build, push, deploy)",
						info.Recipe.Name, step.Name, tag)
				}
			}
		}
	}
}
