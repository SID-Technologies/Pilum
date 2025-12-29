package recepie

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/stretchr/testify/require"
)

func TestRecipeValidateService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		recipe      Recipe
		service     *serviceinfo.ServiceInfo
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid service with all required fields",
			recipe: Recipe{
				Name:     "test-recipe",
				Provider: "gcp",
				RequiredFields: []RequiredField{
					{Name: "name", Description: "service name"},
					{Name: "region", Description: "deployment region"},
				},
			},
			service: &serviceinfo.ServiceInfo{
				Name:   "myservice",
				Region: "us-central1",
			},
			expectError: false,
		},
		{
			name: "missing required field",
			recipe: Recipe{
				Name:     "test-recipe",
				Provider: "gcp",
				RequiredFields: []RequiredField{
					{Name: "name", Description: "service name"},
					{Name: "region", Description: "deployment region"},
				},
			},
			service: &serviceinfo.ServiceInfo{
				Name: "myservice",
			},
			expectError: true,
			errorMsg:    "requires field 'region'",
		},
		{
			name: "field with default value",
			recipe: Recipe{
				Name:     "test-recipe",
				Provider: "gcp",
				RequiredFields: []RequiredField{
					{Name: "name", Description: "service name"},
					{Name: "region", Description: "deployment region", Default: "us-east1"},
				},
			},
			service: &serviceinfo.ServiceInfo{
				Name: "myservice",
			},
			expectError: false,
		},
		{
			name: "no required fields",
			recipe: Recipe{
				Name:           "simple-recipe",
				Provider:       "docker",
				RequiredFields: nil,
			},
			service: &serviceinfo.ServiceInfo{
				Name: "myservice",
			},
			expectError: false,
		},
		{
			name: "empty required fields",
			recipe: Recipe{
				Name:           "simple-recipe",
				Provider:       "docker",
				RequiredFields: []RequiredField{},
			},
			service: &serviceinfo.ServiceInfo{
				Name: "myservice",
			},
			expectError: false,
		},
		{
			name: "homebrew nested field valid",
			recipe: Recipe{
				Name:     "homebrew-recipe",
				Provider: "homebrew",
				RequiredFields: []RequiredField{
					{Name: "name", Description: "binary name"},
					{Name: "homebrew.project_url", Description: "project URL"},
				},
			},
			service: &serviceinfo.ServiceInfo{
				Name: "myapp",
				HomebrewConfig: serviceinfo.HomebrewConfig{
					ProjectURL: "https://github.com/org/myapp",
				},
			},
			expectError: false,
		},
		{
			name: "homebrew nested field missing",
			recipe: Recipe{
				Name:     "homebrew-recipe",
				Provider: "homebrew",
				RequiredFields: []RequiredField{
					{Name: "name", Description: "binary name"},
					{Name: "homebrew.tap_url", Description: "tap repository URL"},
				},
			},
			service: &serviceinfo.ServiceInfo{
				Name: "myapp",
				HomebrewConfig: serviceinfo.HomebrewConfig{
					ProjectURL: "https://github.com/org/myapp",
				},
			},
			expectError: true,
			errorMsg:    "requires field 'homebrew.tap_url'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.recipe.ValidateService(tt.service)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRecipeGetRequiredFields(t *testing.T) {
	t.Parallel()

	recipe := Recipe{
		Name:     "test-recipe",
		Provider: "gcp",
		RequiredFields: []RequiredField{
			{Name: "name", Description: "service name", Type: "string"},
			{Name: "region", Description: "deployment region", Type: "string", Default: "us-central1"},
		},
	}

	fields := recipe.GetRequiredFields()

	require.Len(t, fields, 2)
	require.Equal(t, "name", fields[0].Name)
	require.Equal(t, "service name", fields[0].Description)
	require.Equal(t, "string", fields[0].Type)
	require.Equal(t, "", fields[0].Default)
	require.Equal(t, "region", fields[1].Name)
	require.Equal(t, "us-central1", fields[1].Default)
}

func TestRecipeGetRequiredFieldsEmpty(t *testing.T) {
	t.Parallel()

	recipe := Recipe{
		Name:           "simple-recipe",
		RequiredFields: nil,
	}

	fields := recipe.GetRequiredFields()

	require.Nil(t, fields)
}

func TestGetServiceField(t *testing.T) {
	t.Parallel()

	svc := &serviceinfo.ServiceInfo{
		Name:         "myservice",
		Description:  "A test service",
		Project:      "my-project",
		License:      "MIT",
		Region:       "us-central1",
		Provider:     "gcp",
		Template:     "cloud-run",
		RegistryName: "gcr.io/my-project",
		HomebrewConfig: serviceinfo.HomebrewConfig{
			ProjectURL: "https://github.com/org/myapp",
			TapURL:     "https://github.com/org/homebrew-tap",
			TokenEnv:   "GITHUB_TOKEN",
		},
		Config: map[string]any{
			"custom_field": "custom_value",
		},
	}

	tests := []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "name field",
			fieldName: "name",
			expected:  "myservice",
		},
		{
			name:      "description field",
			fieldName: "description",
			expected:  "A test service",
		},
		{
			name:      "project field",
			fieldName: "project",
			expected:  "my-project",
		},
		{
			name:      "license field",
			fieldName: "license",
			expected:  "MIT",
		},
		{
			name:      "region field",
			fieldName: "region",
			expected:  "us-central1",
		},
		{
			name:      "provider field",
			fieldName: "provider",
			expected:  "gcp",
		},
		{
			name:      "template field",
			fieldName: "template",
			expected:  "cloud-run",
		},
		{
			name:      "registry_name field",
			fieldName: "registry_name",
			expected:  "gcr.io/my-project",
		},
		{
			name:      "homebrew project_url",
			fieldName: "homebrew.project_url",
			expected:  "https://github.com/org/myapp",
		},
		{
			name:      "homebrew tap_url",
			fieldName: "homebrew.tap_url",
			expected:  "https://github.com/org/homebrew-tap",
		},
		{
			name:      "homebrew token_env",
			fieldName: "homebrew.token_env",
			expected:  "GITHUB_TOKEN",
		},
		{
			name:      "custom field from config",
			fieldName: "custom_field",
			expected:  "custom_value",
		},
		{
			name:      "unknown field",
			fieldName: "nonexistent",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := getServiceField(svc, tt.fieldName)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetServiceFieldNilConfig(t *testing.T) {
	t.Parallel()

	svc := &serviceinfo.ServiceInfo{
		Name:   "myservice",
		Config: nil,
	}

	result := getServiceField(svc, "custom_field")
	require.Equal(t, "", result)
}

func TestGetServiceFieldConfigNonString(t *testing.T) {
	t.Parallel()

	svc := &serviceinfo.ServiceInfo{
		Name: "myservice",
		Config: map[string]any{
			"int_field": 42,
		},
	}

	result := getServiceField(svc, "int_field")
	require.Equal(t, "", result) // Non-string values return empty
}

func TestRecipeStepDefaults(t *testing.T) {
	t.Parallel()

	step := RecipeStep{
		Name:          "build",
		ExecutionMode: "root",
	}

	require.Equal(t, "build", step.Name)
	require.Equal(t, "root", step.ExecutionMode)
	require.Nil(t, step.Command)
	require.Nil(t, step.EnvVars)
	require.Nil(t, step.BuildFlags)
	require.Equal(t, 0, step.Timeout)
	require.False(t, step.Debug)
	require.Equal(t, 0, step.Retries)
	require.Nil(t, step.Tags)
}

func TestRecipeStepWithValues(t *testing.T) {
	t.Parallel()

	step := RecipeStep{
		Name:          "deploy",
		Command:       []string{"gcloud", "run", "deploy"},
		ExecutionMode: "service_dir",
		EnvVars:       map[string]string{"KEY": "value"},
		BuildFlags:    map[string]any{"flag": true},
		Timeout:       300,
		Debug:         true,
		Retries:       3,
		Tags:          []string{"deploy", "prod"},
	}

	require.Equal(t, "deploy", step.Name)
	require.Equal(t, []string{"gcloud", "run", "deploy"}, step.Command)
	require.Equal(t, "service_dir", step.ExecutionMode)
	require.Equal(t, map[string]string{"KEY": "value"}, step.EnvVars)
	require.Equal(t, map[string]any{"flag": true}, step.BuildFlags)
	require.Equal(t, 300, step.Timeout)
	require.True(t, step.Debug)
	require.Equal(t, 3, step.Retries)
	require.Equal(t, []string{"deploy", "prod"}, step.Tags)
}

func TestRequiredFieldStruct(t *testing.T) {
	t.Parallel()

	field := RequiredField{
		Name:        "region",
		Description: "The deployment region",
		Type:        "string",
		Default:     "us-central1",
	}

	require.Equal(t, "region", field.Name)
	require.Equal(t, "The deployment region", field.Description)
	require.Equal(t, "string", field.Type)
	require.Equal(t, "us-central1", field.Default)
}

func TestRecipeStruct(t *testing.T) {
	t.Parallel()

	recipe := Recipe{
		Name:        "gcp-cloud-run",
		Description: "Deploy to GCP Cloud Run",
		Provider:    "gcp",
		Service:     "cloud-run",
		RequiredFields: []RequiredField{
			{Name: "name", Description: "service name"},
		},
		Steps: []RecipeStep{
			{Name: "build", ExecutionMode: "root"},
			{Name: "deploy", ExecutionMode: "root"},
		},
	}

	require.Equal(t, "gcp-cloud-run", recipe.Name)
	require.Equal(t, "Deploy to GCP Cloud Run", recipe.Description)
	require.Equal(t, "gcp", recipe.Provider)
	require.Equal(t, "cloud-run", recipe.Service)
	require.Len(t, recipe.RequiredFields, 1)
	require.Len(t, recipe.Steps, 2)
}
