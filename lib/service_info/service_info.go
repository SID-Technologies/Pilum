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

type ServiceInfo struct {
	Name         string         `yaml:"name"`
	Template     string         `yaml:"template"`
	Path         string         `yaml:"-"`
	Config       map[string]any `yaml:"-"`
	BuildConfig  BuildConfig    `yaml:"build"`
	Runtime      RuntimeConfig  `yaml:"runtime"`
	EnvVars      []EnvVars      `yaml:"env_vars"`
	Secrets      []Secrets      `yaml:"secrets"`
	Region       string         `yaml:"region"`
	Project      string         `yaml:"project"`
	Provider     string         `yaml:"provider"`
	RegistryName string         `yaml:"registry_name"`
}

func (s *ServiceInfo) Validate() error {
	if s.Name == "" {
		return errors.New("missing required field: name")
	}
	if s.Template == "" {
		return errors.New("missing required field: template")
	}
	if s.Region == "" {
		return errors.New("missing required field: region")
	}
	if s.Project == "" {
		return errors.New("missing required field: project")
	}
	if s.Provider == "" {
		return errors.New("missing required field: provider")
	}

	// add more rules as needed
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

	return &ServiceInfo{
		Name:        getString(config, "name", ""),
		Template:    template,
		Path:        path,
		Config:      config,
		BuildConfig: buildConfig,
		Runtime:     runtime,
		Region:      getString(config, "region", ""),
		Project:     getString(config, "project", ""),
		Provider:    provider,
		EnvVars:     envVars,
		Secrets:     secretVars,
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
