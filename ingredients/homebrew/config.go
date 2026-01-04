package homebrew

import (
	"github.com/sid-technologies/pilum/lib/configutil"
)

// Config holds Homebrew-specific configuration.
type Config struct {
	TapURL     string // Full repository URL for tap (e.g., https://github.com/org/Homebrew-tap)
	ProjectURL string // Full repository URL for releases (e.g., https://github.com/org/project)
	TokenEnv   string // Environment variable name for auth token
}

// ParseConfig extracts Homebrew configuration from the raw service config.
func ParseConfig(config map[string]any) Config {
	brewMap := configutil.MapFromAny(config["homebrew"])
	if len(brewMap) == 0 {
		return Config{}
	}

	return Config{
		TapURL:     configutil.GetString(brewMap, "tap_url", ""),
		ProjectURL: configutil.GetString(brewMap, "project_url", ""),
		TokenEnv:   configutil.GetString(brewMap, "token_env", ""),
	}
}
