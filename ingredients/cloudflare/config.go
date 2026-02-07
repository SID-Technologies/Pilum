package cloudflare

import (
	"github.com/sid-technologies/pilum/lib/configutil"
)

const (
	// DefaultTokenEnv is the default environment variable name for the Cloudflare API token.
	DefaultTokenEnv = "CLOUDFLARE_API_TOKEN" //nolint:gosec // env var name, not a credential
	// DefaultProductionBranch is the default branch for production deployments.
	DefaultProductionBranch = "main"
	// DefaultPackageManager is the default package manager.
	DefaultPackageManager = "npm"
	// DefaultOutputDir is the default build output directory.
	DefaultOutputDir = "dist"
)

// Config holds Cloudflare Pages-specific configuration.
type Config struct {
	AccountID        string // Cloudflare account ID
	TokenEnv         string // Env var name for API token (default: CLOUDFLARE_API_TOKEN)
	ProductionBranch string // Branch for production deploys (default: main)
	PackageManager   string // npm or pnpm (default: npm)
	OutputDir        string // Build output directory (default: dist)
}

// ParseConfig extracts Cloudflare configuration from the raw service config.
func ParseConfig(config map[string]any) Config {
	cfMap := configutil.MapFromAny(config["cloudflare"])
	buildMap := configutil.MapFromAny(config["build"])

	return Config{
		AccountID:        configutil.GetString(cfMap, "account_id", ""),
		TokenEnv:         configutil.GetString(cfMap, "token_env", DefaultTokenEnv),
		ProductionBranch: configutil.GetString(cfMap, "production_branch", DefaultProductionBranch),
		PackageManager:   configutil.GetString(cfMap, "package_manager", DefaultPackageManager),
		OutputDir:        configutil.GetString(buildMap, "output_dir", DefaultOutputDir),
	}
}
