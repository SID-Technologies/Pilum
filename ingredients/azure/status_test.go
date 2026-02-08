package azure_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/azure"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateStatusCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "my-app",
		Project: "my-resource-group",
	}

	cmd := azure.GenerateStatusCommand(svc)

	require.Equal(t, []string{
		"az", "containerapp", "show",
		"--name", "my-app",
		"--resource-group", "my-resource-group",
		"--output", "json",
	}, cmd)
}

func TestGenerateLogsCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "my-app",
		Project: "my-resource-group",
	}

	cmd := azure.GenerateLogsCommand(svc, false)

	require.Equal(t, []string{
		"az", "containerapp", "logs", "show",
		"--name", "my-app",
		"--resource-group", "my-resource-group",
		"--type", "console",
	}, cmd)
}

func TestGenerateLogsCommandFollow(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "my-app",
		Project: "my-resource-group",
	}

	cmd := azure.GenerateLogsCommand(svc, true)

	require.Contains(t, cmd, "--follow")
}
