package azure_test

import (
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/ingredients/azure"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateDeployCommandBasic(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config:  map[string]any{},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	require.True(t, strings.HasPrefix(cmd, "az containerapp up "), "must start with idempotent `az containerapp up`, got: %s", cmd)
	require.Contains(t, cmd, "--name 'myapp'")
	require.Contains(t, cmd, "--resource-group 'my-rg'")
	require.Contains(t, cmd, "--image 'myregistry.azurecr.io/myapp:latest'")
}

func TestGenerateDeployCommandEmptyConfigHasNoOptionalFlags(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config:  map[string]any{},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	require.NotContains(t, cmd, "--environment")
	require.NotContains(t, cmd, "--cpu")
	require.NotContains(t, cmd, "--memory")
	require.NotContains(t, cmd, "--target-port")
	require.NotContains(t, cmd, "--ingress")
	require.NotContains(t, cmd, "--min-replicas")
	require.NotContains(t, cmd, "--max-replicas")
	// No resource flags => no update chaser at all.
	require.NotContains(t, cmd, "&&")
	require.NotContains(t, cmd, "containerapp update")
}

func TestGenerateDeployCommandWithEnvironment(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config: map[string]any{
			"container_app": map[string]any{
				"environment": "my-env",
			},
		},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	// --environment belongs to `up` (create-time), never to `update`.
	require.Contains(t, cmd, "--environment 'my-env'")
	upPart := cmd
	if i := strings.Index(cmd, "&&"); i >= 0 {
		upPart = cmd[:i]
	}
	require.Contains(t, upPart, "--environment 'my-env'")
}

func TestGenerateDeployCommandIngressGoesToUpNotUpdate(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config: map[string]any{
			"container_app": map[string]any{
				"ingress_port":     3000,
				"ingress_external": false,
			},
		},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	upPart := cmd
	if i := strings.Index(cmd, "&&"); i >= 0 {
		upPart = cmd[:i]
	}
	// `az containerapp update` rejects these flags — they must be on `up`.
	require.Contains(t, upPart, "--target-port 3000")
	require.Contains(t, upPart, "--ingress internal")
}

func TestGenerateDeployCommandResourceLimitsGoToUpdateChaser(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config: map[string]any{
			"container_app": map[string]any{
				"min_replicas": 0,
				"max_replicas": 5,
				"cpu":          "1",
				"memory":       "2Gi",
			},
		},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	parts := strings.SplitN(cmd, "&&", 2)
	require.Len(t, parts, 2, "resource limits require the update chaser, got: %s", cmd)
	upPart, updatePart := parts[0], parts[1]

	require.Contains(t, updatePart, "az containerapp update")
	require.Contains(t, updatePart, "--name 'myapp'")
	require.Contains(t, updatePart, "--resource-group 'my-rg'")
	require.Contains(t, updatePart, "--min-replicas=0")
	require.Contains(t, updatePart, "--max-replicas=5")
	require.Contains(t, updatePart, "--cpu '1'")
	require.Contains(t, updatePart, "--memory '2Gi'")

	// `up` cannot set resource limits — they must not leak into it.
	require.NotContains(t, upPart, "--cpu")
	require.NotContains(t, upPart, "--min-replicas")
}

func TestGenerateDeployCommandFullConfig(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "prod-rg",
		Config: map[string]any{
			"container_app": map[string]any{
				"environment":      "prod-env",
				"min_replicas":     2,
				"max_replicas":     10,
				"cpu":              "0.5",
				"memory":           "1Gi",
				"ingress_port":     8080,
				"ingress_external": true,
			},
		},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:v1.0.0")

	require.True(t, strings.HasPrefix(cmd, "az containerapp up "))
	require.Contains(t, cmd, "--image 'myregistry.azurecr.io/myapp:v1.0.0'")
	require.Contains(t, cmd, "--environment 'prod-env'")
	require.Contains(t, cmd, "--target-port 8080")
	require.Contains(t, cmd, "--ingress external")
	require.Contains(t, cmd, "&& az containerapp update")
	require.Contains(t, cmd, "--min-replicas=2")
	require.Contains(t, cmd, "--max-replicas=10")
	require.Contains(t, cmd, "--cpu '0.5'")
	require.Contains(t, cmd, "--memory '1Gi'")
}

func TestGenerateDeployCommandQuotesUnsafeValues(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "rg with spaces",
		Config:  map[string]any{},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	require.Contains(t, cmd, "--resource-group 'rg with spaces'")
}
