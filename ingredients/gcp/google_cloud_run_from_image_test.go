package gcp_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

// TestGenerateGCPDeployCommand_FromImage_UsesExplicitImage verifies the
// gcp-cloud-run-from-image type passes the pre-built image reference straight
// through to the deploy command. No build artifacts to coordinate with; the
// image string is the input.
func TestGenerateGCPDeployCommand_FromImage_UsesExplicitImage(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "statio-mcp-stripe",
		Region:  "us-central1",
		Project: "romans-dev",
		Image:   "us-central1-docker.pkg.dev/romans-dev/romans-dev/mcp-controller:v1.0.0",
		Config:  map[string]any{},
	}

	// The handler for "deploy from image" passes svc.Image as the image arg.
	cmd := gcp.GenerateGCPDeployCommand(svc, svc.Image)

	require.Contains(t, cmd, svc.Image,
		"deploy-from-image must reference the configured image as-is, not construct one from name/registry")
}

// TestGenerateGCPDeployCommand_FromImage_WithSidecars covers the canonical
// use case: a pre-built mcp-controller image deployed alongside the shared
// OTel collector sidecar. Both image references must appear in the command,
// and the multi-container flag ordering must still hold.
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

	require.Contains(t, cmd, mainImage, "ingress container must use the from-image reference")
	require.Contains(t, cmd, otelImage, "sidecar container image must also appear")

	// Spot-check ordering: --image for the ingress comes before the
	// --container=otel-collector declaration.
	mainImageIdx := indexOf(cmd, mainImage)
	otelContainerIdx := indexOf(cmd, "otel-collector")

	require.Greater(t, mainImageIdx, 0)
	require.Greater(t, otelContainerIdx, mainImageIdx,
		"ingress image must be emitted before sidecar --container declaration")
}

func indexOf(cmd []string, target string) int {
	for i, arg := range cmd {
		if arg == target {
			return i
		}
	}

	return -1
}
