package gcp_test

import (
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func flagIndex(cmd []string, flag string) int {
	for i, arg := range cmd {
		if arg == flag {
			return i
		}
	}

	return -1
}

func containerIndexes(cmd []string) []int {
	var idxs []int

	for i, arg := range cmd {
		if arg == "--container" {
			idxs = append(idxs, i)
		}
	}

	return idxs
}

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

	require.Empty(t, containerIndexes(cmd))
	require.Contains(t, cmd, "--image")
}

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
	require.Len(t, indexes, 2)
	require.Equal(t, "mcp-controller-stripe", cmd[indexes[0]+1])
	require.Equal(t, "otel-collector", cmd[indexes[1]+1])
}

// gcloud rejects out-of-order multi-container deploys, so service-level
// flags must appear before the first --container.
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
	require.Greater(t, firstContainer, 0)

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
			continue
		}

		require.Less(t, idx, firstContainer, "flag %q must appear before --container", flag)
	}

	for _, prefix := range []string{"--min-instances=", "--max-instances=", "--concurrency=", "--timeout="} {
		for i, arg := range cmd {
			if strings.HasPrefix(arg, prefix) {
				require.Less(t, i, firstContainer, "flag %q must appear before --container", prefix)
			}
		}
	}
}

// Multi-container revisions have no default port, so --port must be explicit.
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

	require.Equal(t, "--port=8080", portFlag)
}

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

// gcloud scopes container-level flags to the most-recent --container.
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

	ingressMemoryIdx := -1

	for i := ingressStart; i < sidecarStart; i++ {
		if cmd[i] == "--memory" && i+1 < sidecarStart && cmd[i+1] == "1Gi" {
			ingressMemoryIdx = i
			break
		}
	}

	require.GreaterOrEqual(t, ingressMemoryIdx, 0)

	sidecarMemoryIdx := -1

	for i := sidecarStart; i < len(cmd); i++ {
		if cmd[i] == "--memory" && i+1 < len(cmd) && cmd[i+1] == "64Mi" {
			sidecarMemoryIdx = i
			break
		}
	}

	require.GreaterOrEqual(t, sidecarMemoryIdx, 0)

	sidecarEnvIdx := -1

	for i := sidecarStart; i < len(cmd); i++ {
		if cmd[i] == "--set-env-vars" && i+1 < len(cmd) && strings.Contains(cmd[i+1], "LOG_LEVEL=debug") {
			sidecarEnvIdx = i
			break
		}
	}

	require.GreaterOrEqual(t, sidecarEnvIdx, 0)
}

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

	require.Contains(t, cmd, "--depends-on=mcp-controller-stripe")
}

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
	require.Len(t, indexes, 3)

	require.Equal(t, "api", cmd[indexes[0]+1])
	require.Equal(t, "logger", cmd[indexes[1]+1])
	require.Equal(t, "metrics", cmd[indexes[2]+1])
}
