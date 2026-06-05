package serviceinfo_test

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestNewServiceInfo_NoSidecars_LeavesFieldEmpty(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "api",
		"provider": "gcp",
	}, "/tmp/api")

	require.NotNil(t, svc)
	require.Empty(t, svc.Sidecars)
}

func TestNewServiceInfo_ParsesSingleSidecar(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "mcp-controller-stripe",
		"provider": "gcp",
		"sidecars": []any{
			map[string]any{
				"name":   "otel-collector",
				"image":  "us-docker.pkg.dev/sid-platform/observability/otel-collector:v0.1.0",
				"memory": "128Mi",
				"cpu":    "0.25",
				"env_vars": map[string]any{
					"GOOGLE_CLOUD_PROJECT": "romans-dev",
				},
				"depends_on": []any{"mcp-controller-stripe"},
			},
		},
	}, "/tmp/svc")

	require.NotNil(t, svc)
	require.Len(t, svc.Sidecars, 1)

	sc := svc.Sidecars[0]
	require.Equal(t, "otel-collector", sc.Name)
	require.Equal(t, "us-docker.pkg.dev/sid-platform/observability/otel-collector:v0.1.0", sc.Image)
	require.Equal(t, "128Mi", sc.Memory)
	require.Equal(t, "0.25", sc.CPU)
	require.Equal(t, []string{"mcp-controller-stripe"}, sc.DependsOn)

	require.Len(t, sc.EnvVars, 1)
	require.Equal(t, "GOOGLE_CLOUD_PROJECT", sc.EnvVars[0].Name)
	require.Equal(t, "romans-dev", sc.EnvVars[0].Value)
}

func TestNewServiceInfo_ParsesMultipleSidecars(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "api",
		"provider": "gcp",
		"sidecars": []any{
			map[string]any{"name": "logger", "image": "gcr.io/p/logger:v1"},
			map[string]any{"name": "metrics", "image": "gcr.io/p/metrics:v1"},
			map[string]any{"name": "proxy", "image": "gcr.io/p/proxy:v1"},
		},
	}, "/tmp/svc")

	require.Len(t, svc.Sidecars, 3)
	require.Equal(t, "logger", svc.Sidecars[0].Name)
	require.Equal(t, "metrics", svc.Sidecars[1].Name)
	require.Equal(t, "proxy", svc.Sidecars[2].Name)
}

func TestNewServiceInfo_ParsesSidecarStartupProbe(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "api",
		"provider": "gcp",
		"sidecars": []any{
			map[string]any{
				"name":  "otel",
				"image": "gcr.io/p/otel:v1",
				"startup_probe": map[string]any{
					"path":                  "/healthz",
					"port":                  13133,
					"initial_delay_seconds": 5,
					"period_seconds":        10,
					"timeout_seconds":       3,
					"failure_threshold":     6,
				},
			},
		},
	}, "/tmp/svc")

	require.Len(t, svc.Sidecars, 1)
	require.NotNil(t, svc.Sidecars[0].StartupProbe)

	p := svc.Sidecars[0].StartupProbe
	require.Equal(t, "/healthz", p.Path)
	require.Equal(t, 13133, p.Port)
	require.Equal(t, 5, p.InitialDelaySeconds)
	require.Equal(t, 10, p.PeriodSeconds)
	require.Equal(t, 3, p.TimeoutSeconds)
	require.Equal(t, 6, p.FailureThreshold)
}

// Cloud Run distinguishes absent probe from all-zero probe.
func TestNewServiceInfo_SidecarWithoutProbeIsNil(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "api",
		"provider": "gcp",
		"sidecars": []any{
			map[string]any{"name": "sc", "image": "gcr.io/p/sc:v1"},
		},
	}, "/tmp/svc")

	require.Len(t, svc.Sidecars, 1)
	require.Nil(t, svc.Sidecars[0].StartupProbe)
}

func TestNewServiceInfo_SidecarSecrets(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "api",
		"provider": "gcp",
		"sidecars": []any{
			map[string]any{
				"name":  "sc",
				"image": "gcr.io/p/sc:v1",
				"secrets": map[string]any{
					"API_KEY": "projects/p/secrets/api-key/versions/latest",
				},
			},
		},
	}, "/tmp/svc")

	require.Len(t, svc.Sidecars, 1)
	require.Len(t, svc.Sidecars[0].Secrets, 1)
	require.Equal(t, "API_KEY", svc.Sidecars[0].Secrets[0].Name)
	require.Equal(t, "projects/p/secrets/api-key/versions/latest", svc.Sidecars[0].Secrets[0].Value)
}

func TestNewServiceInfo_ArtifactRegistryImageType(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":          "sid-otel-collector",
		"type":          "gcp-artifact-registry-image",
		"image_name":    "otel-collector",
		"version":       "v0.1.0",
		"registry_name": "us-docker.pkg.dev/sid-platform/observability",
	}, "/tmp/otel")

	require.NotNil(t, svc)
	require.Equal(t, "gcp", svc.Provider)
	require.Equal(t, "otel-collector", svc.ImageName)
	require.Equal(t, "v0.1.0", svc.Version)
}
