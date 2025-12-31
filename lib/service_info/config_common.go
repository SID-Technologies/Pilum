package serviceinfo

// EnvVars represents an environment variable key-value pair.
type EnvVars struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Secrets represents a secret key-value pair.
type Secrets struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// RuntimeConfig contains runtime-specific configuration.
type RuntimeConfig struct {
	Service string `yaml:"service"`
}

// parseEnvVars parses environment variables from config map.
// Returns nil, false if a non-string value is found.
func parseEnvVars(config map[string]any) ([]EnvVars, bool) {
	evs := mapFromAny(config["env_vars"])
	var envVars []EnvVars
	for k, v := range evs {
		val, ok := v.(string)
		if !ok {
			return nil, false
		}
		envVars = append(envVars, EnvVars{Name: k, Value: val})
	}
	return envVars, true
}

// parseSecrets parses secrets from config map.
func parseSecrets(config map[string]any) []Secrets {
	secrets := mapFromAny(config["secrets"])
	var secretVars []Secrets
	for k, v := range secrets {
		if val, ok := v.(string); ok {
			secretVars = append(secretVars, Secrets{Name: k, Value: val})
		}
	}
	return secretVars
}

// parseRuntimeConfig parses runtime configuration from config map.
func parseRuntimeConfig(config map[string]any) RuntimeConfig {
	rt := mapFromAny(config["runtime"])
	runtime := RuntimeConfig{}

	if rt["service"] != nil {
		if svc, ok := rt["service"].(string); ok {
			runtime.Service = svc
		}
	}
	return runtime
}
