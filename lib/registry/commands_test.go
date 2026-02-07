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

		// GCP Cloud Run Job handlers (exact step names from recipe)
		{"deploy job", "gcp", true},
		{"execute job", "gcp", true},

		// Homebrew handlers (exact step names from recipe)
		{"build binaries", "homebrew", true},
		{"create archives", "homebrew", true},
		{"generate checksums", "homebrew", true},
		{"update formula", "homebrew", true},
		{"push to tap", "homebrew", true},

		// npm handlers (exact step names from recipe)
		{"install dependencies", "npm", true},
		{"set version", "npm", true},
		{"build package", "npm", true},
		{"publish package", "npm", true},

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

func TestDeployJobHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name:   "migrations",
			Region: "us-central1",
			Config: map[string]any{},
		},
		ImageName: "us-docker.pkg.dev/project/repo/migrations:latest",
	}

	handler, found := reg.GetHandler("deploy job", "gcp")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, "gcloud", cmd[0])
	require.Equal(t, "run", cmd[1])
	require.Equal(t, "jobs", cmd[2])
	require.Equal(t, "deploy", cmd[3])
	require.Equal(t, "migrations", cmd[4])
}

func TestExecuteJobHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name:   "migrations",
			Region: "us-central1",
			Config: map[string]any{
				"job": map[string]any{
					"execute_on_deploy": true,
				},
			},
		},
	}

	handler, found := reg.GetHandler("execute job", "gcp")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, "gcloud", cmd[0])
	require.Equal(t, "execute", cmd[3])
	require.Contains(t, cmd, "--wait")
}

func TestExecuteJobHandlerSkipsWhenDisabled(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Name:   "migrations",
			Region: "us-central1",
			Config: map[string]any{},
		},
	}

	handler, found := reg.GetHandler("execute job", "gcp")
	require.True(t, found)

	result := handler(ctx)
	require.Nil(t, result) // Should return nil when execute_on_deploy is false
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

func TestNpmInstallDependenciesHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	handler, found := reg.GetHandler("install dependencies", "npm")
	require.True(t, found)

	result := handler(registry.StepContext{})
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, []string{"npm", "ci"}, cmd)
}

func TestNpmSetVersionHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Tag: "v2.1.0",
	}

	handler, found := reg.GetHandler("set version", "npm")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, "npm", cmd[0])
	require.Equal(t, "version", cmd[1])
	require.Equal(t, "2.1.0", cmd[2]) // v prefix stripped
}

func TestNpmBuildPackageHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			BuildConfig: serviceinfo.BuildConfig{
				Cmd: "npm run build",
			},
		},
	}

	handler, found := reg.GetHandler("build package", "npm")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.([]string)
	require.True(t, ok)
	require.Equal(t, "/bin/sh", cmd[0])
	require.Equal(t, "-c", cmd[1])
	require.Equal(t, "npm run build", cmd[2])
}

func TestNpmBuildPackageHandlerSkipsWhenNoBuildCmd(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{},
	}

	handler, found := reg.GetHandler("build package", "npm")
	require.True(t, found)

	result := handler(ctx)
	require.Nil(t, result) // No build cmd → nil
}

func TestNpmPublishPackageHandlerExecution(t *testing.T) {
	t.Parallel()

	reg := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(reg)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{
			Config: map[string]any{
				"npm": map[string]any{
					"scope":     "@sid-technologies",
					"token_env": "NODE_AUTH_TOKEN",
				},
			},
		},
	}

	handler, found := reg.GetHandler("publish package", "npm")
	require.True(t, found)

	result := handler(ctx)
	require.NotNil(t, result)

	cmd, ok := result.(string)
	require.True(t, ok)
	require.Contains(t, cmd, "npm publish")
	require.Contains(t, cmd, "NODE_AUTH_TOKEN")
	require.Contains(t, cmd, "@sid-technologies")
}
