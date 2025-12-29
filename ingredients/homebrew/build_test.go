package homebrew_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/homebrew"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestPlatforms(t *testing.T) {
	t.Parallel()

	require.Len(t, homebrew.Platforms, 4)
	require.Contains(t, homebrew.Platforms, "darwin/amd64")
	require.Contains(t, homebrew.Platforms, "darwin/arm64")
	require.Contains(t, homebrew.Platforms, "linux/amd64")
	require.Contains(t, homebrew.Platforms, "linux/arm64")
}

func TestGenerateBuildCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		service         serviceinfo.ServiceInfo
		tag             string
		outputDir       string
		expectMkdir     bool
		expectPlatforms int
	}{
		{
			name: "basic build command",
			service: serviceinfo.ServiceInfo{
				Name: "myapp",
			},
			tag:             "v1.0.0",
			outputDir:       "dist",
			expectMkdir:     true,
			expectPlatforms: 4,
		},
		{
			name: "build with version var injection",
			service: serviceinfo.ServiceInfo{
				Name: "myapp",
				BuildConfig: serviceinfo.BuildConfig{
					VersionVar: "main.version",
				},
			},
			tag:             "v2.0.0",
			outputDir:       "build",
			expectMkdir:     true,
			expectPlatforms: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := homebrew.GenerateBuildCommand(tt.service, tt.tag, tt.outputDir)

			require.NotEmpty(t, result)

			// Should start with mkdir
			require.Contains(t, result, "mkdir -p "+tt.outputDir)

			// Should contain all platform builds
			require.Contains(t, result, "GOOS=darwin GOARCH=amd64")
			require.Contains(t, result, "GOOS=darwin GOARCH=arm64")
			require.Contains(t, result, "GOOS=linux GOARCH=amd64")
			require.Contains(t, result, "GOOS=linux GOARCH=arm64")

			// Should contain CGO_ENABLED=0
			require.Contains(t, result, "CGO_ENABLED=0")

			// Should contain ldflags
			require.Contains(t, result, "-ldflags=")
			require.Contains(t, result, "-s -w")

			// Should use && to chain commands
			require.Contains(t, result, " && ")
		})
	}
}

func TestGenerateBuildCommandVersionInjection(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "myapp",
		BuildConfig: serviceinfo.BuildConfig{
			VersionVar: "main.version",
		},
	}

	result := homebrew.GenerateBuildCommand(service, "v1.2.3", "dist")

	// Should contain version injection in ldflags
	require.Contains(t, result, "-X main.version=v1.2.3")
}

func TestGenerateBuildCommandNoVersionVar(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:        "myapp",
		BuildConfig: serviceinfo.BuildConfig{},
	}

	result := homebrew.GenerateBuildCommand(service, "v1.0.0", "dist")

	// Should NOT contain -X flag
	require.NotContains(t, result, "-X ")
	// But should still have basic ldflags
	require.Contains(t, result, "-s -w")
}

func TestGenerateBuildCommandOutputPaths(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "myapp",
	}

	result := homebrew.GenerateBuildCommand(service, "v1.0.0", "dist")

	// Should contain correct output paths
	require.Contains(t, result, "dist/myapp_v1.0.0_darwin_amd64")
	require.Contains(t, result, "dist/myapp_v1.0.0_darwin_arm64")
	require.Contains(t, result, "dist/myapp_v1.0.0_linux_amd64")
	require.Contains(t, result, "dist/myapp_v1.0.0_linux_arm64")
}

func TestGenerateBuildCommandDifferentOutputDir(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "cli",
	}

	result := homebrew.GenerateBuildCommand(service, "v2.0.0", "/tmp/build")

	require.Contains(t, result, "mkdir -p /tmp/build")
	require.Contains(t, result, "/tmp/build/cli_v2.0.0_darwin_amd64")
}
