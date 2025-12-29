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
		name           string
		service        serviceinfo.ServiceInfo
		imageName      string
		expectedLen    int
		expectedFirst  string
		containsFlag   string
		containsValue  string
	}{
		{
			name: "basic deploy command",
			service: serviceinfo.ServiceInfo{
				Name:   "myservice",
				Region: "us-central1",
			},
			imageName:     "gcr.io/project/myservice:latest",
			expectedLen:   11,
			expectedFirst: "gcloud",
			containsFlag:  "--region",
			containsValue: "us-central1",
		},
		{
			name: "deploy with different region",
			service: serviceinfo.ServiceInfo{
				Name:   "api-service",
				Region: "europe-west1",
			},
			imageName:     "gcr.io/project/api-service:v1.0.0",
			expectedLen:   11,
			expectedFirst: "gcloud",
			containsFlag:  "--region",
			containsValue: "europe-west1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := gcp.GenerateGCPDeployCommand(tt.service, tt.imageName)

			require.Len(t, cmd, tt.expectedLen)
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
