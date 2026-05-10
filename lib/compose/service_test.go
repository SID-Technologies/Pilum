package compose

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestBuildServiceEntryImageResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		svc      serviceinfo.ServiceInfo
		opts     Options
		expected string
	}{
		{
			name:     "registry flag",
			svc:      serviceinfo.ServiceInfo{Name: "api"},
			opts:     Options{Registry: "ghcr.io/myorg", Tag: "v1"},
			expected: "ghcr.io/myorg/api:v1",
		},
		{
			// Project-only (no registry_name): falls back to using project as
			// the AR repo. Mirrors pilum build's behavior for the same input.
			name: "GCP artifact registry — project fallback",
			svc: serviceinfo.ServiceInfo{
				Name:     "api",
				Provider: "gcp",
				Region:   "us-central1",
				Project:  "my-project",
			},
			opts:     Options{Tag: "latest"},
			expected: "us-central1-docker.pkg.dev/my-project/my-project/api:latest",
		},
		{
			// With explicit registry_name (the platform-core case): repo
			// must come from registry_name, NOT project. This is the case
			// the previous implementation got wrong.
			name: "GCP artifact registry — explicit registry_name",
			svc: serviceinfo.ServiceInfo{
				Name:         "api",
				Provider:     "gcp",
				Region:       "us-central1",
				Project:      "my-project",
				RegistryName: "platform",
			},
			opts:     Options{Tag: "latest"},
			expected: "us-central1-docker.pkg.dev/my-project/platform/api:latest",
		},
		{
			name:     "registry name from config",
			svc:      serviceinfo.ServiceInfo{Name: "api", RegistryName: "gcr.io/my-project"},
			opts:     Options{Tag: "v2"},
			expected: "gcr.io/my-project/api:v2",
		},
		{
			name:     "plain image name",
			svc:      serviceinfo.ServiceInfo{Name: "api"},
			opts:     Options{Tag: "latest"},
			expected: "api:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := BuildServiceEntry(tt.svc, 0, tt.opts)
			require.Equal(t, tt.expected, svc.Image)
		})
	}
}

func TestBuildServiceEntrySecretDefaults(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "api",
		Secrets: []serviceinfo.Secrets{
			{Name: "JWT_SECRET"},
			{Name: "ADMIN_KEY"},
			{Name: "SOME_OTHER_SECRET"},
		},
	}

	cs := BuildServiceEntry(svc, 0, Options{Tag: "latest"})

	require.Contains(t, cs.Environment, "JWT_SECRET=dev-jwt-secret-do-not-use-in-production")
	require.Contains(t, cs.Environment, "ADMIN_KEY=dev-admin-key")
	require.Contains(t, cs.Environment, "SOME_OTHER_SECRET=${SOME_OTHER_SECRET}")
}

func TestBuildServiceEntryEnvVars(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "api",
		EnvVars: []serviceinfo.EnvVars{{Name: "FOO", Value: "bar"}},
	}

	cs := BuildServiceEntry(svc, 0, Options{Tag: "latest"})

	require.Contains(t, cs.Environment, "FOO=bar")
}

func TestBuildServiceEntryCloudRunEnvVars(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "api",
		Config: map[string]any{
			"cloud_run": map[string]any{
				"env_vars": map[string]any{
					"ALLOWED_ORIGINS": "http://localhost:3000",
				},
			},
		},
	}

	cs := BuildServiceEntry(svc, 0, Options{Tag: "latest"})

	require.Contains(t, cs.Environment, "ALLOWED_ORIGINS=http://localhost:3000")
}

func TestBuildServiceEntryHealthCheck(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "api",
		Config: map[string]any{
			"health_check": map[string]any{
				"path":          "/healthz",
				"initial_delay": "10s",
			},
		},
	}

	cs := BuildServiceEntry(svc, 0, Options{Tag: "latest"})

	require.NotNil(t, cs.Healthcheck)
	require.Contains(t, cs.Healthcheck.Test[1], "/healthz")
	require.Equal(t, "10s", cs.Healthcheck.StartPeriod)
}

func TestBuildServiceEntryRuntimeLimits(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "api",
		Config: map[string]any{
			"runtime": map[string]any{
				"memory": "2048Mi",
				"cpu":    2,
			},
		},
	}

	cs := BuildServiceEntry(svc, 0, Options{Tag: "latest"})

	require.Equal(t, "2048m", cs.MemLimit)
	require.Equal(t, "2", cs.CPUs)
}

func TestBuildServiceEntryPortExplicit(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "api",
		Config: map[string]any{"port": 9090},
	}

	cs := BuildServiceEntry(svc, 5, Options{Tag: "latest"})

	require.Contains(t, cs.Ports, "9090:8080")
}

func TestBuildServiceEntryPortAutoAssigned(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{Name: "api"}

	cs := BuildServiceEntry(svc, 3, Options{Tag: "latest"})

	require.Contains(t, cs.Ports, "8083:8080")
}
