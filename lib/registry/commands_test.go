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

	// Test that various handlers are registered
	tests := []struct {
		stepName string
		provider string
		found    bool
	}{
		// Docker handlers
		{"docker", "", true},
		{"push", "", true},
		{"publish", "", true},

		// Build handlers
		{"build", "", true},

		// Deploy handlers
		{"deploy", "gcp", true},
		{"deploy", "aws", true},
		{"deploy", "azure", true},

		// Homebrew handlers
		{"build", "homebrew", true},
		{"archive", "homebrew", true},
		{"checksum", "homebrew", true},
		{"formula", "homebrew", true},
		{"tap", "homebrew", true},

		// Unknown
		{"unknown-step", "", false},
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

func TestDockerHandlerExecution(t *testing.T) {
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

	handler, found := reg.GetHandler("docker", "")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	// Result should be []string
	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, "docker", cmd[0])
	require.Equal(t, "build", cmd[1])
}

func TestPushHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		ImageName: "gcr.io/project/myservice:v1.0.0",
	}

	handler, found := reg.GetHandler("push", "")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, []string{"docker", "push", "gcr.io/project/myservice:v1.0.0"}, cmd)
}

func TestPublishHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		ImageName: "myimage:tag",
	}

	handler, found := reg.GetHandler("publish", "")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, "docker", cmd[0])
	require.Equal(t, "push", cmd[1])
}

func TestBuildHandlerExecution(t *testing.T) {
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

	handler, found := reg.GetHandler("build", "")
	require.True(t, found)

	result := handler(ctx)
	// Result can be nil if no build command configured, or []string
	if result != nil {
		cmd, ok := result.([]string)
		require.True(t, ok)
		require.Equal(t, "/bin/sh", cmd[0])
	}
}

func TestGCPDeployHandlerExecution(t *testing.T) {
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

	handler, found := reg.GetHandler("deploy", "gcp")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, "gcloud", cmd[0])
	require.Equal(t, "run", cmd[1])
	require.Equal(t, "deploy", cmd[2])
}

func TestAWSDeployHandlerReturnsNil(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{}

	handler, found := reg.GetHandler("deploy", "aws")
	require.True(t, found)

	result := handler(ctx)
	require.Nil(t, result) // Not yet implemented
}

func TestAzureDeployHandlerReturnsNil(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{}

	handler, found := reg.GetHandler("deploy", "azure")
	require.True(t, found)

	result := handler(ctx)
	require.Nil(t, result) // Not yet implemented
}

func TestHomebrewBuildHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name: "myapp",
		},
		Tag: "v1.0.0",
	}

	handler, found := reg.GetHandler("build", "homebrew")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "mkdir -p dist")
	require.Contains(t, cmd, "GOOS=darwin")
}

func TestHomebrewArchiveHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name: "myapp",
		},
		Tag: "v1.0.0",
	}

	handler, found := reg.GetHandler("archive", "homebrew")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "tar -czf")
}

func TestHomebrewChecksumHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	handler, found := reg.GetHandler("checksum", "homebrew")
	require.True(t, found)

	result := handler(registry.StepContext{})
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "shasum -a 256")
}

func TestHomebrewFormulaHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name:        "myapp",
			Description: "Test app",
			License:     "MIT",
			HomebrewConfig: serviceinfo.HomebrewConfig{
				ProjectURL: "https://github.com/org/myapp",
			},
		},
		Tag: "v1.0.0",
	}

	handler, found := reg.GetHandler("formula", "homebrew")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "class Myapp < Formula")
}

func TestHomebrewTapHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name: "myapp",
			HomebrewConfig: serviceinfo.HomebrewConfig{
				TapURL:   "https://github.com/org/tap",
				TokenEnv: "GITHUB_TOKEN",
			},
		},
		Tag: "v1.0.0",
	}

	handler, found := reg.GetHandler("tap", "homebrew")
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

	// Generic build
	genericHandler, found := reg.GetHandler("build", "")
	require.True(t, found)

	// Homebrew-specific build
	homebrewHandler, found := reg.GetHandler("build", "homebrew")
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
