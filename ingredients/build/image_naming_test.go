package build_test

import (
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/ingredients/build"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

// extractImageName parses `--image=NAME:TAG` (or two-arg form) out of a built
// command. We test against the returned image name in these tests; helper
// kept here in case future tests want the actual command shape.
func extractImageName(_ *testing.T, cmdImage string) (base, tag string) {
	parts := strings.SplitN(cmdImage, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return cmdImage, ""
}

// TestGenerateBuildCommand_ImageNameOverridesServiceName covers the new
// gcp-artifact-registry-image use case: Pilum service is `sid-otel-collector`,
// but the published image at `otel-collector` so consumers reference a short
// stable name without the SID prefix bleeding into URLs.
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
	require.Equal(t, "us-docker.pkg.dev/sid-platform/observability/otel-collector", base,
		"image_name must override service.Name in the registry path so consumers see a clean image identifier")
}

// TestGenerateBuildCommand_VersionFromPilumYaml verifies the YAML `version:`
// field is used as the tag when no CLI --tag is passed. This is what makes
// the gcp-artifact-registry-image type self-contained — operators pin the
// version in source, not at deploy time.
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
	require.Equal(t, "v0.1.0", tag, "version: in pilum.yaml must become the image tag when no CLI --tag passed")
}

// TestGenerateBuildCommand_CLITagOverridesYAMLVersion locks the precedence:
// CI passes --tag with a git SHA, YAML version stays as a documented default
// for ad-hoc local builds. CLI wins so CI doesn't have to mutate source.
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
	require.Equal(t, "abc1234", tag, "CLI --tag must override pilum.yaml version: so CI can stamp git SHAs without source edits")
}

// TestGenerateBuildCommand_DefaultsToLatest covers the path that existed
// before this PR: no ImageName, no Version, no CLI tag → `:latest`.
// Existing services must keep this shape exactly.
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
	require.Equal(t, "latest", tag, "absent any version source, fall back to :latest")
}
