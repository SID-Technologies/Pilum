package gcp_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateGCPDeployCommand_FromImage_UsesExplicitImage(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "statio-mcp-stripe",
		Region:  "us-central1",
		Project: "romans-dev",
		Image:   "us-central1-docker.pkg.dev/romans-dev/romans-dev/mcp-controller:v1.0.0",
		Config:  map[string]any{},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, svc.Image)

	require.Contains(t, cmd, svc.Image)
}

func TestGenerateGCPDeployCommand_FromImage_WithSidecars(t *testing.T) {
	t.Parallel()

	const mainImage = "us-central1-docker.pkg.dev/romans-dev/romans-dev/mcp-controller:v1.0.0"
	const otelImage = "us-central1-docker.pkg.dev/romans-dev/romans-dev/otel-collector:v0.1.0"

	svc := serviceinfo.ServiceInfo{
		Name:    "statio-mcp-stripe",
		Region:  "us-central1",
		Project: "romans-dev",
		Image:   mainImage,
		Sidecars: []serviceinfo.Sidecar{
			{
				Name:   "otel-collector",
				Image:  otelImage,
				Memory: "128Mi",
				CPU:    "0.25",
			},
		},
		Config: map[string]any{
			"cloud_run": map[string]any{
				"memory": "4Gi",
				"cpu":    "2",
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, svc.Image)

	require.Contains(t, cmd, mainImage)
	require.Contains(t, cmd, otelImage)

	mainImageIdx := indexOf(cmd, mainImage)
	otelContainerIdx := indexOf(cmd, "otel-collector")

	require.Greater(t, mainImageIdx, 0)
	require.Greater(t, otelContainerIdx, mainImageIdx)
}

func indexOf(cmd []string, target string) int {
	for i, arg := range cmd {
		if arg == target {
			return i
		}
	}

	return -1
}
