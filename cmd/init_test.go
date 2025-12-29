package cmd

import (
	"bufio"
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/lib/recepie"
	"github.com/stretchr/testify/require"
)

func TestGetAvailableProviders(t *testing.T) {
	t.Parallel()

	recipes := []recepie.RecipeInfo{
		{Provider: "gcp", Recipe: recepie.Recipe{Name: "gcp-cloud-run"}},
		{Provider: "aws", Recipe: recepie.Recipe{Name: "aws-lambda"}},
		{Provider: "gcp", Recipe: recepie.Recipe{Name: "gcp-gke"}}, // duplicate provider
		{Provider: "homebrew", Recipe: recepie.Recipe{Name: "homebrew"}},
	}

	providers := getAvailableProviders(recipes)

	require.Len(t, providers, 3)
	require.Contains(t, providers, "gcp")
	require.Contains(t, providers, "aws")
	require.Contains(t, providers, "homebrew")
}

func TestGetAvailableProvidersEmpty(t *testing.T) {
	t.Parallel()

	providers := getAvailableProviders([]recepie.RecipeInfo{})
	require.Empty(t, providers)
}

func TestFindRecipeByProvider(t *testing.T) {
	t.Parallel()

	recipes := []recepie.RecipeInfo{
		{Provider: "gcp", Recipe: recepie.Recipe{Name: "gcp-cloud-run"}},
		{Provider: "aws", Recipe: recepie.Recipe{Name: "aws-lambda"}},
	}

	recipe := findRecipeByProvider(recipes, "aws")
	require.NotNil(t, recipe)
	require.Equal(t, "aws-lambda", recipe.Name)
}

func TestFindRecipeByProviderNotFound(t *testing.T) {
	t.Parallel()

	recipes := []recepie.RecipeInfo{
		{Provider: "gcp", Recipe: recepie.Recipe{Name: "gcp-cloud-run"}},
	}

	recipe := findRecipeByProvider(recipes, "azure")
	require.Nil(t, recipe)
}

func TestGenerateServiceYAML(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"name":    "my-service",
		"project": "my-project",
		"region":  "us-central1",
	}

	yaml := generateServiceYAML("gcp", values)

	require.Contains(t, yaml, "name: my-service")
	require.Contains(t, yaml, "provider: gcp")
	require.Contains(t, yaml, "project: my-project")
	require.Contains(t, yaml, "region: us-central1")
	require.Contains(t, yaml, "build:")
}

func TestGenerateServiceYAMLWithNestedKeys(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"name":                "my-cli",
		"homebrew.project_url": "https://github.com/org/project",
	}

	yaml := generateServiceYAML("homebrew", values)

	require.Contains(t, yaml, "name: my-cli")
	require.Contains(t, yaml, "provider: homebrew")
	require.Contains(t, yaml, "homebrew:")
	require.Contains(t, yaml, "project_url: https://github.com/org/project")
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

	recipePathFlag := cmd.Flags().Lookup("recipe-path")
	require.NotNil(t, recipePathFlag)
	require.Equal(t, "./recepies", recipePathFlag.DefValue)
}
