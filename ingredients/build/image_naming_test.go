package build_test

import (
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/ingredients/build"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func extractImageName(_ *testing.T, cmdImage string) (base, tag string) {
	parts := strings.SplitN(cmdImage, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return cmdImage, ""
}

func TestGenerateBuildCommand_ImageNameOverridesServiceName(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:         "sid-otel-collector",
		ImageName:    "otel-collector",
		Provider:     "gcp",
		Region:       "us",
		Project:      "sid-platform",
		RegistryName: "observability",
		BuildConfig: serviceinfo.BuildConfig{
			Cmd: "echo build",
		},
	}

	_, fullImage := build.GenerateBuildCommand(svc, "", "")

	base, _ := extractImageName(t, fullImage)
	require.Equal(t, "us-docker.pkg.dev/sid-platform/observability/otel-collector", base)
}

func TestGenerateBuildCommand_VersionFromPilumYaml(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "otel-collector",
		Version: "v0.1.0",
		BuildConfig: serviceinfo.BuildConfig{
			Cmd: "echo build",
		},
	}

	_, fullImage := build.GenerateBuildCommand(svc, "", "")

	_, tag := extractImageName(t, fullImage)
	require.Equal(t, "v0.1.0", tag)
}

func TestGenerateBuildCommand_CLITagOverridesYAMLVersion(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "otel-collector",
		Version: "v0.1.0",
		BuildConfig: serviceinfo.BuildConfig{
			Cmd: "echo build",
		},
	}

	_, fullImage := build.GenerateBuildCommand(svc, "", "abc1234")

	_, tag := extractImageName(t, fullImage)
	require.Equal(t, "abc1234", tag)
}

func TestGenerateBuildCommand_DefaultsToLatest(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "api",
		BuildConfig: serviceinfo.BuildConfig{
			Cmd: "echo build",
		},
	}

	_, fullImage := build.GenerateBuildCommand(svc, "", "")

	_, tag := extractImageName(t, fullImage)
	require.Equal(t, "latest", tag)
}
