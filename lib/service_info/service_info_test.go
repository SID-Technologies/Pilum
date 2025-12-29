package serviceinfo_test

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestServiceInfoValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		service     *serviceinfo.ServiceInfo
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid service",
			service: &serviceinfo.ServiceInfo{
				Name:     "myservice",
				Provider: "gcp",
			},
			expectError: false,
		},
		{
			name: "missing name",
			service: &serviceinfo.ServiceInfo{
				Name:     "",
				Provider: "gcp",
			},
			expectError: true,
			errorMsg:    "missing required field: name",
		},
		{
			name: "missing provider",
			service: &serviceinfo.ServiceInfo{
				Name:     "myservice",
				Provider: "",
			},
			expectError: true,
			errorMsg:    "missing required field: provider",
		},
		{
			name: "missing both",
			service: &serviceinfo.ServiceInfo{
				Name:     "",
				Provider: "",
			},
			expectError: true,
			errorMsg:    "missing required field: name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.service.Validate()

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewServiceInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         map[string]any
		path           string
		expectedName   string
		expectedRegion string
	}{
		{
			name: "basic service config",
			config: map[string]any{
				"name":     "myservice",
				"provider": "gcp",
				"region":   "us-central1",
				"project":  "my-project",
			},
			path:           "/path/to/service",
			expectedName:   "myservice",
			expectedRegion: "us-central1",
		},
		{
			name: "service with template",
			config: map[string]any{
				"name":     "myservice",
				"template": "gcp-cloud-run",
			},
			path:         "/path/to/service",
			expectedName: "myservice",
		},
		{
			name: "service with type fallback",
			config: map[string]any{
				"name": "myservice",
				"type": "aws-lambda",
			},
			path:         "/path/to/service",
			expectedName: "myservice",
		},
		{
			name:         "empty config",
			config:       map[string]any{},
			path:         "/path/to/service",
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := serviceinfo.NewServiceInfo(tt.config, tt.path)

			require.NotNil(t, svc)
			require.Equal(t, tt.expectedName, svc.Name)
			require.Equal(t, tt.path, svc.Path)
			if tt.expectedRegion != "" {
				require.Equal(t, tt.expectedRegion, svc.Region)
			}
		})
	}
}

func TestNewServiceInfoProviderDerivation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		template         string
		expectedProvider string
	}{
		{
			name:             "gcp-cloud-run template",
			template:         "gcp-cloud-run",
			expectedProvider: "gcp",
		},
		{
			name:             "gcp template",
			template:         "gcp",
			expectedProvider: "gcp",
		},
		{
			name:             "aws-lambda template",
			template:         "aws-lambda",
			expectedProvider: "aws",
		},
		{
			name:             "aws-ecs template",
			template:         "aws-ecs",
			expectedProvider: "aws",
		},
		{
			name:             "aws template",
			template:         "aws",
			expectedProvider: "aws",
		},
		{
			name:             "azure-container-apps template",
			template:         "azure-container-apps",
			expectedProvider: "azure",
		},
		{
			name:             "azure template",
			template:         "azure",
			expectedProvider: "azure",
		},
		{
			name:             "homebrew template",
			template:         "homebrew",
			expectedProvider: "homebrew",
		},
		{
			name:             "unknown template",
			template:         "unknown",
			expectedProvider: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := map[string]any{
				"name":     "myservice",
				"template": tt.template,
			}

			svc := serviceinfo.NewServiceInfo(config, "/path")

			require.Equal(t, tt.expectedProvider, svc.Provider)
		})
	}
}

func TestNewServiceInfoExplicitProviderOverridesDerivation(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"name":     "myservice",
		"template": "gcp-cloud-run",
		"provider": "custom",
	}

	svc := serviceinfo.NewServiceInfo(config, "/path")

	require.Equal(t, "custom", svc.Provider)
}

func TestNewServiceInfoWithBuildConfig(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"name":     "myservice",
		"provider": "gcp",
		"build": map[string]any{
			"language":    "go",
			"version":     "1.23",
			"cmd":         "go build -o ./dist/app",
			"version_var": "main.version",
			"env_vars": map[string]any{
				"CGO_ENABLED": "0",
				"GO111MODULE": "on",
			},
			"flags": map[string]any{
				"ldflags": []any{"-s", "-w"},
			},
		},
	}

	svc := serviceinfo.NewServiceInfo(config, "/path")

	require.NotNil(t, svc)
	require.Equal(t, "go", svc.BuildConfig.Language)
	require.Equal(t, "1.23", svc.BuildConfig.Version)
	require.Equal(t, "go build -o ./dist/app", svc.BuildConfig.Cmd)
	require.Equal(t, "main.version", svc.BuildConfig.VersionVar)
	require.Len(t, svc.BuildConfig.EnvVars, 2)
	require.Len(t, svc.BuildConfig.Flags, 1)
}

func TestNewServiceInfoWithHomebrewConfig(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"name":     "myservice",
		"provider": "homebrew",
		"homebrew": map[string]any{
			"tap_url":     "https://github.com/org/homebrew-tap",
			"project_url": "https://github.com/org/project",
			"token_env":   "GITHUB_TOKEN",
		},
	}

	svc := serviceinfo.NewServiceInfo(config, "/path")

	require.NotNil(t, svc)
	require.Equal(t, "https://github.com/org/homebrew-tap", svc.HomebrewConfig.TapURL)
	require.Equal(t, "https://github.com/org/project", svc.HomebrewConfig.ProjectURL)
	require.Equal(t, "GITHUB_TOKEN", svc.HomebrewConfig.TokenEnv)
}

func TestNewServiceInfoWithEnvVars(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"name":     "myservice",
		"provider": "gcp",
		"env_vars": map[string]any{
			"DATABASE_URL": "postgres://localhost/db",
			"API_KEY":      "secret",
		},
	}

	svc := serviceinfo.NewServiceInfo(config, "/path")

	require.NotNil(t, svc)
	require.Len(t, svc.EnvVars, 2)
}

func TestNewServiceInfoWithSecrets(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"name":     "myservice",
		"provider": "gcp",
		"secrets": map[string]any{
			"DB_PASSWORD": "projects/123/secrets/db-pass",
			"API_SECRET":  "projects/123/secrets/api-secret",
		},
	}

	svc := serviceinfo.NewServiceInfo(config, "/path")

	require.NotNil(t, svc)
	require.Len(t, svc.Secrets, 2)
}

func TestNewServiceInfoWithRuntime(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"name":     "myservice",
		"provider": "gcp",
		"runtime": map[string]any{
			"service": "my-cloud-run-service",
		},
	}

	svc := serviceinfo.NewServiceInfo(config, "/path")

	require.NotNil(t, svc)
	require.Equal(t, "my-cloud-run-service", svc.Runtime.Service)
}

func TestNewServiceInfoPreservesRawConfig(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"name":         "myservice",
		"provider":     "gcp",
		"custom_field": "custom_value",
	}

	svc := serviceinfo.NewServiceInfo(config, "/path")

	require.NotNil(t, svc)
	require.NotNil(t, svc.Config)
	require.Equal(t, "custom_value", svc.Config["custom_field"])
}

func TestNewServiceInfoWithYamlV2Map(t *testing.T) {
	t.Parallel()

	// Simulates how yaml.v2 parses maps (as map[interface{}]interface{})
	config := map[string]any{
		"name":     "myservice",
		"provider": "gcp",
		"build": map[any]any{
			"language": "go",
			"version":  "1.23",
			"cmd":      "go build",
		},
	}

	svc := serviceinfo.NewServiceInfo(config, "/path")

	require.NotNil(t, svc)
	require.Equal(t, "go", svc.BuildConfig.Language)
}

func TestNewServiceInfoAllFields(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"name":          "full-service",
		"description":   "A complete service config",
		"provider":      "gcp",
		"region":        "us-central1",
		"project":       "my-project",
		"license":       "MIT",
		"registry_name": "my-registry",
		"template":      "gcp-cloud-run",
	}

	svc := serviceinfo.NewServiceInfo(config, "/service/path")

	require.Equal(t, "full-service", svc.Name)
	require.Equal(t, "A complete service config", svc.Description)
	require.Equal(t, "gcp", svc.Provider)
	require.Equal(t, "us-central1", svc.Region)
	require.Equal(t, "my-project", svc.Project)
	require.Equal(t, "MIT", svc.License)
	require.Equal(t, "my-registry", svc.RegistryName)
	require.Equal(t, "gcp-cloud-run", svc.Template)
	require.Equal(t, "/service/path", svc.Path)
}

func TestBuildFlagStruct(t *testing.T) {
	t.Parallel()

	flag := serviceinfo.BuildFlag{
		Name:   "ldflags",
		Values: []string{"-s", "-w", "-X main.version=1.0.0"},
	}

	require.Equal(t, "ldflags", flag.Name)
	require.Len(t, flag.Values, 3)
	require.Equal(t, "-s", flag.Values[0])
}

func TestEnvVarsStruct(t *testing.T) {
	t.Parallel()

	ev := serviceinfo.EnvVars{
		Name:  "DATABASE_URL",
		Value: "postgres://localhost/db",
	}

	require.Equal(t, "DATABASE_URL", ev.Name)
	require.Equal(t, "postgres://localhost/db", ev.Value)
}

func TestSecretsStruct(t *testing.T) {
	t.Parallel()

	secret := serviceinfo.Secrets{
		Name:  "API_KEY",
		Value: "projects/123/secrets/api-key",
	}

	require.Equal(t, "API_KEY", secret.Name)
	require.Equal(t, "projects/123/secrets/api-key", secret.Value)
}

func TestBuildConfigStruct(t *testing.T) {
	t.Parallel()

	bc := serviceinfo.BuildConfig{
		Language:   "go",
		Version:    "1.23",
		Cmd:        "go build -o app",
		VersionVar: "main.version",
		EnvVars: []serviceinfo.EnvVars{
			{Name: "CGO_ENABLED", Value: "0"},
		},
		Flags: []serviceinfo.BuildFlag{
			{Name: "ldflags", Values: []string{"-s", "-w"}},
		},
	}

	require.Equal(t, "go", bc.Language)
	require.Equal(t, "1.23", bc.Version)
	require.Equal(t, "go build -o app", bc.Cmd)
	require.Equal(t, "main.version", bc.VersionVar)
	require.Len(t, bc.EnvVars, 1)
	require.Len(t, bc.Flags, 1)
}

func TestRuntimeConfigStruct(t *testing.T) {
	t.Parallel()

	rc := serviceinfo.RuntimeConfig{
		Service: "my-service",
	}

	require.Equal(t, "my-service", rc.Service)
}

func TestHomebrewConfigStruct(t *testing.T) {
	t.Parallel()

	hc := serviceinfo.HomebrewConfig{
		TapURL:     "https://github.com/org/tap",
		ProjectURL: "https://github.com/org/project",
		TokenEnv:   "GITHUB_TOKEN",
	}

	require.Equal(t, "https://github.com/org/tap", hc.TapURL)
	require.Equal(t, "https://github.com/org/project", hc.ProjectURL)
	require.Equal(t, "GITHUB_TOKEN", hc.TokenEnv)
}

func TestDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		service  serviceinfo.ServiceInfo
		expected string
	}{
		{
			name: "single region service",
			service: serviceinfo.ServiceInfo{
				Name:          "api-gateway",
				Region:        "us-central1",
				IsMultiRegion: false,
			},
			expected: "api-gateway",
		},
		{
			name: "multi-region service",
			service: serviceinfo.ServiceInfo{
				Name:          "api-gateway",
				Region:        "us-central1",
				IsMultiRegion: true,
			},
			expected: "api-gateway (us-central1)",
		},
		{
			name: "multi-region without region set",
			service: serviceinfo.ServiceInfo{
				Name:          "api-gateway",
				Region:        "",
				IsMultiRegion: true,
			},
			expected: "api-gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.service.DisplayName())
		})
	}
}

func TestExpandMultiRegion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		service         serviceinfo.ServiceInfo
		expectedCount   int
		expectedRegions []string
	}{
		{
			name: "single region - no expansion",
			service: serviceinfo.ServiceInfo{
				Name:    "api",
				Region:  "us-central1",
				Regions: nil,
			},
			expectedCount:   1,
			expectedRegions: []string{"us-central1"},
		},
		{
			name: "multi-region - expands to 3",
			service: serviceinfo.ServiceInfo{
				Name:    "api",
				Regions: []string{"us-central1", "europe-west1", "asia-east1"},
			},
			expectedCount:   3,
			expectedRegions: []string{"us-central1", "europe-west1", "asia-east1"},
		},
		{
			name: "empty regions array - no expansion",
			service: serviceinfo.ServiceInfo{
				Name:    "api",
				Region:  "us-east1",
				Regions: []string{},
			},
			expectedCount:   1,
			expectedRegions: []string{"us-east1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := serviceinfo.ExpandMultiRegion(tt.service)

			require.Len(t, result, tt.expectedCount)

			for i, svc := range result {
				require.Equal(t, tt.service.Name, svc.Name)
				require.Equal(t, tt.expectedRegions[i], svc.Region)

				if tt.expectedCount > 1 {
					require.True(t, svc.IsMultiRegion)
					require.Nil(t, svc.Regions)
				}
			}
		})
	}
}

func TestNewServiceInfoWithRegions(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"name":     "global-api",
		"provider": "gcp",
		"project":  "my-project",
		"regions":  []any{"us-central1", "europe-west1", "asia-east1"},
	}

	svc := serviceinfo.NewServiceInfo(config, "/path")

	require.NotNil(t, svc)
	require.Equal(t, "global-api", svc.Name)
	require.Len(t, svc.Regions, 3)
	require.Contains(t, svc.Regions, "us-central1")
	require.Contains(t, svc.Regions, "europe-west1")
	require.Contains(t, svc.Regions, "asia-east1")
}
