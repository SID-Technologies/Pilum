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

// LoadRecipesFromDirectory loads all recipe YAML files from the specified directory
// Returns a slice of RecipeInfo structs, ordered by the files' names.
func LoadRecipesFromDirectory(dirPath string) ([]RecipeInfo, error) {
	// Get all files from the directory
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read directory %s", dirPath)
	}

	// Filter for YAML files and sort by name
	var yamlFiles []string
	for _, file := range files {
		if !file.IsDir() && (filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml") {
			yamlFiles = append(yamlFiles, file.Name())
		}
	}

	// Sort files by name to ensure consistent order
	sort.Strings(yamlFiles)

	// Load recipes from each file
	recipeInfos := make([]RecipeInfo, 0, len(yamlFiles))
	for _, fileName := range yamlFiles {
		filePath := filepath.Join(dirPath, fileName)

		// Load YAML file
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read file %s", filePath)
		}

		// To ensure order is preserved, first unmarshal into a map
		// YAML doesn't guarantee order but we'll enforce it in our transformation
		var rawData map[string]any
		if err := yaml.Unmarshal(data, &rawData); err != nil {
			return nil, errors.Wrap(err, "failed to parse YAML from %s", filePath)
		}

		// Now marshal back to YAML with consistent ordering
		orderedYAML, err := yaml.Marshal(rawData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to reorder YAML from %s", filePath)
		}

		// Finally unmarshal to Recipe struct
		var recipe Recipe
		if err := yaml.Unmarshal(orderedYAML, &recipe); err != nil {
			return nil, errors.Wrap(err, "failed to parse ordered YAML from %s", filePath)
		}

		// Create RecipeInfo with provider and service info
		recipeInfo := RecipeInfo{
			Provider: recipe.Provider,
			Service:  recipe.Service,
			Recipe:   recipe,
		}

		output.Debugf("Loaded recipe: %s from file: %s", recipe.Name, filePath)
		for _, step := range recipe.Steps {
			output.Debugf("  Step: %s", step.Name)
		}

		recipeInfos = append(recipeInfos, recipeInfo)
	}

	return recipeInfos, nil
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
		return nil, errors.Wrap(err, "failed to read embedded recipes")
	}

	// Filter for YAML files and sort by name
	var yamlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
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

		data, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read embedded file %s", filePath)
		}

		var rawData map[string]any
		if err := yaml.Unmarshal(data, &rawData); err != nil {
			return nil, errors.Wrap(err, "failed to parse YAML from %s", filePath)
		}

		orderedYAML, err := yaml.Marshal(rawData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to reorder YAML from %s", filePath)
		}

		var recipe Recipe
		if err := yaml.Unmarshal(orderedYAML, &recipe); err != nil {
			return nil, errors.Wrap(err, "failed to parse ordered YAML from %s", filePath)
		}

		recipeInfo := RecipeInfo{
			Provider: recipe.Provider,
			Service:  recipe.Service,
			Recipe:   recipe,
		}

		output.Debugf("Loaded embedded recipe: %s", recipe.Name)
		for _, step := range recipe.Steps {
			output.Debugf("  Step: %s", step.Name)
		}

		recipeInfos = append(recipeInfos, recipeInfo)
	}

	return recipeInfos, nil
}
