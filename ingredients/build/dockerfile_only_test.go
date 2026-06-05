package build_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/build"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateBuildCommand_NoBuildCmd_StillReturnsImageName(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:         "sid-otel-collector",
		ImageName:    "otel-collector",
		Version:      "v0.1.0",
		Provider:     "gcp",
		Region:       "us-central1",
		Project:      "romans-dev",
		RegistryName: "romans-dev",
	}

	cmd, imageName := build.GenerateBuildCommand(svc, "", "")

	require.Nil(t, cmd)
	require.Equal(t,
		"us-central1-docker.pkg.dev/romans-dev/romans-dev/otel-collector:v0.1.0",
		imageName)
}

func TestResolveImageName_StableAcrossBuildCmdPresence(t *testing.T) {
	t.Parallel()

	base := serviceinfo.ServiceInfo{
		Name:         "api",
		Provider:     "gcp",
		Region:       "us-central1",
		Project:      "p",
		RegistryName: "p",
	}

	withBuild := base
	withBuild.BuildConfig = serviceinfo.BuildConfig{Cmd: "go build -o ./dist"}

	withoutBuild := base

	require.Equal(t,
		build.ResolveImageName(withBuild, "", ""),
		build.ResolveImageName(withoutBuild, "", ""))
}

func TestGenerateBuildCommand_NoBuildCmd_RespectsTagPrecedence(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:         "x",
		ImageName:    "x-image",
		Version:      "v0.7.0",
		Provider:     "gcp",
		Region:       "us",
		Project:      "p",
		RegistryName: "p",
	}

	_, imageName := build.GenerateBuildCommand(svc, "", "")
	require.Contains(t, imageName, ":v0.7.0")

	_, imageNameWithCLI := build.GenerateBuildCommand(svc, "", "abc1234")
	require.Contains(t, imageNameWithCLI, ":abc1234")
}
