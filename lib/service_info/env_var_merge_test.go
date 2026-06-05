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

func TestEnvVars_MalformedTopLevelValue_SkippedNotFatal(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name": "api",
		"env_vars": map[string]any{
			"GOOD": "ok",
			"BAD":  42,
		},
	}, "/tmp/api")

	require.NotNil(t, svc)

	got := envVarMap(t, svc.EnvVars)
	require.Equal(t, "ok", got["GOOD"])
	require.NotContains(t, got, "BAD")
}

func TestEnvVars_MalformedNestedValue_SkippedNotFatal(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name": "api",
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
	require.NotContains(t, got, "BAD")
}
