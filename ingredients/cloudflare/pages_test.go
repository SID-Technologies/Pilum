package cloudflare

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateInstallCommand_Npm(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{},
	}

	cmd := GenerateInstallCommand(svc)

	require.Equal(t, []string{"npm", "ci"}, cmd)
}

func TestGenerateInstallCommand_Pnpm(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"cloudflare": map[string]any{
				"package_manager": "pnpm",
			},
		},
	}

	cmd := GenerateInstallCommand(svc)

	require.Equal(t, []string{"pnpm", "install", "--frozen-lockfile"}, cmd)
}

func TestGenerateBuildCommand_WithCmd(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		BuildConfig: serviceinfo.BuildConfig{
			Cmd: "npm run build",
		},
	}

	cmd := GenerateBuildCommand(svc)

	require.Equal(t, []string{"/bin/sh", "-c", "npm run build"}, cmd)
}

func TestGenerateBuildCommand_CustomCmd(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		BuildConfig: serviceinfo.BuildConfig{
			Cmd: "pnpm run build:prod",
		},
	}

	cmd := GenerateBuildCommand(svc)

	require.Equal(t, []string{"/bin/sh", "-c", "pnpm run build:prod"}, cmd)
}

func TestGenerateBuildCommand_NoCmd(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{}

	cmd := GenerateBuildCommand(svc)

	require.Nil(t, cmd)
}

func TestGenerateDeployCommand_Production(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
		Config: map[string]any{
			"cloudflare": map[string]any{
				"account_id": "abc123",
			},
		},
	}

	cmd := GenerateDeployCommand(svc, "main")

	require.Contains(t, cmd, "CLOUDFLARE_API_TOKEN")
	require.Contains(t, cmd, "CLOUDFLARE_ACCOUNT_ID='abc123'")
	require.Contains(t, cmd, "--project-name='my-site'")
	require.Contains(t, cmd, "--branch='main'")
	require.Contains(t, cmd, "npx wrangler pages deploy 'dist'")
}

func TestGenerateDeployCommand_LatestTag(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
		Config: map[string]any{
			"cloudflare": map[string]any{
				"account_id": "abc123",
			},
		},
	}

	cmd := GenerateDeployCommand(svc, "latest")

	// "latest" should deploy to production branch
	require.Contains(t, cmd, "--branch='main'")
}

func TestGenerateDeployCommand_Preview(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
		Config: map[string]any{
			"cloudflare": map[string]any{
				"account_id": "abc123",
			},
		},
	}

	cmd := GenerateDeployCommand(svc, "feature-branch")

	require.Contains(t, cmd, "--branch='feature-branch'")
}

func TestGenerateDeployCommand_AccountIDEnv(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
		Config: map[string]any{
			"cloudflare": map[string]any{
				"account_id_env": "CF_ACCT",
			},
		},
	}

	cmd := GenerateDeployCommand(svc, "main")

	// Account ID is read from the named env var, not injected as a literal.
	require.Contains(t, cmd, `export CLOUDFLARE_ACCOUNT_ID="$CF_ACCT"`)
	require.Contains(t, cmd, `if [ -z "$CF_ACCT" ]; then`)
	require.NotContains(t, cmd, "CLOUDFLARE_ACCOUNT_ID=''")
}

func TestGenerateDeployCommand_CustomTokenEnv(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
		Config: map[string]any{
			"cloudflare": map[string]any{
				"account_id": "abc123",
				"token_env":  "MY_CF_TOKEN",
			},
		},
	}

	cmd := GenerateDeployCommand(svc, "main")

	require.Contains(t, cmd, `if [ -z "$MY_CF_TOKEN" ]`)
	require.Contains(t, cmd, "Error: MY_CF_TOKEN environment variable is not set")
	require.Contains(t, cmd, `export CLOUDFLARE_API_TOKEN="$MY_CF_TOKEN"`)
}

func TestGenerateDeployCommand_CustomOutputDir(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
		Config: map[string]any{
			"cloudflare": map[string]any{
				"account_id": "abc123",
			},
			"build": map[string]any{
				"output_dir": "build",
			},
		},
	}

	cmd := GenerateDeployCommand(svc, "main")

	require.Contains(t, cmd, "npx wrangler pages deploy 'build'")
}

func TestGenerateDeployCommand_CustomProductionBranch(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
		Config: map[string]any{
			"cloudflare": map[string]any{
				"account_id":        "abc123",
				"production_branch": "production",
			},
		},
	}

	// Tag matches custom production branch
	cmd := GenerateDeployCommand(svc, "production")
	require.Contains(t, cmd, "--branch='production'")

	// Tag differs from custom production branch -> preview
	cmd = GenerateDeployCommand(svc, "staging")
	require.Contains(t, cmd, "--branch='staging'")
}

func TestGenerateDeployCommand_ValidatesTokenEnv(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-site",
		Config: map[string]any{
			"cloudflare": map[string]any{
				"account_id": "abc123",
			},
		},
	}

	cmd := GenerateDeployCommand(svc, "main")

	require.Contains(t, cmd, `if [ -z "$CLOUDFLARE_API_TOKEN" ]`)
	require.Contains(t, cmd, "Error: CLOUDFLARE_API_TOKEN environment variable is not set")
}
