package npm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfigEmpty(t *testing.T) {
	t.Parallel()

	cfg := ParseConfig(map[string]any{})

	require.Equal(t, DefaultRegistry, cfg.Registry)
	require.Equal(t, DefaultAccess, cfg.Access)
	require.Equal(t, "npm", cfg.PackageManager)
	require.Empty(t, cfg.Scope)
	require.Empty(t, cfg.TokenEnv)
}

func TestParseConfigFull(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"npm": map[string]any{
			"scope":           "@myorg",
			"registry":        "https://registry.npmjs.org",
			"token_env":       "NPM_TOKEN",
			"access":          "public",
			"package_manager": "pnpm",
		},
	}

	cfg := ParseConfig(config)

	require.Equal(t, "@myorg", cfg.Scope)
	require.Equal(t, "https://registry.npmjs.org", cfg.Registry)
	require.Equal(t, "NPM_TOKEN", cfg.TokenEnv)
	require.Equal(t, "public", cfg.Access)
	require.Equal(t, "pnpm", cfg.PackageManager)
}

func TestParseConfigPartial(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"npm": map[string]any{
			"scope":     "@sid-technologies",
			"token_env": "GITHUB_TOKEN",
		},
	}

	cfg := ParseConfig(config)

	require.Equal(t, "@sid-technologies", cfg.Scope)
	require.Equal(t, "GITHUB_TOKEN", cfg.TokenEnv)
	// Defaults
	require.Equal(t, DefaultRegistry, cfg.Registry)
	require.Equal(t, DefaultAccess, cfg.Access)
}

func TestParseConfigNil(t *testing.T) {
	t.Parallel()

	cfg := ParseConfig(nil)

	require.Equal(t, DefaultRegistry, cfg.Registry)
	require.Equal(t, DefaultAccess, cfg.Access)
	require.Empty(t, cfg.Scope)
	require.Empty(t, cfg.TokenEnv)
}
