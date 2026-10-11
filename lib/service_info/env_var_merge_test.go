package serviceinfo_test

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func envVarMap(t *testing.T, evs []serviceinfo.EnvVars) map[string]string {
	t.Helper()

	m := make(map[string]string, len(evs))
	for _, ev := range evs {
		m[ev.Name] = ev.Value
	}

	return m
}

func TestEnvVars_TopLevelOnly_PreservesExistingBehavior(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name": "api",
		"env_vars": map[string]any{
			"FOO": "bar",
			"BAZ": "qux",
		},
	}, "/tmp/api")

	got := envVarMap(t, svc.EnvVars)
	require.Equal(t, map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
	}, got)
}

func TestEnvVars_NestedCloudRun_NowIncluded(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name": "api",
		"cloud_run": map[string]any{
			"env_vars": map[string]any{
				"MCP_COMMAND":  "npx server-x",
				"IDLE_TIMEOUT": "10m",
			},
		},
	}, "/tmp/api")

	got := envVarMap(t, svc.EnvVars)
	require.Equal(t, map[string]string{
		"MCP_COMMAND":  "npx server-x",
		"IDLE_TIMEOUT": "10m",
	}, got)
}

func TestEnvVars_BothSources_TopLevelWinsOnConflict(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name": "api",
		"env_vars": map[string]any{
			"MCP_COMMAND": "from-top-level",
		},
		"cloud_run": map[string]any{
			"env_vars": map[string]any{
				"MCP_COMMAND": "from-nested",
				"NESTED_ONLY": "still-included",
			},
		},
	}, "/tmp/api")

	got := envVarMap(t, svc.EnvVars)
	require.Equal(t, "from-top-level", got["MCP_COMMAND"])
	require.Equal(t, "still-included", got["NESTED_ONLY"])
}

func TestEnvVars_NestedContainerApp_AlsoMerged(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name": "api",
		"container_app": map[string]any{
			"env_vars": map[string]any{
				"AZURE_TENANT_ID": "tenant-xyz",
			},
		},
	}, "/tmp/api")

	got := envVarMap(t, svc.EnvVars)
	require.Equal(t, "tenant-xyz", got["AZURE_TENANT_ID"])
}

func TestEnvVars_NumericTopLevelValue_Stringified(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "api",
		"provider": "gcp",
		"env_vars": map[string]any{
			"GOOD": "ok",
			"BAD":  42,
		},
	}, "/tmp/api")

	require.NotNil(t, svc)

	got := envVarMap(t, svc.EnvVars)
	require.Equal(t, "ok", got["GOOD"])
	require.Equal(t, "42", got["BAD"])
	require.NoError(t, svc.Validate())
}

func TestEnvVars_NumericNestedValue_Stringified(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "api",
		"provider": "gcp",
		"cloud_run": map[string]any{
			"env_vars": map[string]any{
				"GOOD": "ok",
				"BAD":  42,
			},
		},
	}, "/tmp/api")

	require.NotNil(t, svc)

	got := envVarMap(t, svc.EnvVars)
	require.Equal(t, "ok", got["GOOD"])
	require.Equal(t, "42", got["BAD"])
	require.NoError(t, svc.Validate())
}

func TestNewServiceInfoMergesMIGEnvVars(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "proxy",
		"type":     "gcp-mig-container",
		"env_vars": map[string]any{"PORT": "3128"},
		"mig": map[string]any{
			"env_vars": map[string]any{"PORT": "9999", "LOG_LEVEL": "info"},
		},
	}, ".")

	require.Equal(t, map[string]string{"PORT": "3128", "LOG_LEVEL": "info"}, envVarMap(t, svc.EnvVars),
		"top-level wins, like cloud_run.env_vars")
}

// A list of {name, value} is read like a map. It used to be dropped, which
// made Cloud Run deploys send --clear-secrets and wipe live secrets.
func TestSecretsAndEnvVars_ListForm(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name":     "api",
		"provider": "gcp",
		"env_vars": []any{
			map[string]any{"name": "LOG_LEVEL", "value": "info"},
			map[string]any{"name": "PORT", "value": 8080},
		},
		"secrets": []any{
			map[string]any{"name": "DB_PASSWORD", "value": "db-password:latest"},
		},
	}, "services/api")

	require.NoError(t, svc.Validate())
	require.Equal(t, map[string]string{"LOG_LEVEL": "info", "PORT": "8080"}, envVarMap(t, svc.EnvVars))
	require.Equal(t, []serviceinfo.Secrets{{Name: "DB_PASSWORD", Value: "db-password:latest"}}, svc.Secrets)
}

// A block Pilum can't read fails validation instead of deploying as if the
// service had no secrets.
func TestSecretsAndEnvVars_UnreadableFailsValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{
			name:   "secrets as a string",
			config: map[string]any{"secrets": "DB_PASSWORD"},
			want:   "secrets: must be a map",
		},
		{
			name:   "secret with no value",
			config: map[string]any{"secrets": map[string]any{"DB_PASSWORD": nil}},
			want:   "secrets: DB_PASSWORD must be a string",
		},
		{
			name:   "list entry without name",
			config: map[string]any{"env_vars": []any{map[string]any{"value": "x"}}},
			want:   "env_vars: entry 1 is missing name",
		},
		{
			name: "nested env var holding a map",
			config: map[string]any{"cloud_run": map[string]any{
				"env_vars": map[string]any{"OPTS": map[string]any{"a": "b"}},
			}},
			want: "cloud_run.env_vars: OPTS must be a string",
		},
		{
			name: "sidecar secrets as a list of strings",
			config: map[string]any{"sidecars": []any{map[string]any{
				"name": "proxy", "image": "proxy:1", "secrets": []any{"TOKEN"},
			}}},
			want: "sidecars.proxy.secrets: entry 1 must be {name, value}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := map[string]any{"name": "api", "provider": "gcp"}
			for k, v := range tt.config {
				config[k] = v
			}

			err := serviceinfo.NewServiceInfo(config, "services/api").Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.want)
		})
	}
}
