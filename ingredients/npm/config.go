package npm

import (
	"github.com/sid-technologies/pilum/lib/configutil"
)

const (
	// DefaultRegistry is the default npm registry URL (GitHub Packages).
	DefaultRegistry = "https://npm.pkg.github.com"
	// DefaultAccess is the default package access level.
	DefaultAccess = "restricted"
)

// Config holds npm-specific configuration.
type Config struct {
	Scope    string // npm scope (e.g., "@sid-technologies")
	Registry string // npm registry URL (e.g., "https://npm.pkg.github.com")
	TokenEnv string // Environment variable name for auth token
	Access   string // Package access level: "restricted" or "public"
}

// ParseConfig extracts npm configuration from the raw service config.
func ParseConfig(config map[string]any) Config {
	npmMap := configutil.MapFromAny(config["npm"])

	registry := configutil.GetString(npmMap, "registry", DefaultRegistry)
	access := configutil.GetString(npmMap, "access", DefaultAccess)

	return Config{
		Scope:    configutil.GetString(npmMap, "scope", ""),
		Registry: registry,
		TokenEnv: configutil.GetString(npmMap, "token_env", ""),
		Access:   access,
	}
}
