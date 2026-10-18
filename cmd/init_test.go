package cmd

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/lib/providers"
	"github.com/sid-technologies/pilum/lib/recipe"
	"github.com/sid-technologies/pilum/lib/templates"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

	yaml := generateServiceYAML("gcp", "cloud-run", values, nil, buildConfig)

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

	yaml := generateServiceYAML("homebrew", "", values, nil, buildConfig)

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

	yaml := generateServiceYAML("homebrew", "", values, nil, buildConfig)

	require.Contains(t, yaml, "name: my-service")
	require.Contains(t, yaml, "provider: homebrew")
	require.NotContains(t, yaml, "type:")
}

func TestFindRecipeByKey(t *testing.T) {
	t.Parallel()

	recipes := []recipe.Info{
		{Provider: "gcp", Service: "cloud-run", Recipe: recipe.Recipe{Name: "gcp-cloud-run"}},
		{Provider: "gcp", Service: "gke", Recipe: recipe.Recipe{Name: "gcp-gke"}},
		{Provider: "aws", Service: "lambda", Recipe: recipe.Recipe{Name: "aws-lambda"}},
		{Provider: "homebrew", Service: "", Recipe: recipe.Recipe{Name: "homebrew"}},
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

// Every service `pilum init` offers must have a recipe, or init writes a
// pilum.yaml that check rejects and deploy can't run.
func TestInitOffersOnlyServicesWithRecipes(t *testing.T) {
	t.Parallel()

	recipes, err := recipe.LoadEmbeddedRecipes()
	require.NoError(t, err)

	for _, provider := range providers.GetProviders() {
		if provider == "test" || provider == "unknown" {
			continue
		}

		services := providers.GetServices(provider)
		if len(services) == 0 {
			require.NotNil(t, findRecipeByKey(recipes, provider, ""), provider)
			continue
		}

		for _, service := range services {
			require.NotNil(t, findRecipeByKey(recipes, provider, service), provider+"-"+service)
		}
	}
}

// `cpu: 1` parses as a number, which Cloud Run's config reads as missing;
// string fields are quoted and int/bool fields are not.
func TestGenerateServiceYAMLQuotesByFieldType(t *testing.T) {
	t.Parallel()

	recipes, err := recipe.LoadEmbeddedRecipes()
	require.NoError(t, err)

	rec := findRecipeByKey(recipes, "gcp", "cloud-run")
	require.NotNil(t, rec)

	values := map[string]string{
		"name":                            "api",
		"cloud_run.cpu":                   "1",
		"cloud_run.min_instances":         "0",
		"cloud_run.allow_unauthenticated": "true",
	}
	buildConfig := &templates.BuildConfig{Language: "go", Version: "1.23", Cmd: "go build"}

	out := generateServiceYAML("gcp", "cloud-run", values, fieldTypes(rec), buildConfig)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(out), &parsed))

	cloudRun, ok := parsed["cloud_run"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "1", cloudRun["cpu"])
	require.Equal(t, 0, cloudRun["min_instances"])
	require.Equal(t, true, cloudRun["allow_unauthenticated"])
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

	fields := []recipe.Field{
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

	fields := []recipe.Field{
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

	nameFlag := cmd.Flags().Lookup("name")
	require.NotNil(t, nameFlag)
	require.Equal(t, "n", nameFlag.Shorthand)

	languageFlag := cmd.Flags().Lookup("language")
	require.NotNil(t, languageFlag)
	require.Equal(t, "l", languageFlag.Shorthand)

	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	require.Equal(t, "f", forceFlag.Shorthand)
}

func TestInitOptionsIsNonInteractive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opts     initOptions
		expected bool
	}{
		{
			name:     "all flags set",
			opts:     initOptions{provider: "gcp", name: "my-api", language: "go"},
			expected: true,
		},
		{
			name:     "with optional service",
			opts:     initOptions{provider: "gcp", service: "cloud-run", name: "my-api", language: "go"},
			expected: true,
		},
		{
			name:     "missing provider",
			opts:     initOptions{name: "my-api", language: "go"},
			expected: false,
		},
		{
			name:     "missing name",
			opts:     initOptions{provider: "gcp", language: "go"},
			expected: false,
		},
		{
			name:     "missing language",
			opts:     initOptions{provider: "gcp", name: "my-api"},
			expected: false,
		},
		{
			name:     "all empty",
			opts:     initOptions{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.opts.isNonInteractive())
		})
	}
}

func TestRunInitNonInteractiveInvalidProvider(t *testing.T) {
	t.Parallel()

	err := runInitNonInteractive(initOptions{
		provider: "nonexistent",
		name:     "my-service",
		language: "go",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown provider 'nonexistent'")
}

func TestRunInitNonInteractiveMissingService(t *testing.T) {
	t.Parallel()

	// GCP requires a service (cloud-run, cloud-run-job, etc.)
	err := runInitNonInteractive(initOptions{
		provider: "gcp",
		name:     "my-service",
		language: "go",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "requires --service flag")
}

func TestRunInitNonInteractiveInvalidService(t *testing.T) {
	t.Parallel()

	err := runInitNonInteractive(initOptions{
		provider: "gcp",
		service:  "nonexistent",
		name:     "my-service",
		language: "go",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown service 'nonexistent'")
}

func TestRunInitNonInteractiveInvalidLanguage(t *testing.T) {
	t.Parallel()

	err := runInitNonInteractive(initOptions{
		provider: "homebrew",
		name:     "my-cli",
		language: "cobol",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown language 'cobol'")
}

func TestRunInitNonInteractiveExistingFile(t *testing.T) {
	// Not parallel: uses os.Chdir which is process-global

	// Create a temp directory with a pilum.yaml
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(dir+"/pilum.yaml", []byte("name: existing"), 0600))

	// Change to temp dir for test
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	err := runInitNonInteractive(initOptions{
		provider: "homebrew",
		name:     "my-cli",
		language: "go",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestRunInitNonInteractiveForceOverwrite(t *testing.T) {
	// Not parallel: uses os.Chdir which is process-global

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(dir+"/pilum.yaml", []byte("name: existing"), 0600))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	err := runInitNonInteractive(initOptions{
		provider: "homebrew",
		name:     "my-cli",
		language: "go",
		force:    true,
	})

	require.NoError(t, err)

	// Verify file was overwritten
	content, readErr := os.ReadFile(dir + "/pilum.yaml")
	require.NoError(t, readErr)
	require.Contains(t, string(content), "name: my-cli")
	require.Contains(t, string(content), "provider: homebrew")
}

func TestRunInitNonInteractiveGeneratesYAML(t *testing.T) {
	// Not parallel: uses os.Chdir which is process-global

	dir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	err := runInitNonInteractive(initOptions{
		provider: "gcp",
		service:  "cloud-run",
		name:     "my-api",
		language: "go",
	})

	require.NoError(t, err)

	content, readErr := os.ReadFile(dir + "/pilum.yaml")
	require.NoError(t, readErr)

	yaml := string(content)
	require.Contains(t, yaml, "name: my-api")
	require.Contains(t, yaml, "provider: gcp")
	require.Contains(t, yaml, "type: gcp-cloud-run")
	require.Contains(t, yaml, "build:")
	require.Contains(t, yaml, "language: go")
}

func TestRunInitDispatchesCorrectly(t *testing.T) {
	// Not parallel: uses os.Chdir which is process-global

	// Non-interactive: all required flags → should not need stdin
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(origDir) }()

	err := runInit(initOptions{
		provider: "homebrew",
		name:     "my-tool",
		language: "go",
	})

	require.NoError(t, err)

	content, readErr := os.ReadFile(dir + "/pilum.yaml")
	require.NoError(t, readErr)
	require.Contains(t, string(content), "name: my-tool")
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

func TestGenerateServiceYAMLRecipeFields(t *testing.T) {
	t.Parallel()

	buildConfig := &templates.BuildConfig{Language: "node", Version: "20", Cmd: "npm run build"}
	values := map[string]string{
		"name":               "site",
		"cloudflare.account": "abc123",
		"build.output_dir":   "dist",
		"build.cmd":          "npm run build",
	}
	out := generateServiceYAML("cloudflare", "pages", values, nil, buildConfig)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(out), &parsed), out)
	build := parsed["build"].(map[string]any)
	require.Equal(t, "dist", build["output_dir"])
	require.Equal(t, "npm run build", build["cmd"])
	require.Equal(t, map[string]any{"account_id": "abc123"}, parsed["cloudflare"])
}
