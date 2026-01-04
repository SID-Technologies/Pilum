package cmd

import (
	"bufio"
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/lib/recepie"
	"github.com/sid-technologies/pilum/lib/templates"

	"github.com/stretchr/testify/require"
)

func TestGenerateServiceYAML(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"name":    "my-service",
		"project": "my-project",
		"region":  "us-central1",
	}

	buildConfig := &templates.BuildConfig{
		Language: "go",
		Version:  "1.23",
		Cmd:      "go build -o ./dist",
		EnvVars:  map[string]string{"CGO_ENABLED": "0"},
	}

	yaml := generateServiceYAML("gcp", "cloud-run", values, buildConfig)

	require.Contains(t, yaml, "name: my-service")
	require.Contains(t, yaml, "provider: gcp")
	require.Contains(t, yaml, "type: gcp-cloud-run")
	require.Contains(t, yaml, "project: my-project")
	require.Contains(t, yaml, "region: us-central1")
	require.Contains(t, yaml, "build:")
	require.Contains(t, yaml, "language: go")
}

func TestGenerateServiceYAMLWithNestedKeys(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"name":                 "my-cli",
		"homebrew.project_url": "https://github.com/org/project",
	}

	buildConfig := &templates.BuildConfig{
		Language: "go",
		Version:  "1.23",
		Cmd:      "go build -o ./dist",
	}

	yaml := generateServiceYAML("homebrew", "", values, buildConfig)

	require.Contains(t, yaml, "name: my-cli")
	require.Contains(t, yaml, "provider: homebrew")
	require.Contains(t, yaml, "homebrew:")
	require.Contains(t, yaml, "project_url: https://github.com/org/project")
}

func TestGenerateServiceYAMLNoService(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"name": "my-service",
	}

	buildConfig := &templates.BuildConfig{
		Language: "go",
		Version:  "1.23",
		Cmd:      "go build -o ./dist",
	}

	yaml := generateServiceYAML("homebrew", "", values, buildConfig)

	require.Contains(t, yaml, "name: my-service")
	require.Contains(t, yaml, "provider: homebrew")
	require.NotContains(t, yaml, "type:")
}

func TestFindRecipeByKey(t *testing.T) {
	t.Parallel()

	recipes := []recepie.RecipeInfo{
		{Provider: "gcp", Service: "cloud-run", Recipe: recepie.Recipe{Name: "gcp-cloud-run"}},
		{Provider: "gcp", Service: "gke", Recipe: recepie.Recipe{Name: "gcp-gke"}},
		{Provider: "aws", Service: "lambda", Recipe: recepie.Recipe{Name: "aws-lambda"}},
		{Provider: "homebrew", Service: "", Recipe: recepie.Recipe{Name: "homebrew"}},
	}

	// Exact match
	recipe := findRecipeByKey(recipes, "gcp", "cloud-run")
	require.NotNil(t, recipe)
	require.Equal(t, "gcp-cloud-run", recipe.Name)

	// Different service
	recipe = findRecipeByKey(recipes, "gcp", "gke")
	require.NotNil(t, recipe)
	require.Equal(t, "gcp-gke", recipe.Name)

	// Provider-only match
	recipe = findRecipeByKey(recipes, "homebrew", "")
	require.NotNil(t, recipe)
	require.Equal(t, "homebrew", recipe.Name)

	// No match for unknown provider/service
	recipe = findRecipeByKey(recipes, "azure", "container-apps")
	require.Nil(t, recipe)
}

func TestMustGetwd(t *testing.T) {
	t.Parallel()

	dir := mustGetwd()
	require.NotEmpty(t, dir)
	require.NotEqual(t, "my-service", dir)
}

func TestPromptWithDefault(t *testing.T) {
	t.Parallel()

	// Simulate user pressing enter (empty input)
	reader := bufio.NewReader(strings.NewReader("\n"))

	result, err := prompt(reader, "Enter name", "default-value")
	require.NoError(t, err)
	require.Equal(t, "default-value", result)
}

func TestPromptWithInput(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(strings.NewReader("custom-value\n"))

	result, err := prompt(reader, "Enter name", "default-value")
	require.NoError(t, err)
	require.Equal(t, "custom-value", result)
}

func TestPromptTrimsWhitespace(t *testing.T) {
	t.Parallel()

	reader := bufio.NewReader(strings.NewReader("  trimmed  \n"))

	result, err := prompt(reader, "Enter name", "")
	require.NoError(t, err)
	require.Equal(t, "trimmed", result)
}

func TestPromptForFields(t *testing.T) {
	t.Parallel()

	fields := []recepie.Field{
		{Name: "project", Description: "Project ID", Default: "default-project"},
		{Name: "region", Description: "Region", Default: "us-central1"},
	}

	// Simulate user pressing enter for both (using defaults)
	reader := bufio.NewReader(strings.NewReader("\n\n"))
	values := make(map[string]string)

	err := promptForFields(reader, fields, values, true)
	require.NoError(t, err)
	require.Equal(t, "default-project", values["project"])
	require.Equal(t, "us-central1", values["region"])
}

func TestPromptForFieldsSkipsExisting(t *testing.T) {
	t.Parallel()

	fields := []recepie.Field{
		{Name: "name", Description: "Service name"},
		{Name: "project", Description: "Project ID"},
	}

	// Pre-populate name
	values := map[string]string{"name": "already-set"}
	reader := bufio.NewReader(strings.NewReader("my-project\n"))

	err := promptForFields(reader, fields, values, true)
	require.NoError(t, err)
	require.Equal(t, "already-set", values["name"]) // Should not be overwritten
	require.Equal(t, "my-project", values["project"])
}

func TestInitCmdExists(t *testing.T) {
	t.Parallel()

	cmd := InitCmd()
	require.NotNil(t, cmd)
	require.Equal(t, "init", cmd.Use)
	require.NotEmpty(t, cmd.Short)
}

func TestInitCmdFlags(t *testing.T) {
	t.Parallel()

	cmd := InitCmd()

	providerFlag := cmd.Flags().Lookup("provider")
	require.NotNil(t, providerFlag)
	require.Equal(t, "p", providerFlag.Shorthand)

	serviceFlag := cmd.Flags().Lookup("service")
	require.NotNil(t, serviceFlag)
	require.Equal(t, "s", serviceFlag.Shorthand)
}

func TestGetBuildTemplates(t *testing.T) {
	t.Parallel()

	languages := templates.GetAvailableLanguages()
	require.NotEmpty(t, languages)
	require.Contains(t, languages, "go")
}

func TestGetBuildConfig(t *testing.T) {
	t.Parallel()

	config, err := templates.GetBuildConfig("go")
	require.NoError(t, err)
	require.NotNil(t, config)
	require.Equal(t, "go", config.Language)
	require.NotEmpty(t, config.Version)
	require.NotEmpty(t, config.Cmd)
}
