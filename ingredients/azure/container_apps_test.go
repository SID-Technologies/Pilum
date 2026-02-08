package azure_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/azure"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateDeployCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       serviceinfo.ServiceInfo
		imageName     string
		minLen        int
		containsFlag  string
		containsValue string
	}{
		{
			name: "basic deploy command",
			service: serviceinfo.ServiceInfo{
				Name:    "myapp",
				Project: "my-rg",
				Config:  map[string]any{},
			},
			imageName:     "myregistry.azurecr.io/myapp:latest",
			minLen:        8,
			containsFlag:  "--name",
			containsValue: "myapp",
		},
		{
			name: "deploy with resource group",
			service: serviceinfo.ServiceInfo{
				Name:    "api-service",
				Project: "prod-rg",
				Config:  map[string]any{},
			},
			imageName:     "myregistry.azurecr.io/api-service:v1.0.0",
			minLen:        8,
			containsFlag:  "--resource-group",
			containsValue: "prod-rg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := azure.GenerateDeployCommand(tt.service, tt.imageName)

			require.GreaterOrEqual(t, len(cmd), tt.minLen)
			require.Equal(t, "az", cmd[0])
			require.Equal(t, "containerapp", cmd[1])
			require.Equal(t, "update", cmd[2])

			foundFlag := false
			for i, arg := range cmd {
				if arg == tt.containsFlag && i+1 < len(cmd) {
					require.Equal(t, tt.containsValue, cmd[i+1])
					foundFlag = true
					break
				}
			}
			require.True(t, foundFlag, "expected flag %s not found", tt.containsFlag)
		})
	}
}

func TestGenerateDeployCommandImageName(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config:  map[string]any{},
	}

	imageName := "myregistry.azurecr.io/myapp:v2.0.0"
	cmd := azure.GenerateDeployCommand(service, imageName)

	foundImage := false
	for i, arg := range cmd {
		if arg == "--image" && i+1 < len(cmd) {
			require.Equal(t, imageName, cmd[i+1])
			foundImage = true
			break
		}
	}
	require.True(t, foundImage, "--image flag not found")
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

	foundEnv := false
	for i, arg := range cmd {
		if arg == "--environment" && i+1 < len(cmd) {
			require.Equal(t, "my-env", cmd[i+1])
			foundEnv = true
			break
		}
	}
	require.True(t, foundEnv, "--environment flag not found")
}

func TestGenerateDeployCommandWithReplicas(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config: map[string]any{
			"container_app": map[string]any{
				"min_replicas": 0,
				"max_replicas": 5,
			},
		},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "--min-replicas=0")
	require.Contains(t, cmdStr, "--max-replicas=5")
}

func TestGenerateDeployCommandWithCPUAndMemory(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config: map[string]any{
			"container_app": map[string]any{
				"cpu":    "1",
				"memory": "2Gi",
			},
		},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	foundCPU := false
	foundMemory := false
	for i, arg := range cmd {
		if arg == "--cpu" && i+1 < len(cmd) {
			require.Equal(t, "1", cmd[i+1])
			foundCPU = true
		}
		if arg == "--memory" && i+1 < len(cmd) {
			require.Equal(t, "2Gi", cmd[i+1])
			foundMemory = true
		}
	}
	require.True(t, foundCPU, "--cpu flag not found")
	require.True(t, foundMemory, "--memory flag not found")
}

func TestGenerateDeployCommandWithIngressExternal(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config: map[string]any{
			"container_app": map[string]any{
				"ingress_port":     8080,
				"ingress_external": true,
			},
		},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "--target-port 8080")
	require.Contains(t, cmdStr, "--ingress external")
}

func TestGenerateDeployCommandWithIngressInternal(t *testing.T) {
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

	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "--target-port 3000")
	require.Contains(t, cmdStr, "--ingress internal")
}

func TestGenerateDeployCommandEmptyConfig(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myapp",
		Project: "my-rg",
		Config:  map[string]any{},
	}

	cmd := azure.GenerateDeployCommand(service, "myregistry.azurecr.io/myapp:latest")

	// Should not have any optional flags when config is empty
	for _, arg := range cmd {
		require.NotEqual(t, "--environment", arg)
		require.NotEqual(t, "--cpu", arg)
		require.NotEqual(t, "--memory", arg)
		require.NotEqual(t, "--target-port", arg)
		require.NotEqual(t, "--ingress", arg)
		require.NotContains(t, arg, "--min-replicas")
		require.NotContains(t, arg, "--max-replicas")
	}
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

	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "az containerapp update")
	require.Contains(t, cmdStr, "--name myapp")
	require.Contains(t, cmdStr, "--resource-group prod-rg")
	require.Contains(t, cmdStr, "--image myregistry.azurecr.io/myapp:v1.0.0")
	require.Contains(t, cmdStr, "--environment prod-env")
	require.Contains(t, cmdStr, "--min-replicas=2")
	require.Contains(t, cmdStr, "--max-replicas=10")
	require.Contains(t, cmdStr, "--cpu 0.5")
	require.Contains(t, cmdStr, "--memory 1Gi")
	require.Contains(t, cmdStr, "--target-port 8080")
	require.Contains(t, cmdStr, "--ingress external")
}
