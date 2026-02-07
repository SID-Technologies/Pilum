package npm

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateInstallCommand creates the npm ci command for clean dependency installation.
func GenerateInstallCommand() []string {
	return []string{"npm", "ci"}
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
		registryHost, tokenEnv,
		scope, registry,
		access,
	)
}
