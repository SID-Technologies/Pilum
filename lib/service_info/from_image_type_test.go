package serviceinfo_test

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

// TestNewServiceInfo_FromImageType_ParsesImageField — the deploy-only type
// reads its image reference from the top-level `image:` field. Without
// this, fan-out deploys (one image, many services) have no way to point
// at the pre-built image they want to deploy.
func TestNewServiceInfo_FromImageType_ParsesImageField(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":  "statio-mcp-stripe",
		"type":  "gcp-cloud-run-from-image",
		"image": "us-central1-docker.pkg.dev/romans-dev/romans-dev/mcp-controller:v1.0.0",
	}, "/tmp/stripe")

	require.NotNil(t, svc)
	require.Equal(t, "gcp", svc.Provider, "gcp-cloud-run-from-image must derive provider=gcp")
	require.Equal(t, "us-central1-docker.pkg.dev/romans-dev/romans-dev/mcp-controller:v1.0.0", svc.Image,
		"`image:` field must populate svc.Image for the deploy handler to find")
}

// TestNewServiceInfo_FromImageType_RecipeKeyResolves — the recipe loader
// keys on Type. If the type doesn't resolve, the deploy dispatcher can't
// find the handler.
func TestNewServiceInfo_FromImageType_RecipeKeyResolves(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":  "statio-mcp-stripe",
		"type":  "gcp-cloud-run-from-image",
		"image": "gcr.io/p/img:v1",
	}, "/tmp/stripe")

	require.Equal(t, "gcp-cloud-run-from-image", svc.RecipeKey(),
		"RecipeKey must equal the type so registry lookup succeeds")
}
