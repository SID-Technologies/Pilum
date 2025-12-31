package serviceinfo

// HomebrewConfig contains Homebrew-specific configuration.
type HomebrewConfig struct {
	TapURL     string `yaml:"tap_url"`     // Full repository URL for tap (e.g., https://github.com/org/Homebrew-tap)
	ProjectURL string `yaml:"project_url"` // Full repository URL for releases (e.g., https://github.com/org/project)
	TokenEnv   string `yaml:"token_env"`   // Environment variable name for auth token
}

// parseHomebrewConfig parses Homebrew configuration from config map.
func parseHomebrewConfig(config map[string]any) HomebrewConfig {
	brewMap := mapFromAny(config["homebrew"])
	if len(brewMap) == 0 {
		return HomebrewConfig{}
	}

	return HomebrewConfig{
		TapURL:     getString(brewMap, "tap_url", ""),
		ProjectURL: getString(brewMap, "project_url", ""),
		TokenEnv:   getString(brewMap, "token_env", ""),
	}
}
