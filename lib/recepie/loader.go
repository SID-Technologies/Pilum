package recepie

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/sid-technologies/pilum/lib/errors"

	"github.com/rs/zerolog/log"
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

		log.Info().Msgf("Loaded recipe: %s from file: %s", recipe.Name, filePath)
		for _, step := range recipe.Steps {
			log.Info().Msgf("Step: %s", step.Name)
		}

		recipeInfos = append(recipeInfos, recipeInfo)
	}

	return recipeInfos, nil
}
