package homebrew

import (
	"github.com/sid-technologies/pilum/lib/configutil"
)

// HomebrewConfig holds Homebrew-specific configuration.
type HomebrewConfig struct {
	TapURL     string // Full repository URL for tap (e.g., https://github.com/org/Homebrew-tap)
	ProjectURL string // Full repository URL for releases (e.g., https://github.com/org/project)
	TokenEnv   string // Environment variable name for auth token
}

// ParseHomebrewConfig extracts Homebrew configuration from the raw service config.
func ParseHomebrewConfig(config map[string]any) HomebrewConfig {
	brewMap := configutil.MapFromAny(config["homebrew"])
	if len(brewMap) == 0 {
		return HomebrewConfig{}
	}

	return HomebrewConfig{
		TapURL:     configutil.GetString(brewMap, "tap_url", ""),
		ProjectURL: configutil.GetString(brewMap, "project_url", ""),
		TokenEnv:   configutil.GetString(brewMap, "token_env", ""),
	}
}
