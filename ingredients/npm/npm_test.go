package npm

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateInstallCommand(t *testing.T) {
	t.Parallel()

	cmd := GenerateInstallCommand()

	require.Equal(t, []string{"npm", "ci"}, cmd)
}

func TestGenerateSetVersionCommand_StripsVPrefix(t *testing.T) {
	t.Parallel()

	cmd := GenerateSetVersionCommand("v1.2.3")

	require.Equal(t, []string{"npm", "version", "1.2.3", "--no-git-tag-version", "--allow-same-version"}, cmd)
}

func TestGenerateSetVersionCommand_NoPrefix(t *testing.T) {
	t.Parallel()

	cmd := GenerateSetVersionCommand("1.2.3")

	require.Equal(t, []string{"npm", "version", "1.2.3", "--no-git-tag-version", "--allow-same-version"}, cmd)
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

func TestGenerateBuildCommand_NoCmd(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{}

	cmd := GenerateBuildCommand(svc)

	require.Nil(t, cmd)
}

func TestGeneratePublishCommand_FullConfig(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"npm": map[string]any{
				"scope":     "@myorg",
				"registry":  "https://registry.npmjs.org",
				"token_env": "NPM_TOKEN",
				"access":    "public",
			},
		},
	}

	cmd := GeneratePublishCommand(svc)

	require.Contains(t, cmd, "NPM_TOKEN")
	require.Contains(t, cmd, "//registry.npmjs.org/:_authToken=$NPM_TOKEN")
	require.Contains(t, cmd, "@myorg:registry=https://registry.npmjs.org")
	require.Contains(t, cmd, "npm publish --access public")
}

func TestGeneratePublishCommand_Defaults(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"npm": map[string]any{
				"scope":     "@sid-technologies",
				"token_env": "NODE_AUTH_TOKEN",
			},
		},
	}

	cmd := GeneratePublishCommand(svc)

	require.Contains(t, cmd, "NODE_AUTH_TOKEN")
	require.Contains(t, cmd, "//npm.pkg.github.com/:_authToken=$NODE_AUTH_TOKEN")
	require.Contains(t, cmd, "@sid-technologies:registry=https://npm.pkg.github.com")
	require.Contains(t, cmd, "npm publish --access restricted")
}

func TestGeneratePublishCommand_ValidatesTokenEnv(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"npm": map[string]any{
				"scope":     "@myorg",
				"token_env": "MY_TOKEN",
			},
		},
	}

	cmd := GeneratePublishCommand(svc)

	require.Contains(t, cmd, `if [ -z "$MY_TOKEN" ]`)
	require.Contains(t, cmd, "Error: MY_TOKEN environment variable is not set")
}

func TestGeneratePublishCommand_EmptyConfig(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{},
	}

	cmd := GeneratePublishCommand(svc)

	// Should still produce a valid script with defaults
	require.Contains(t, cmd, "npm publish --access restricted")
	require.Contains(t, cmd, "npm.pkg.github.com")
}
