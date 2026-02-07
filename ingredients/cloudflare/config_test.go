package cloudflare

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfigEmpty(t *testing.T) {
	t.Parallel()

	cfg := ParseConfig(map[string]any{})

	require.Equal(t, DefaultTokenEnv, cfg.TokenEnv)
	require.Equal(t, DefaultProductionBranch, cfg.ProductionBranch)
	require.Equal(t, DefaultPackageManager, cfg.PackageManager)
	require.Equal(t, DefaultOutputDir, cfg.OutputDir)
	require.Empty(t, cfg.AccountID)
}

func TestParseConfigFull(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"cloudflare": map[string]any{
			"account_id":        "abc123",
			"token_env":         "MY_CF_TOKEN",
			"production_branch": "production",
			"package_manager":   "pnpm",
		},
		"build": map[string]any{
			"output_dir": "build",
		},
	}

	cfg := ParseConfig(config)

	require.Equal(t, "abc123", cfg.AccountID)
	require.Equal(t, "MY_CF_TOKEN", cfg.TokenEnv)
	require.Equal(t, "production", cfg.ProductionBranch)
	require.Equal(t, "pnpm", cfg.PackageManager)
	require.Equal(t, "build", cfg.OutputDir)
}

func TestParseConfigPartial(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"cloudflare": map[string]any{
			"account_id": "abc123",
		},
	}

	cfg := ParseConfig(config)

	require.Equal(t, "abc123", cfg.AccountID)
	// Defaults
	require.Equal(t, DefaultTokenEnv, cfg.TokenEnv)
	require.Equal(t, DefaultProductionBranch, cfg.ProductionBranch)
	require.Equal(t, DefaultPackageManager, cfg.PackageManager)
	require.Equal(t, DefaultOutputDir, cfg.OutputDir)
}

func TestParseConfigNil(t *testing.T) {
	t.Parallel()

	cfg := ParseConfig(nil)

	require.Equal(t, DefaultTokenEnv, cfg.TokenEnv)
	require.Equal(t, DefaultProductionBranch, cfg.ProductionBranch)
	require.Equal(t, DefaultPackageManager, cfg.PackageManager)
	require.Equal(t, DefaultOutputDir, cfg.OutputDir)
	require.Empty(t, cfg.AccountID)
}
