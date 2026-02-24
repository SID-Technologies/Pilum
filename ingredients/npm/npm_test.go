package npm

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateInstallCommand_DefaultNpm(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"npm": map[string]any{},
		},
	}

	cmd := GenerateInstallCommand(svc)

	require.Equal(t, []string{"npm", "ci"}, cmd)
}

func TestGenerateInstallCommand_Pnpm(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"npm": map[string]any{
				"package_manager": "pnpm",
			},
		},
	}

	cmd := GenerateInstallCommand(svc)

	require.Equal(t, []string{"pnpm", "install", "--frozen-lockfile"}, cmd)
}

func TestGenerateInstallCommand_Yarn(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"npm": map[string]any{
				"package_manager": "yarn",
			},
		},
	}

	cmd := GenerateInstallCommand(svc)

	require.Equal(t, []string{"yarn", "install", "--frozen-lockfile"}, cmd)
}

func TestGenerateInstallCommand_Bun(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"npm": map[string]any{
				"package_manager": "bun",
			},
		},
	}

	cmd := GenerateInstallCommand(svc)

	require.Equal(t, []string{"bun", "install", "--frozen-lockfile"}, cmd)
}

func TestGenerateInstallCommand_EmptyConfig(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{},
	}

	cmd := GenerateInstallCommand(svc)

	require.Equal(t, []string{"npm", "ci"}, cmd, "should default to npm when no config")
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

func TestGenerateResolveWorkspacesCommand_Pnpm(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"npm": map[string]any{
				"package_manager": "pnpm",
			},
		},
	}

	cmd := GenerateResolveWorkspacesCommand(svc)

	require.NotEmpty(t, cmd, "should generate a script for pnpm")
	require.Contains(t, cmd, "workspace:")
	require.Contains(t, cmd, "package.json")
	require.Contains(t, cmd, "Resolved")
}

func TestGenerateResolveWorkspacesCommand_NonPnpm(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{
			"npm": map[string]any{
				"package_manager": "npm",
			},
		},
	}

	cmd := GenerateResolveWorkspacesCommand(svc)

	require.Empty(t, cmd, "should return empty string for non-pnpm")
}

func TestGenerateResolveWorkspacesCommand_DefaultManager(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Config: map[string]any{},
	}

	cmd := GenerateResolveWorkspacesCommand(svc)

	require.Empty(t, cmd, "should return empty string when no package manager specified")
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
	require.Contains(t, cmd, "npm publish --access 'public'")
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
	require.Contains(t, cmd, "npm publish --access 'restricted'")
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
	require.Contains(t, cmd, "npm publish --access 'restricted'")
	require.Contains(t, cmd, "npm.pkg.github.com")
}
