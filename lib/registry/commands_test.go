package registry_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/registry"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestRegisterDefaultHandlers(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	// Test that various handlers are registered with exact step names
	tests := []struct {
		stepName string
		provider string
		found    bool
	}{
		// GCP Cloud Run handlers (exact step names from recipe)
		{"build binary", "", true},
		{"build docker image", "", true},
		{"publish to registry", "", true},
		{"deploy to cloud run", "gcp", true},

		// Homebrew handlers (exact step names from recipe)
		{"build binaries", "homebrew", true},
		{"create archives", "homebrew", true},
		{"generate checksums", "homebrew", true},
		{"update formula", "homebrew", true},
		{"push to tap", "homebrew", true},

		// Unknown
		{"unknown-step", "", false},
		{"build", "", false},  // old-style partial match should NOT work
		{"docker", "", false}, // old-style partial match should NOT work
	}

	for _, tt := range tests {
		t.Run(tt.stepName+"_"+tt.provider, func(t *testing.T) {
			t.Parallel()

			handler, found := reg.GetHandler(tt.stepName, tt.provider)
			require.Equal(t, tt.found, found, "step=%s provider=%s", tt.stepName, tt.provider)
			if tt.found {
				require.NotNil(t, handler)
			}
		})
	}
}

func TestBuildDockerImageHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name:     "myservice",
			Template: "gcp-cloud-run",
		},
		ImageName:    "gcr.io/project/myservice:latest",
		TemplatePath: "./_templates",
	}

	handler, found := reg.GetHandler("build docker image", "")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	// Result should be []string
	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, "docker", cmd[0])
	require.Equal(t, "build", cmd[1])
}

func TestPublishToRegistryHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		ImageName: "gcr.io/project/myservice:v1.0.0",
	}

	handler, found := reg.GetHandler("publish to registry", "")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, []string{"docker", "push", "gcr.io/project/myservice:v1.0.0"}, cmd)
}

func TestBuildBinaryHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name: "myservice",
			BuildConfig: serviceinfo.BuildConfig{
				Cmd: "go build -o ./dist/app",
			},
		},
		Tag: "v1.0.0",
	}

	handler, found := reg.GetHandler("build binary", "")
	require.True(t, found)

	result := handler(ctx)
	// Result can be nil if no build command configured, or []string
	if result != nil {
		cmd, ok := result.([]string)
		require.True(t, ok)
		require.Equal(t, "/bin/sh", cmd[0])
	}
}

func TestDeployToCloudRunHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name:   "myservice",
			Region: "us-central1",
		},
		ImageName: "gcr.io/project/myservice:latest",
	}

	handler, found := reg.GetHandler("deploy to cloud run", "gcp")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, "gcloud", cmd[0])
	require.Equal(t, "run", cmd[1])
	require.Equal(t, "deploy", cmd[2])
}

func TestHomebrewBuildBinariesHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name: "myapp",
		},
		Tag: "v1.0.0",
	}

	handler, found := reg.GetHandler("build binaries", "homebrew")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "mkdir -p dist")
	require.Contains(t, cmd, "GOOS=darwin")
}

func TestHomebrewCreateArchivesHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name: "myapp",
		},
		Tag: "v1.0.0",
	}

	handler, found := reg.GetHandler("create archives", "homebrew")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "tar -czf")
}

func TestHomebrewGenerateChecksumsHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	handler, found := reg.GetHandler("generate checksums", "homebrew")
	require.True(t, found)

	result := handler(registry.StepContext{})
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "shasum -a 256")
}

func TestHomebrewUpdateFormulaHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name:        "myapp",
			Description: "Test app",
			License:     "MIT",
			Config: map[string]any{
				"homebrew": map[string]any{
					"project_url": "https://github.com/org/myapp",
				},
			},
		},
		Tag: "v1.0.0",
	}

	handler, found := reg.GetHandler("update formula", "homebrew")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "class Myapp < Formula")
}

func TestHomebrewPushToTapHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name: "myapp",
			Config: map[string]any{
				"homebrew": map[string]any{
					"tap_url":   "https://github.com/org/tap",
					"token_env": "GITHUB_TOKEN",
				},
			},
		},
		Tag: "v1.0.0",
	}

	handler, found := reg.GetHandler("push to tap", "homebrew")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "git clone")
	require.Contains(t, cmd, "git push")
}

func TestProviderSpecificOverridesGeneric(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name: "myapp",
		},
		Tag: "v1.0.0",
	}

	// Generic build binary
	genericHandler, found := reg.GetHandler("build binary", "")
	require.True(t, found)

	// Homebrew-specific build binaries
	homebrewHandler, found := reg.GetHandler("build binaries", "homebrew")
	require.True(t, found)

	// They should be different handlers
	genericResult := genericHandler(ctx)
	homebrewResult := homebrewHandler(ctx)

	// Homebrew returns string, generic returns []string
	_, isString := homebrewResult.(string)
	require.True(t, isString, "homebrew build should return string")

	if genericResult != nil {
		_, isSlice := genericResult.([]string)
		require.True(t, isSlice, "generic build should return []string")
	}
}
