package serviceinfo

// BuildFlag represents a build flag with multiple values.
type BuildFlag struct {
	Name   string   `yaml:"name"`   // e.g., "ldflags", "gcflags"
	Values []string `yaml:"values"` // e.g., ["-s", "-w"]
}

// BuildConfig contains build-related configuration.
type BuildConfig struct {
	Language   string      `yaml:"language"`
	Version    string      `yaml:"version"`
	Cmd        string      `yaml:"cmd"`
	EnvVars    []EnvVars   `yaml:"env_vars"`
	Flags      []BuildFlag `yaml:"flags"`
	VersionVar string      `yaml:"version_var"` // Go variable path for version injection (e.g., "main.version")
}

// parseBuildConfig parses build configuration from config map.
func parseBuildConfig(config map[string]any) BuildConfig {
	buildMap := mapFromAny(config["build"])
	if len(buildMap) == 0 {
		return BuildConfig{}
	}

	bc := BuildConfig{
		Language:   getString(buildMap, "language", ""),
		Version:    getString(buildMap, "version", ""),
		Cmd:        getString(buildMap, "cmd", ""),
		VersionVar: getString(buildMap, "version_var", ""),
	}

	// Parse build env vars
	buildEnvVars := mapFromAny(buildMap["env_vars"])
	for k, v := range buildEnvVars {
		if val, ok := v.(string); ok {
			bc.EnvVars = append(bc.EnvVars, EnvVars{Name: k, Value: val})
		}
	}

	// Parse build flags (e.g., ldflags: ["-s", "-w"])
	flagsMap := mapFromAny(buildMap["flags"])
	for flagName, flagVal := range flagsMap {
		var values []string
		switch v := flagVal.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					values = append(values, s)
				}
			}
		case []string:
			values = v
		case string:
			values = []string{v}
		}
		if len(values) > 0 {
			bc.Flags = append(bc.Flags, BuildFlag{Name: flagName, Values: values})
		}
	}

	return bc
}
