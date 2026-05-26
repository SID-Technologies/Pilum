package serviceinfo_test

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

// envVarMap turns []EnvVars into a map for order-independent assertions.
// We use a map because the merge step uses a map internally; iteration
// order isn't stable.
func envVarMap(t *testing.T, evs []serviceinfo.EnvVars) map[string]string {
	t.Helper()

	m := make(map[string]string, len(evs))
	for _, ev := range evs {
		m[ev.Name] = ev.Value
	}

	return m
}

// TestEnvVars_TopLevelOnly_PreservesExistingBehavior — services that already
// declared env vars only at the top level must keep working unchanged.
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

// TestEnvVars_NestedCloudRun_NowIncluded — the bug fix's headline test. A
// pilum.yaml with env vars ONLY under cloud_run.env_vars used to drop them
// silently from real deploys. They must now show up in svc.EnvVars so every
// downstream reader (compose generator AND gcp deploy command builder) sees
// the same set.
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
	}, got, "nested cloud_run.env_vars must surface in svc.EnvVars so deploy paths see them")
}

// TestEnvVars_BothSources_TopLevelWinsOnConflict locks the precedence rule.
// Top-level is the canonical / portable location, so it overrides nested.
// This matters when migrating a pilum.yaml from nested to top-level: leaving
// the nested version in place during the transition can't surprise-revert
// the new top-level value.
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
	require.Equal(t, "from-top-level", got["MCP_COMMAND"],
		"top-level env_vars must win when both locations set the same key")
	require.Equal(t, "still-included", got["NESTED_ONLY"],
		"nested-only keys must survive even when other keys are overridden by top-level")
}

// TestEnvVars_NestedContainerApp_AlsoMerged covers the Azure equivalent —
// the same divergence exists in principle for any target-specific nested
// block. We merge all known ones at parse time to prevent the bug class
// from recurring per-target.
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
	require.Equal(t, "tenant-xyz", got["AZURE_TENANT_ID"],
		"nested container_app.env_vars must merge the same way as cloud_run.env_vars")
}

// TestEnvVars_MalformedTopLevelValue_SkippedNotFatal — non-string env values
// are skipped rather than fatal. The previous strict behavior returned nil
// from NewServiceInfo, which would have panicked the production caller in
// get_services.go that dereferences svc.Name without a nil check. So this
// is both safer AND less surprising.
func TestEnvVars_MalformedTopLevelValue_SkippedNotFatal(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name": "api",
		"env_vars": map[string]any{
			"GOOD": "ok",
			"BAD":  42, // int, not string
		},
	}, "/tmp/api")

	require.NotNil(t, svc, "malformed top-level env value must NOT panic / nil out the whole service")

	got := envVarMap(t, svc.EnvVars)
	require.Equal(t, "ok", got["GOOD"])
	require.NotContains(t, got, "BAD", "non-string top-level values must be skipped, not coerced")
}

// TestEnvVars_MalformedNestedValue_SkippedNotFatal — nested env vars are
// the legacy/migration path; one bad value shouldn't kill the whole service
// (less strict than top-level, by design). The good entries still come
// through.
func TestEnvVars_MalformedNestedValue_SkippedNotFatal(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.NewServiceInfo(map[string]any{
		"name": "api",
		"cloud_run": map[string]any{
			"env_vars": map[string]any{
				"GOOD": "ok",
				"BAD":  42, // int, not string
			},
		},
	}, "/tmp/api")

	require.NotNil(t, svc, "malformed nested env value must NOT abort parsing")

	got := envVarMap(t, svc.EnvVars)
	require.Equal(t, "ok", got["GOOD"])
	require.NotContains(t, got, "BAD", "non-string nested values must be skipped, not coerced")
}
