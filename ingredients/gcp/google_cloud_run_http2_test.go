package gcp_test

import (
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func countArg(cmd []string, target string) int {
	n := 0
	for _, arg := range cmd {
		if arg == target {
			n++
		}
	}

	return n
}

func hasPrefixArg(cmd []string, prefix string) bool {
	for _, arg := range cmd {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}

	return false
}

func singleContainerSvc(cloudRun map[string]any) serviceinfo.ServiceInfo {
	return serviceinfo.ServiceInfo{
		Name:    "api",
		Region:  "us-central1",
		Project: "p",
		Config:  map[string]any{"cloud_run": cloudRun},
	}
}

func sidecarSvc(cloudRun map[string]any) serviceinfo.ServiceInfo {
	return serviceinfo.ServiceInfo{
		Name:    "api",
		Region:  "us-central1",
		Project: "p",
		Sidecars: []serviceinfo.Sidecar{
			{Name: "otel-collector", Image: "gcr.io/p/otel:v1"},
		},
		Config: map[string]any{"cloud_run": cloudRun},
	}
}

func TestParseCloudRunConfig_UseHTTP2(t *testing.T) {
	t.Parallel()

	require.True(t, gcp.ParseCloudRunConfig(map[string]any{
		"cloud_run": map[string]any{"use_http2": true},
	}).UseHTTP2)
	require.False(t, gcp.ParseCloudRunConfig(map[string]any{
		"cloud_run": map[string]any{"memory": "512Mi"},
	}).UseHTTP2)
	require.False(t, gcp.ParseCloudRunConfig(map[string]any{}).UseHTTP2)
}

func TestGenerateGCPDeployCommand_SingleContainer_UseHTTP2(t *testing.T) {
	t.Parallel()

	cmd := gcp.GenerateGCPDeployCommand(singleContainerSvc(map[string]any{"use_http2": true}), "gcr.io/p/api:v1")

	require.Equal(t, 1, countArg(cmd, "--use-http2"))
	require.NotContains(t, cmd, "--no-use-http2")
}

func TestGenerateGCPDeployCommand_SingleContainer_HTTP2UnsetEmitsNothing(t *testing.T) {
	t.Parallel()

	for name, cloudRun := range map[string]map[string]any{
		"absent":         {"memory": "512Mi"},
		"explicit false": {"use_http2": false},
	} {
		cmd := gcp.GenerateGCPDeployCommand(singleContainerSvc(cloudRun), "gcr.io/p/api:v1")

		require.NotContains(t, cmd, "--use-http2", name)
		require.NotContains(t, cmd, "--no-use-http2", name)
	}
}

func TestGenerateGCPDeployCommand_SingleContainer_PortOnlyWhenSet(t *testing.T) {
	t.Parallel()

	withPort := gcp.GenerateGCPDeployCommand(singleContainerSvc(map[string]any{"port": 8080}), "gcr.io/p/api:v1")
	require.Contains(t, withPort, "--port=8080")
	require.Empty(t, containerIndexes(withPort))

	withoutPort := gcp.GenerateGCPDeployCommand(singleContainerSvc(map[string]any{"memory": "512Mi"}), "gcr.io/p/api:v1")
	require.False(t, hasPrefixArg(withoutPort, "--port"))
}

// --use-http2 is a container flag: it must sit in the ingress container's group, not a sidecar's.
func TestGenerateGCPDeployCommand_Sidecars_UseHTTP2InIngressGroup(t *testing.T) {
	t.Parallel()

	cmd := gcp.GenerateGCPDeployCommand(sidecarSvc(map[string]any{"port": 8080, "use_http2": true}), "gcr.io/p/api:v1")

	containers := containerIndexes(cmd)
	require.Len(t, containers, 2)
	require.Equal(t, "api", cmd[containers[0]+1])

	idx := flagIndex(cmd, "--use-http2")
	require.Equal(t, 1, countArg(cmd, "--use-http2"))
	require.Greater(t, idx, containers[0])
	require.Less(t, idx, containers[1])
}

func TestGenerateGCPDeployCommand_Sidecars_HTTP2UnsetEmitsNothing(t *testing.T) {
	t.Parallel()

	cmd := gcp.GenerateGCPDeployCommand(sidecarSvc(map[string]any{"port": 8080}), "gcr.io/p/api:v1")

	require.NotContains(t, cmd, "--use-http2")
	require.NotContains(t, cmd, "--no-use-http2")
}
