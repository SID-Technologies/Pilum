package gcp_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateDeployJobCommand(t *testing.T) {
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
			name: "basic deploy job command",
			service: serviceinfo.ServiceInfo{
				Name:   "migrations",
				Region: "us-central1",
				Config: map[string]any{},
			},
			imageName:     "us-docker.pkg.dev/project/repo/migrations:latest",
			minLen:        9,
			expectedFirst: "gcloud",
			containsFlag:  "--region",
			containsValue: "us-central1",
		},
		{
			name: "deploy job with different region",
			service: serviceinfo.ServiceInfo{
				Name:   "data-export",
				Region: "europe-west1",
				Config: map[string]any{},
			},
			imageName:     "us-docker.pkg.dev/project/repo/data-export:v1.0.0",
			minLen:        9,
			expectedFirst: "gcloud",
			containsFlag:  "--region",
			containsValue: "europe-west1",
		},
		{
			name: "deploy job with project",
			service: serviceinfo.ServiceInfo{
				Name:    "migrations",
				Region:  "us-central1",
				Project: "my-project",
				Config:  map[string]any{},
			},
			imageName:     "us-docker.pkg.dev/project/repo/migrations:latest",
			minLen:        11,
			expectedFirst: "gcloud",
			containsFlag:  "--project",
			containsValue: "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := gcp.GenerateDeployJobCommand(tt.service, tt.imageName)

			require.GreaterOrEqual(t, len(cmd), tt.minLen)
			require.Equal(t, tt.expectedFirst, cmd[0])
			require.Equal(t, "run", cmd[1])
			require.Equal(t, "jobs", cmd[2])
			require.Equal(t, "deploy", cmd[3])
			require.Equal(t, tt.service.Name, cmd[4])

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

func TestGenerateDeployJobCommandAllConfig(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "batch-job",
		Region:  "us-central1",
		Project: "my-project",
		Config: map[string]any{
			"job": map[string]any{
				"timeout":         "300s",
				"max_retries":     3,
				"parallelism":     4,
				"task_count":      10,
				"cpu":             "2",
				"memory":          "2Gi",
				"service_account": "sa@proj.iam.gserviceaccount.com",
			},
		},
	}

	cmd := gcp.GenerateDeployJobCommand(service, "gcr.io/project/batch-job:latest")

	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "--task-timeout 300s")
	require.Contains(t, cmdStr, "--max-retries=3")
	require.Contains(t, cmdStr, "--parallelism=4")
	require.Contains(t, cmdStr, "--tasks=10")
	require.Contains(t, cmdStr, "--cpu 2")
	require.Contains(t, cmdStr, "--memory 2Gi")
	require.Contains(t, cmdStr, "--service-account sa@proj.iam.gserviceaccount.com")
	require.Contains(t, cmdStr, "--project my-project")
}

func TestGenerateDeployJobCommandEmptyConfig(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "migrations",
		Region: "us-central1",
		Config: map[string]any{},
	}

	cmd := gcp.GenerateDeployJobCommand(service, "gcr.io/project/migrations:latest")

	// Should have default timeout but no job-specific flags
	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}

	require.Contains(t, cmdStr, "--task-timeout 600s")
	require.NotContains(t, cmdStr, "--max-retries")
	require.NotContains(t, cmdStr, "--parallelism")
	require.NotContains(t, cmdStr, "--tasks=")
	require.NotContains(t, cmdStr, "--service-account")
}

func TestGenerateDeployJobCommandWithSecrets(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "migrations",
		Region: "us-central1",
		Config: map[string]any{},
		Secrets: []serviceinfo.Secrets{
			{Name: "DB_PASSWORD", Value: "projects/123/secrets/db-pass:latest"},
			{Name: "API_KEY", Value: "projects/123/secrets/api-key:latest"},
		},
	}

	cmd := gcp.GenerateDeployJobCommand(service, "gcr.io/project/migrations:latest")

	foundSecrets := false
	for i, arg := range cmd {
		if arg == "--set-secrets" && i+1 < len(cmd) {
			secretsValue := cmd[i+1]
			require.Contains(t, secretsValue, "DB_PASSWORD=")
			require.Contains(t, secretsValue, "API_KEY=")
			require.Contains(t, secretsValue, ",")
			foundSecrets = true
			break
		}
	}
	require.True(t, foundSecrets, "--set-secrets flag not found")
}

func TestGenerateDeployJobCommandNoSecrets(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "migrations",
		Region:  "us-central1",
		Config:  map[string]any{},
		Secrets: []serviceinfo.Secrets{},
	}

	cmd := gcp.GenerateDeployJobCommand(service, "gcr.io/project/migrations:latest")

	for _, arg := range cmd {
		require.NotEqual(t, "--set-secrets", arg)
	}
}

func TestGenerateDeployJobCommandSingleSecret(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "migrations",
		Region: "us-central1",
		Config: map[string]any{},
		Secrets: []serviceinfo.Secrets{
			{Name: "DATABASE_URL", Value: "projects/123/secrets/db-url:latest"},
		},
	}

	cmd := gcp.GenerateDeployJobCommand(service, "gcr.io/project/migrations:latest")

	for i, arg := range cmd {
		if arg == "--set-secrets" && i+1 < len(cmd) {
			require.Equal(t, "DATABASE_URL=projects/123/secrets/db-url:latest", cmd[i+1])
			return
		}
	}
	t.Fatal("--set-secrets flag not found")
}

func TestGenerateExecuteJobCommandEnabled(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "migrations",
		Region:  "us-central1",
		Project: "my-project",
		Config: map[string]any{
			"job": map[string]any{
				"execute_on_deploy": true,
			},
		},
	}

	cmd := gcp.GenerateExecuteJobCommand(service)

	require.NotNil(t, cmd)
	require.Equal(t, "gcloud", cmd[0])
	require.Equal(t, "run", cmd[1])
	require.Equal(t, "jobs", cmd[2])
	require.Equal(t, "execute", cmd[3])
	require.Equal(t, "migrations", cmd[4])

	cmdStr := ""
	for _, arg := range cmd {
		cmdStr += arg + " "
	}
	require.Contains(t, cmdStr, "--region us-central1")
	require.Contains(t, cmdStr, "--wait")
	require.Contains(t, cmdStr, "--project my-project")
}

func TestGenerateExecuteJobCommandDisabled(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "migrations",
		Region: "us-central1",
		Config: map[string]any{
			"job": map[string]any{
				"execute_on_deploy": false,
			},
		},
	}

	cmd := gcp.GenerateExecuteJobCommand(service)
	require.Nil(t, cmd)
}

func TestGenerateExecuteJobCommandDefaultDisabled(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "migrations",
		Region: "us-central1",
		Config: map[string]any{},
	}

	cmd := gcp.GenerateExecuteJobCommand(service)
	require.Nil(t, cmd)
}

func TestGenerateExecuteJobCommandNoProject(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:   "migrations",
		Region: "us-central1",
		Config: map[string]any{
			"job": map[string]any{
				"execute_on_deploy": true,
			},
		},
	}

	cmd := gcp.GenerateExecuteJobCommand(service)

	require.NotNil(t, cmd)
	for _, arg := range cmd {
		require.NotEqual(t, "--project", arg)
	}
}
