package recepie

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/recepies"

	"gopkg.in/yaml.v3"
)

// RecipeInfo contains the essential recipe information.
type RecipeInfo struct {
	Provider string
	Service  string
	Recipe   Recipe
}

// LoadRecipesFromDirectory loads all recipe YAML files from the specified directory.
// Returns a slice of RecipeInfo structs, ordered by the files' names.
func LoadRecipesFromDirectory(dirPath string) ([]RecipeInfo, error) {
	return loadRecipesFromFS(os.DirFS(dirPath), ".")
}

// LoadEmbeddedRecipes loads recipes from the embedded filesystem.
// This is the default when no --recipe-path is specified.
func LoadEmbeddedRecipes() ([]RecipeInfo, error) {
	return loadRecipesFromFS(recepies.FS, ".")
}

// loadRecipesFromFS loads recipes from any fs.FS implementation.
func loadRecipesFromFS(fsys fs.FS, root string) ([]RecipeInfo, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read recipes directory")
	}

	// Filter for YAML files and sort by name for consistent ordering
	var yamlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && isYAMLFile(entry.Name()) {
			yamlFiles = append(yamlFiles, entry.Name())
		}
	}
	sort.Strings(yamlFiles)

	// Load recipes from each file
	recipeInfos := make([]RecipeInfo, 0, len(yamlFiles))
	for _, fileName := range yamlFiles {
		filePath := fileName
		if root != "." {
			filePath = root + "/" + fileName
		}

		recipeInfo, err := loadRecipeFile(fsys, filePath)
		if err != nil {
			return nil, err
		}

		recipeInfos = append(recipeInfos, recipeInfo)
	}

	return recipeInfos, nil
}

// loadRecipeFile loads a single recipe from the given filesystem and path.
func loadRecipeFile(fsys fs.FS, filePath string) (RecipeInfo, error) {
	data, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return RecipeInfo{}, errors.Wrap(err, "failed to read recipe file %s", filePath)
	}

	var recipe Recipe
	if err := yaml.Unmarshal(data, &recipe); err != nil {
		return RecipeInfo{}, errors.Wrap(err, "failed to parse recipe YAML from %s", filePath)
	}

	output.Debugf("Loaded recipe: %s from %s", recipe.Name, filePath)
	for _, step := range recipe.Steps {
		output.Debugf("  Step: %s", step.Name)
	}

	return RecipeInfo{
		Provider: recipe.Provider,
		Service:  recipe.Service,
		Recipe:   recipe,
	}, nil
}

// isYAMLFile returns true if the filename has a YAML extension.
func isYAMLFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}
