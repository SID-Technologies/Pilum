package serviceinfo_test

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestNewServiceInfo_FromImageType_ParsesImageField(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":  "statio-mcp-stripe",
		"type":  "gcp-cloud-run-from-image",
		"image": "us-central1-docker.pkg.dev/romans-dev/romans-dev/mcp-controller:v1.0.0",
	}, "/tmp/stripe")

	require.NotNil(t, svc)
	require.Equal(t, "gcp", svc.Provider)
	require.Equal(t, "us-central1-docker.pkg.dev/romans-dev/romans-dev/mcp-controller:v1.0.0", svc.Image)
}

func TestNewServiceInfo_FromImageType_RecipeKeyResolves(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":  "statio-mcp-stripe",
		"type":  "gcp-cloud-run-from-image",
		"image": "gcr.io/p/img:v1",
	}, "/tmp/stripe")

	require.Equal(t, "gcp-cloud-run-from-image", svc.RecipeKey())
}
