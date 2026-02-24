package cloudflare

import (
	"fmt"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/shellutil"
)

// GenerateInstallCommand creates the dependency install command.
// Uses npm ci or pnpm install --frozen-lockfile based on package_manager config.
func GenerateInstallCommand(svc serviceinfo.ServiceInfo) []string {
	cfg := ParseConfig(svc.Config)

	if cfg.PackageManager == "pnpm" {
		return []string{"pnpm", "install", "--frozen-lockfile"}
	}
	return []string{"npm", "ci"}
}

// GenerateBuildCommand creates the build command for the static site.
// Returns nil if no build command is configured (runner treats nil as success-skip).
func GenerateBuildCommand(svc serviceinfo.ServiceInfo) []string {
	if svc.BuildConfig.Cmd == "" {
		return nil
	}
	return []string{"/bin/sh", "-c", svc.BuildConfig.Cmd}
}

// GenerateDeployCommand creates a shell script that deploys to Cloudflare Pages via wrangler.
// The tag determines whether this is a production or preview deployment:
// - If tag matches production_branch or is "latest", it deploys to production
// - Otherwise it deploys as a preview with the tag as the branch name
func GenerateDeployCommand(svc serviceinfo.ServiceInfo, tag string) string {
	cfg := ParseConfig(svc.Config)

	tokenEnv := cfg.TokenEnv
	accountID := cfg.AccountID
	outputDir := cfg.OutputDir
	productionBranch := cfg.ProductionBranch

	// Determine branch flag: production or preview
	branchFlag := fmt.Sprintf("--branch=%s", shellutil.Quote(tag))
	if tag == productionBranch || tag == "latest" {
		branchFlag = fmt.Sprintf("--branch=%s", shellutil.Quote(productionBranch))
	}

	return fmt.Sprintf(`if [ -z "$%s" ]; then
  echo "Error: %s environment variable is not set"
  exit 1
fi
export CLOUDFLARE_API_TOKEN="$%s"
export CLOUDFLARE_ACCOUNT_ID=%s
npx wrangler pages deploy %s --project-name=%s %s`,
		tokenEnv, tokenEnv,
		tokenEnv,
		shellutil.Quote(accountID),
		shellutil.Quote(outputDir), shellutil.Quote(svc.Name), branchFlag,
	)
}
