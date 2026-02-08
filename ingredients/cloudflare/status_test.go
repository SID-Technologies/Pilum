package cloudflare_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/cloudflare"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateStatusCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
	}

	cmd := cloudflare.GenerateStatusCommand(svc)

	require.Equal(t, "npx", cmd[0])
	require.Equal(t, "wrangler", cmd[1])
	require.Equal(t, "pages", cmd[2])
	require.Contains(t, cmd, "--project-name")
	require.Contains(t, cmd, "my-site")
}

func TestGenerateLogsCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
	}

	cmd := cloudflare.GenerateLogsCommand(svc)

	require.Equal(t, "npx", cmd[0])
	require.Equal(t, "wrangler", cmd[1])
	require.Contains(t, cmd, "--project-name")
	require.Contains(t, cmd, "my-site")
}
