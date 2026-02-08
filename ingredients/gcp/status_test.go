package gcp_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateStatusCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "api-gateway",
		Region:  "us-central1",
		Project: "my-project",
	}

	cmd := gcp.GenerateStatusCommand(svc)

	require.Equal(t, "gcloud", cmd[0])
	require.Equal(t, "run", cmd[1])
	require.Equal(t, "services", cmd[2])
	require.Equal(t, "describe", cmd[3])
	require.Equal(t, "api-gateway", cmd[4])
	require.Contains(t, cmd, "--region")
	require.Contains(t, cmd, "us-central1")
	require.Contains(t, cmd, "--format")
	require.Contains(t, cmd, "json")
	require.Contains(t, cmd, "--project")
	require.Contains(t, cmd, "my-project")
}

func TestGenerateStatusCommandNoProject(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "api-gateway",
		Region: "us-central1",
	}

	cmd := gcp.GenerateStatusCommand(svc)

	require.NotContains(t, cmd, "--project")
}

func TestGenerateJobStatusCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "data-processor",
		Region:  "europe-west1",
		Project: "my-project",
	}

	cmd := gcp.GenerateJobStatusCommand(svc)

	require.Equal(t, "gcloud", cmd[0])
	require.Equal(t, "run", cmd[1])
	require.Equal(t, "jobs", cmd[2])
	require.Equal(t, "describe", cmd[3])
	require.Equal(t, "data-processor", cmd[4])
	require.Contains(t, cmd, "--region")
	require.Contains(t, cmd, "europe-west1")
	require.Contains(t, cmd, "--project")
}

func TestGenerateLogsCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "api-gateway",
		Region:  "us-central1",
		Project: "my-project",
	}

	cmd := gcp.GenerateLogsCommand(svc, 100, false)

	require.Equal(t, "gcloud", cmd[0])
	require.Equal(t, "run", cmd[1])
	require.Equal(t, "services", cmd[2])
	require.Equal(t, "logs", cmd[3])
	require.Equal(t, "read", cmd[4])
	require.Equal(t, "api-gateway", cmd[5])
	require.Contains(t, cmd, "--limit=100")
	require.Contains(t, cmd, "--project")
}

func TestGenerateLogsCommandNoLines(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "api-gateway",
		Region: "us-central1",
	}

	cmd := gcp.GenerateLogsCommand(svc, 0, false)

	for _, arg := range cmd {
		require.NotContains(t, arg, "--limit")
	}
}

func TestGenerateJobLogsCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "data-processor",
		Region:  "us-central1",
		Project: "my-project",
	}

	cmd := gcp.GenerateJobLogsCommand(svc, 50, false)

	require.Equal(t, "gcloud", cmd[0])
	require.Equal(t, "run", cmd[1])
	require.Equal(t, "jobs", cmd[2])
	require.Equal(t, "executions", cmd[3])
	require.Equal(t, "list", cmd[4])
	require.Contains(t, cmd, "--job")
	require.Contains(t, cmd, "data-processor")
	require.Contains(t, cmd, "--limit=50")
}
