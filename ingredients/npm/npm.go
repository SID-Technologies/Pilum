package npm

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/shellutil"
)

// GenerateInstallCommand creates the dependency install command based on package_manager config.
func GenerateInstallCommand(svc serviceinfo.ServiceInfo) []string {
	cfg := ParseConfig(svc.Config)

	switch cfg.PackageManager {
	case "pnpm":
		return []string{"pnpm", "install", "--frozen-lockfile"}
	case "yarn":
		return []string{"yarn", "install", "--frozen-lockfile"}
	case "bun":
		return []string{"bun", "install", "--frozen-lockfile"}
	default:
		return []string{"npm", "ci"}
	}
}

// GenerateSetVersionCommand creates the npm version command to sync package.json with the release tag.
// Strips leading "v" from the tag since npm versions don't use the v prefix.
func GenerateSetVersionCommand(tag string) []string {
	version := strings.TrimPrefix(tag, "v")
	return []string{"npm", "version", version, "--no-git-tag-version", "--allow-same-version"}
}

// GenerateBuildCommand creates the build command for the npm package.
// Returns nil if no build command is configured (runner treats nil as success-skip).
func GenerateBuildCommand(svc serviceinfo.ServiceInfo) []string {
	if svc.BuildConfig.Cmd == "" {
		return nil
	}
	return []string{"/bin/sh", "-c", svc.BuildConfig.Cmd}
}

// GeneratePublishCommand creates a shell script that configures npm auth and publishes the package.
func GeneratePublishCommand(svc serviceinfo.ServiceInfo) string {
	cfg := ParseConfig(svc.Config)

	tokenEnv := cfg.TokenEnv
	registry := cfg.Registry
	access := cfg.Access
	scope := cfg.Scope

	// Extract the registry host from the URL for .npmrc auth line
	// e.g., "https://npm.pkg.github.com" -> "npm.pkg.github.com"
	registryHost := strings.TrimPrefix(registry, "https://")
	registryHost = strings.TrimPrefix(registryHost, "http://")
	registryHost = strings.TrimSuffix(registryHost, "/")

	return fmt.Sprintf(`if [ -z "$%s" ]; then
  echo "Error: %s environment variable is not set"
  exit 1
fi
echo "//%s/:_authToken=$%s" > .npmrc
echo "%s:registry=%s" >> .npmrc
npm publish --access %s`,
		tokenEnv, tokenEnv,
		shellutil.SanitizeHeredocValue(registryHost), tokenEnv,
		shellutil.SanitizeHeredocValue(scope), shellutil.SanitizeHeredocValue(registry),
		shellutil.Quote(access),
	)
}
