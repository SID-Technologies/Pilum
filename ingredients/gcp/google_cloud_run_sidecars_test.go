package gcp_test

import (
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

// flagIndex returns the index of the first occurrence of `flag` in cmd, or
// -1 if absent. Helper for assertions that care about flag ORDER (gcloud
// requires service-level flags before any --container=NAME flag).
func flagIndex(cmd []string, flag string) int {
	for i, arg := range cmd {
		if arg == flag {
			return i
		}
	}

	return -1
}

// containerIndexes returns every index of the `--container` flag in cmd, in
// order. Used to assert there are exactly N+1 container blocks (1 ingress
// + N sidecars) and that they appear after all service-level flags.
func containerIndexes(cmd []string) []int {
	var idxs []int

	for i, arg := range cmd {
		if arg == "--container" {
			idxs = append(idxs, i)
		}
	}

	return idxs
}

// TestGenerateGCPDeployCommand_NoSidecars_PreservesSingleContainerShape locks
// in that adding the multi-container code path is invisible to existing
// services. No --container flags emitted, --image appears at top level.
func TestGenerateGCPDeployCommand_NoSidecars_PreservesSingleContainerShape(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "api",
		Region: "us-central1",
		Config: map[string]any{
			"cloud_run": map[string]any{
				"memory": "512Mi",
				"cpu":    "1",
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/api:v1")

	require.Empty(t, containerIndexes(cmd),
		"single-container deploys must not emit --container flags — flag-ordering would break for existing services")
	require.Contains(t, cmd, "--image", "single-container deploy still emits --image at the top level")
}

// TestGenerateGCPDeployCommand_WithSidecar_EmitsContainerGroups is the
// headline test for the feature: declaring a sidecar must produce a
// multi-container deploy command with one --container group per container.
func TestGenerateGCPDeployCommand_WithSidecar_EmitsContainerGroups(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "mcp-controller-stripe",
		Region: "us-central1",
		Sidecars: []serviceinfo.Sidecar{
			{
				Name:   "otel-collector",
				Image:  "us-docker.pkg.dev/sid-platform/observability/otel-collector:v0.1.0",
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

	cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/mcp-controller:v1")

	indexes := containerIndexes(cmd)
	require.Len(t, indexes, 2, "one --container for the ingress, one for each sidecar")

	require.Equal(t, "mcp-controller-stripe", cmd[indexes[0]+1],
		"ingress container is named after the service so sidecars' depends_on stays predictable")
	require.Equal(t, "otel-collector", cmd[indexes[1]+1])
}

// TestGenerateGCPDeployCommand_ServiceLevelFlagsBeforeFirstContainer locks
// in the gcloud syntax requirement: every service-scoped flag must appear
// before the first --container declaration. Out-of-order flags cause the
// deploy to fail with a flag-parsing error, so this is a contract test.
func TestGenerateGCPDeployCommand_ServiceLevelFlagsBeforeFirstContainer(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "api",
		Region:  "us-central1",
		Project: "my-project",
		Sidecars: []serviceinfo.Sidecar{
			{Name: "logger", Image: "gcr.io/p/logger:v1"},
		},
		Config: map[string]any{
			"cloud_run": map[string]any{
				"min_instances":      1,
				"max_instances":      5,
				"concurrency":        80,
				"timeout_seconds":    300,
				"cpu_throttling":     true,
				"cloudsql_instances": []any{"my-project:us-central1:db1"},
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/api:v1")

	firstContainer := flagIndex(cmd, "--container")
	require.Greater(t, firstContainer, 0, "deploy must include at least one --container flag with sidecars present")

	serviceLevelFlags := []string{
		"--region",
		"--platform",
		"--project",
		"--cpu-throttling",
		"--add-cloudsql-instances",
	}

	for _, flag := range serviceLevelFlags {
		idx := flagIndex(cmd, flag)
		if idx < 0 {
			continue // flag not present in this test config — fine
		}

		require.Less(t, idx, firstContainer,
			"service-level flag %q (idx=%d) must appear BEFORE the first --container (idx=%d); gcloud rejects out-of-order multi-container deploys",
			flag, idx, firstContainer)
	}

	// min/max/concurrency/timeout are positional via =VALUE; assert by
	// substring since they're not standalone tokens.
	for _, prefix := range []string{"--min-instances=", "--max-instances=", "--concurrency=", "--timeout="} {
		for i, arg := range cmd {
			if strings.HasPrefix(arg, prefix) {
				require.Less(t, i, firstContainer,
					"service-level flag %q must appear before --container", prefix)
			}
		}
	}
}

// TestGenerateGCPDeployCommand_IngressContainerHasExplicitPort covers the
// Cloud Run constraint that multi-container revisions have no default port —
// the ingress container's --port must be emitted explicitly or the deploy
// fails with "port required when sidecars present".
func TestGenerateGCPDeployCommand_IngressContainerHasExplicitPort(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "api",
		Region: "us-central1",
		Sidecars: []serviceinfo.Sidecar{
			{Name: "sc", Image: "gcr.io/p/sc:v1"},
		},
		Config: map[string]any{},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/api:v1")

	var portFlag string

	for _, arg := range cmd {
		if strings.HasPrefix(arg, "--port=") {
			portFlag = arg
			break
		}
	}

	require.NotEmpty(t, portFlag, "multi-container deploys must emit --port on the ingress container")
	require.Equal(t, "--port=8080", portFlag, "ingress container defaults to port 8080 matching the historical single-container default")
}

// TestGenerateGCPDeployCommand_CustomIngressPort verifies the port can be
// overridden via cloud_run.port — operators who run their app on something
// other than 8080 need this knob.
func TestGenerateGCPDeployCommand_CustomIngressPort(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "api",
		Region: "us-central1",
		Sidecars: []serviceinfo.Sidecar{
			{Name: "sc", Image: "gcr.io/p/sc:v1"},
		},
		Config: map[string]any{
			"cloud_run": map[string]any{
				"port": 9090,
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/api:v1")

	require.Contains(t, cmd, "--port=9090")
}

// TestGenerateGCPDeployCommand_SidecarFlagsAreScopedToTheirContainer covers
// the subtle gcloud rule: --image, --memory, --set-env-vars after a
// --container apply to THAT container, not the previous one. We rely on
// emitting flags in a specific order; this test pins it down.
func TestGenerateGCPDeployCommand_SidecarFlagsAreScopedToTheirContainer(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "api",
		Region: "us-central1",
		Sidecars: []serviceinfo.Sidecar{
			{
				Name:   "logger",
				Image:  "gcr.io/p/logger:v1",
				Memory: "64Mi",
				EnvVars: []serviceinfo.EnvVars{
					{Name: "LOG_LEVEL", Value: "debug"},
				},
			},
		},
		Config: map[string]any{
			"cloud_run": map[string]any{
				"memory": "1Gi",
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/api:v1")

	indexes := containerIndexes(cmd)
	require.Len(t, indexes, 2)

	ingressStart := indexes[0]
	sidecarStart := indexes[1]

	// The ingress container's memory (1Gi) must live between its --container
	// declaration and the sidecar's --container declaration.
	ingressMemoryIdx := -1

	for i := ingressStart; i < sidecarStart; i++ {
		if cmd[i] == "--memory" && i+1 < sidecarStart && cmd[i+1] == "1Gi" {
			ingressMemoryIdx = i
			break
		}
	}

	require.GreaterOrEqual(t, ingressMemoryIdx, 0,
		"ingress --memory 1Gi must be emitted between the ingress --container and the sidecar --container")

	// The sidecar's memory (64Mi) must live AFTER the sidecar's --container.
	sidecarMemoryIdx := -1

	for i := sidecarStart; i < len(cmd); i++ {
		if cmd[i] == "--memory" && i+1 < len(cmd) && cmd[i+1] == "64Mi" {
			sidecarMemoryIdx = i
			break
		}
	}

	require.GreaterOrEqual(t, sidecarMemoryIdx, 0,
		"sidecar --memory 64Mi must be emitted after the sidecar's --container declaration")

	// The sidecar's env var must also live after the sidecar's --container,
	// not before. gcloud scopes --set-env-vars to the current container.
	sidecarEnvIdx := -1

	for i := sidecarStart; i < len(cmd); i++ {
		if cmd[i] == "--set-env-vars" && i+1 < len(cmd) && strings.Contains(cmd[i+1], "LOG_LEVEL=debug") {
			sidecarEnvIdx = i
			break
		}
	}

	require.GreaterOrEqual(t, sidecarEnvIdx, 0,
		"sidecar's env vars must be scoped under the sidecar's --container")
}

// TestGenerateGCPDeployCommand_DependsOn_RendersFlag covers container-level
// startup ordering. Cloud Run will fail the deploy if a depends-on references
// a container name that isn't declared in the same revision — that's a
// validation concern handled at deploy time, not here.
func TestGenerateGCPDeployCommand_DependsOn_RendersFlag(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "mcp-controller-stripe",
		Region: "us-central1",
		Sidecars: []serviceinfo.Sidecar{
			{
				Name:      "otel-collector",
				Image:     "gcr.io/p/otel:v1",
				DependsOn: []string{"mcp-controller-stripe"},
			},
		},
		Config: map[string]any{},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/mcp:v1")

	require.Contains(t, cmd, "--depends-on=mcp-controller-stripe",
		"sidecar depends_on must render as a --depends-on=NAME flag scoped under its --container group")
}

// TestGenerateGCPDeployCommand_MultipleSidecars handles the >1 sidecar case —
// each gets its own --container group with correctly-scoped flags.
func TestGenerateGCPDeployCommand_MultipleSidecars(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "api",
		Region: "us-central1",
		Sidecars: []serviceinfo.Sidecar{
			{Name: "logger", Image: "gcr.io/p/logger:v1"},
			{Name: "metrics", Image: "gcr.io/p/metrics:v1"},
		},
		Config: map[string]any{},
	}

	cmd := gcp.GenerateGCPDeployCommand(svc, "gcr.io/p/api:v1")

	indexes := containerIndexes(cmd)
	require.Len(t, indexes, 3, "ingress + 2 sidecars = 3 --container groups")

	require.Equal(t, "api", cmd[indexes[0]+1])
	require.Equal(t, "logger", cmd[indexes[1]+1])
	require.Equal(t, "metrics", cmd[indexes[2]+1])
}
