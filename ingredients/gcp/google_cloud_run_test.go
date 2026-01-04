package gcp_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateGCPDeployCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       serviceinfo.ServiceInfo
		imageName     string
		minLen        int
		expectedFirst string
		containsFlag  string
		containsValue string
	}{
		{
			name: "basic deploy command",
			service: serviceinfo.ServiceInfo{
				Name:   "myservice",
				Region: "us-central1",
				Config: map[string]any{}, // Empty config means no cloud_run settings
			},
			imageName:     "gcr.io/project/myservice:latest",
			minLen:        11,
			expectedFirst: "gcloud",
			containsFlag:  "--region",
			containsValue: "us-central1",
		},
		{
			name: "deploy with different region",
			service: serviceinfo.ServiceInfo{
				Name:   "api-service",
				Region: "europe-west1",
				Config: map[string]any{}, // Empty config means no cloud_run settings
			},
			imageName:     "gcr.io/project/api-service:v1.0.0",
			minLen:        11,
			expectedFirst: "gcloud",
			containsFlag:  "--region",
			containsValue: "europe-west1",
		},
		{
			name: "deploy with project",
			service: serviceinfo.ServiceInfo{
				Name:    "myservice",
				Region:  "us-central1",
				Project: "my-project",
			},
			imageName:     "gcr.io/project/myservice:latest",
			minLen:        13, // 11 + 2 for --project value
			expectedFirst: "gcloud",
			containsFlag:  "--project",
			containsValue: "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := gcp.GenerateGCPDeployCommand(tt.service, tt.imageName)

			require.GreaterOrEqual(t, len(cmd), tt.minLen)
			require.Equal(t, tt.expectedFirst, cmd[0])
			require.Equal(t, "run", cmd[1])
			require.Equal(t, "deploy", cmd[2])
			require.Equal(t, tt.service.Name, cmd[3])

			// Check flag and value are present
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

func TestGenerateGCPDeployCommandWithSecrets(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		Secrets: []serviceinfo.Secrets{
			{Name: "DB_PASSWORD", Value: "projects/123/secrets/db-pass:latest"},
			{Name: "API_KEY", Value: "projects/123/secrets/api-key:latest"},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should have --set-secrets flag
	foundSecrets := false
	for i, arg := range cmd {
		if arg == "--set-secrets" && i+1 < len(cmd) {
			secretsValue := cmd[i+1]
			require.Contains(t, secretsValue, "DB_PASSWORD=")
			require.Contains(t, secretsValue, "API_KEY=")
			require.Contains(t, secretsValue, ",") // Multiple secrets joined
			foundSecrets = true
			break
		}
	}
	require.True(t, foundSecrets, "--set-secrets flag not found")
}

func TestGenerateGCPDeployCommandNoSecrets(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myservice",
		Region:  "us-central1",
		Config:  map[string]any{},
		Secrets: []serviceinfo.Secrets{},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should NOT have --set-secrets flag
	for _, arg := range cmd {
		require.NotEqual(t, "--set-secrets", arg)
	}
}

func TestGenerateGCPDeployCommandImageName(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
	}

	imageName := "us-central1-docker.pkg.dev/my-project/repo/myservice:v2.0.0"
	cmd := gcp.GenerateGCPDeployCommand(service, imageName)

	// Check --image flag
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

func TestGenerateGCPDeployCommandPlatformManaged(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should have --platform managed
	foundPlatform := false
	for i, arg := range cmd {
		if arg == "--platform" && i+1 < len(cmd) {
			require.Equal(t, "managed", cmd[i+1])
			foundPlatform = true
			break
		}
	}
	require.True(t, foundPlatform, "--platform flag not found")
}

func TestGenerateGCPDeployCommandAllowUnauthenticated(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should have --allow-unauthenticated
	found := false
	for _, arg := range cmd {
		if arg == "--allow-unauthenticated" {
			found = true
			break
		}
	}
	require.True(t, found, "--allow-unauthenticated flag not found")
}

func TestGenerateGCPDeployCommandSingleSecret(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{},
		Secrets: []serviceinfo.Secrets{
			{Name: "API_KEY", Value: "projects/123/secrets/key:latest"},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should have --set-secrets flag with single secret
	for i, arg := range cmd {
		if arg == "--set-secrets" && i+1 < len(cmd) {
			secretsValue := cmd[i+1]
			require.Equal(t, "API_KEY=projects/123/secrets/key:latest", secretsValue)
			return
		}
	}
	t.Fatal("--set-secrets flag not found")
}

func TestGenerateGCPDeployCommandCloudRunConfig(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myservice",
		Region:  "us-central1",
		Project: "my-project",
		Config: map[string]any{
			"cloud_run": map[string]any{
				"min_instances":   0,
				"max_instances":   10,
				"cpu_throttling":  true,
				"memory":          "2048Mi",
				"cpu":             "2",
				"concurrency":     80,
				"timeout_seconds": 300,
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Check all Cloud Run config flags are present
	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "--min-instances=0")
	require.Contains(t, cmdStr, "--max-instances=10")
	require.Contains(t, cmdStr, "--cpu-throttling")
	require.Contains(t, cmdStr, "--memory 2048Mi")
	require.Contains(t, cmdStr, "--cpu 2")
	require.Contains(t, cmdStr, "--concurrency=80")
	require.Contains(t, cmdStr, "--timeout=300")
	require.Contains(t, cmdStr, "--project my-project")
}

func TestGenerateGCPDeployCommandCPUThrottlingDisabled(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{
			"cloud_run": map[string]any{
				"cpu_throttling": false,
			},
		},
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should NOT have --cpu-throttling when set to false
	for _, arg := range cmd {
		require.NotEqual(t, "--cpu-throttling", arg)
	}
}

func TestGenerateGCPDeployCommandEmptyCloudRunConfig(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "myservice",
		Region: "us-central1",
		Config: map[string]any{}, // No cloud_run section
	}

	cmd := gcp.GenerateGCPDeployCommand(service, "gcr.io/project/myservice:latest")

	// Should not have any Cloud Run specific flags when config is empty
	for _, arg := range cmd {
		require.NotContains(t, arg, "--min-instances")
		require.NotContains(t, arg, "--max-instances")
		require.NotEqual(t, "--cpu-throttling", arg)
		require.NotEqual(t, "--memory", arg)
		require.NotEqual(t, "--cpu", arg)
		require.NotContains(t, arg, "--concurrency")
		require.NotContains(t, arg, "--timeout")
	}
}
