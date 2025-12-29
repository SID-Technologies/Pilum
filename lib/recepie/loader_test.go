package recepie_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sid-technologies/pilum/lib/recepie"
	"github.com/stretchr/testify/require"
)

func getTestDataPath(t *testing.T) string {
	t.Helper()
	// Get path relative to this test file
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(wd, "testdata")
}

func TestLoadRecipesFromDirectory(t *testing.T) {
	t.Parallel()

	// Create a temp directory with test recipes
	tmpDir := t.TempDir()

	// Write test recipe files
	recipe1 := `
name: alpha-recipe
provider: alpha
steps:
  - name: build
    tags:
      - build
`
	recipe2 := `
name: beta-recipe
provider: beta
steps:
  - name: deploy
    tags:
      - deploy
`
	err := os.WriteFile(filepath.Join(tmpDir, "01-alpha.yaml"), []byte(recipe1), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "02-beta.yaml"), []byte(recipe2), 0644)
	require.NoError(t, err)

	// Load recipes
	recipes, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Len(t, recipes, 2)

	// Should be sorted by filename
	require.Equal(t, "alpha", recipes[0].Provider)
	require.Equal(t, "beta", recipes[1].Provider)
}

func TestLoadRecipesFromDirectoryWithYmlExtension(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	recipe := `
name: yml-recipe
provider: yml
steps:
  - name: test
    tags:
      - build
`
	err := os.WriteFile(filepath.Join(tmpDir, "recipe.yml"), []byte(recipe), 0644)
	require.NoError(t, err)

	recipes, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Len(t, recipes, 1)
	require.Equal(t, "yml", recipes[0].Provider)
}

func TestLoadRecipesFromDirectoryEmpty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	recipes, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Empty(t, recipes)
}

func TestLoadRecipesFromDirectoryNonExistent(t *testing.T) {
	t.Parallel()

	_, err := recepie.LoadRecipesFromDirectory("/nonexistent/path/xyz")

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read directory")
}

func TestLoadRecipesFromDirectoryInvalidYaml(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	invalidYaml := `
name: invalid
provider: test
steps:
  - name: step1
    tags: [build
`
	err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidYaml), 0644)
	require.NoError(t, err)

	_, err = recepie.LoadRecipesFromDirectory(tmpDir)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse YAML")
}

func TestLoadRecipesFromDirectoryIgnoresNonYaml(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	recipe := `
name: valid-recipe
provider: valid
steps:
  - name: test
    tags:
      - build
`
	err := os.WriteFile(filepath.Join(tmpDir, "recipe.yaml"), []byte(recipe), 0644)
	require.NoError(t, err)

	// Create non-YAML files that should be ignored
	err = os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# Readme"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{}"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "script.sh"), []byte("#!/bin/bash"), 0644)
	require.NoError(t, err)

	recipes, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Len(t, recipes, 1)
	require.Equal(t, "valid", recipes[0].Provider)
}

func TestLoadRecipesFromDirectoryIgnoresSubdirectories(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	recipe := `
name: root-recipe
provider: root
steps:
  - name: test
    tags:
      - build
`
	err := os.WriteFile(filepath.Join(tmpDir, "recipe.yaml"), []byte(recipe), 0644)
	require.NoError(t, err)

	// Create a subdirectory with a YAML file that should be ignored
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	subRecipe := `
name: sub-recipe
provider: sub
steps:
  - name: test
    tags:
      - build
`
	err = os.WriteFile(filepath.Join(subDir, "sub-recipe.yaml"), []byte(subRecipe), 0644)
	require.NoError(t, err)

	recipes, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Len(t, recipes, 1)
	require.Equal(t, "root", recipes[0].Provider)
}

func TestLoadRecipesFromDirectoryPreservesStepOrder(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	recipe := `
name: ordered-recipe
provider: ordered
steps:
  - name: step-one
    tags:
      - build
  - name: step-two
    tags:
      - build
  - name: step-three
    tags:
      - deploy
`
	err := os.WriteFile(filepath.Join(tmpDir, "recipe.yaml"), []byte(recipe), 0644)
	require.NoError(t, err)

	recipes, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Len(t, recipes, 1)
	require.Len(t, recipes[0].Recipe.Steps, 3)
	require.Equal(t, "step-one", recipes[0].Recipe.Steps[0].Name)
	require.Equal(t, "step-two", recipes[0].Recipe.Steps[1].Name)
	require.Equal(t, "step-three", recipes[0].Recipe.Steps[2].Name)
}

func TestLoadRecipesFromDirectoryWithRequiredFields(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	recipe := `
name: recipe-with-fields
provider: fields
required_fields:
  - name: project
    description: Project ID
    type: string
    default: my-project
  - name: region
    description: Region to deploy
    type: string
steps:
  - name: deploy
    tags:
      - deploy
`
	err := os.WriteFile(filepath.Join(tmpDir, "recipe.yaml"), []byte(recipe), 0644)
	require.NoError(t, err)

	recipes, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Len(t, recipes, 1)
	require.Len(t, recipes[0].Recipe.RequiredFields, 2)
	require.Equal(t, "project", recipes[0].Recipe.RequiredFields[0].Name)
	require.Equal(t, "my-project", recipes[0].Recipe.RequiredFields[0].Default)
	require.Equal(t, "region", recipes[0].Recipe.RequiredFields[1].Name)
}

func TestLoadRecipesFromDirectoryWithStepDetails(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	recipe := `
name: detailed-recipe
provider: detailed
steps:
  - name: build
    execution_mode: service_dir
    timeout: 300
    retries: 2
    tags:
      - build
    env_vars:
      GO111MODULE: "on"
      CGO_ENABLED: "0"
  - name: custom-command
    command: echo hello
    execution_mode: root
    tags:
      - deploy
`
	err := os.WriteFile(filepath.Join(tmpDir, "recipe.yaml"), []byte(recipe), 0644)
	require.NoError(t, err)

	recipes, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Len(t, recipes, 1)

	step1 := recipes[0].Recipe.Steps[0]
	require.Equal(t, "build", step1.Name)
	require.Equal(t, "service_dir", step1.ExecutionMode)
	require.Equal(t, 300, step1.Timeout)
	require.Equal(t, 2, step1.Retries)
	require.Equal(t, "on", step1.EnvVars["GO111MODULE"])
	require.Equal(t, "0", step1.EnvVars["CGO_ENABLED"])

	step2 := recipes[0].Recipe.Steps[1]
	require.Equal(t, "custom-command", step2.Name)
	require.Equal(t, "echo hello", step2.Command)
}

func TestLoadRecipesFromDirectorySortedByFilename(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create files with names that would sort differently alphabetically
	recipes := map[string]string{
		"c-recipe.yaml": `
name: charlie
provider: c
steps:
  - name: test
    tags:
      - build
`,
		"a-recipe.yaml": `
name: alpha
provider: a
steps:
  - name: test
    tags:
      - build
`,
		"b-recipe.yaml": `
name: bravo
provider: b
steps:
  - name: test
    tags:
      - build
`,
	}

	for filename, content := range recipes {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	loaded, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Len(t, loaded, 3)

	// Should be sorted by filename
	require.Equal(t, "a", loaded[0].Provider)
	require.Equal(t, "b", loaded[1].Provider)
	require.Equal(t, "c", loaded[2].Provider)
}

func TestRecipeInfoFields(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	recipe := `
name: full-recipe
description: A complete test recipe
provider: full-provider
service: full-service
steps:
  - name: test
    tags:
      - build
`
	err := os.WriteFile(filepath.Join(tmpDir, "recipe.yaml"), []byte(recipe), 0644)
	require.NoError(t, err)

	recipes, err := recepie.LoadRecipesFromDirectory(tmpDir)

	require.NoError(t, err)
	require.Len(t, recipes, 1)

	info := recipes[0]
	require.Equal(t, "full-provider", info.Provider)
	require.Equal(t, "full-service", info.Service)
	require.Equal(t, "full-recipe", info.Recipe.Name)
	require.Equal(t, "A complete test recipe", info.Recipe.Description)
}
