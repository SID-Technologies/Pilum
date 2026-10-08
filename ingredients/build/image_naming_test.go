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

func TestResolveImageName_MIGContainerUsesImageRepository(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "statio-egress-proxy",
		Type:     "gcp-mig-container",
		Provider: "gcp",
		Region:   "us-central1",
		Project:  "statio-499700",
		Image:    "us-central1-docker.pkg.dev/statio-499700/statio/statio-egress-proxy",
	}

	require.Equal(t, "us-central1-docker.pkg.dev/statio-499700/statio/statio-egress-proxy:v2",
		build.ResolveImageName(svc, svc.RegistryName, "v2"))

	// Other types keep treating `image:` as something they don't build.
	svc.Type = "gcp-cloud-run"
	require.Equal(t, "us-central1-docker.pkg.dev/statio-499700/statio-499700/statio-egress-proxy:v2",
		build.ResolveImageName(svc, svc.RegistryName, "v2"))
}
