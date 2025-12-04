package serviceinfo

import (
	"github.com/sid-technologies/pilum/lib/errors"
)

type EnvVars struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Secrets struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type BuildFlag struct {
	Name   string   `yaml:"name"`   // e.g., "ldflags", "gcflags"
	Values []string `yaml:"values"` // e.g., ["-s", "-w"]
}

type BuildConfig struct {
	Language string      `yaml:"language"`
	Version  string      `yaml:"version"`
	Cmd      string      `yaml:"cmd"`
	EnvVars  []EnvVars   `yaml:"env_vars"`
	Flags    []BuildFlag `yaml:"flags"`
}

type RuntimeConfig struct {
	Service string `yaml:"service"`
}

type HomebrewConfig struct {
	TapURL     string `yaml:"tap_url"`     // Full repository URL for tap (e.g., https://github.com/org/Homebrew-tap)
	ProjectURL string `yaml:"project_url"` // Full repository URL for releases (e.g., https://github.com/org/project)
	TokenEnv   string `yaml:"token_env"`   // Environment variable name for auth token
}

type ServiceInfo struct {
	Name           string         `yaml:"name"`
	Description    string         `yaml:"description"`
	Template       string         `yaml:"template"`
	Path           string         `yaml:"-"`
	Config         map[string]any `yaml:"-"`
	BuildConfig    BuildConfig    `yaml:"build"`
	Runtime        RuntimeConfig  `yaml:"runtime"`
	HomebrewConfig HomebrewConfig `yaml:"homebrew"`
	EnvVars        []EnvVars      `yaml:"env_vars"`
	Secrets        []Secrets      `yaml:"secrets"`
	Region         string         `yaml:"region"`
	Project        string         `yaml:"project"`
	License        string         `yaml:"license"`
	Provider       string         `yaml:"provider"`
	RegistryName   string         `yaml:"registry_name"`
}

func (s *ServiceInfo) Validate() error {
	// Minimal base validation - provider-specific validation is done by recipes
	if s.Name == "" {
		return errors.New("missing required field: name")
	}
	if s.Provider == "" {
		return errors.New("missing required field: provider")
	}
	return nil
}

func NewServiceInfo(config map[string]any, path string) *ServiceInfo {
	rt := mapFromAny(config["runtime"])
	runtime := RuntimeConfig{}

	if rt["service"] != nil {
		if svc, ok := rt["service"].(string); ok {
			runtime.Service = svc
		}
	}

	// env vars conversions
	evs := mapFromAny(config["env_vars"])
	var envVars []EnvVars
	for k, v := range evs {
		key := k
		val, ok := v.(string)
		if !ok {
			return nil
		}
		envVars = append(envVars, EnvVars{Name: key, Value: val})
	}

	// secrets conversion
	secrets := mapFromAny(config["secrets"])
	var secretVars []Secrets
	for k, v := range secrets {
		secretVars = append(secretVars, Secrets{Name: k, Value: v.(string)})
	}

	// Parse build config
	buildConfig := parseBuildConfig(config)

	// Template can be specified as "template" or "type"
	template := getString(config, "template", "")
	if template == "" {
		template = getString(config, "type", "")
	}

	// Provider can be explicit or derived from type
	provider := getString(config, "provider", "")
	if provider == "" {
		// Derive provider from type if not explicitly set
		switch template {
		case "gcp-cloud-run", "gcp":
			provider = "gcp"
		case "aws-lambda", "aws-ecs", "aws":
			provider = "aws"
		case "azure-container-apps", "azure":
			provider = "azure"
		case "homebrew":
			provider = "homebrew"
		default:
			// Unknown template type, leave provider empty
		}
	}

	// Parse homebrew config if present
	homebrewConfig := parseHomebrewConfig(config)

	return &ServiceInfo{
		Name:           getString(config, "name", ""),
		Description:    getString(config, "description", ""),
		Template:       template,
		Path:           path,
		Config:         config,
		BuildConfig:    buildConfig,
		Runtime:        runtime,
		HomebrewConfig: homebrewConfig,
		Region:         getString(config, "region", ""),
		Project:        getString(config, "project", ""),
		License:        getString(config, "license", ""),
		Provider:       provider,
		EnvVars:        envVars,
		Secrets:        secretVars,
	}
}

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

func parseBuildConfig(config map[string]any) BuildConfig {
	buildMap := mapFromAny(config["build"])
	if len(buildMap) == 0 {
		return BuildConfig{}
	}

	bc := BuildConfig{
		Language: getString(buildMap, "language", ""),
		Version:  getString(buildMap, "version", ""),
		Cmd:      getString(buildMap, "cmd", ""),
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
