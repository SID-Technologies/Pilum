package serviceinfo

import (
	"github.com/sid-technologies/pilum/lib/configutil"
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
	Language   string      `yaml:"language"`
	Version    string      `yaml:"version"`
	Cmd        string      `yaml:"cmd"`
	EnvVars    []EnvVars   `yaml:"env_vars"`
	Flags      []BuildFlag `yaml:"flags"`
	VersionVar string      `yaml:"version_var"` // Go variable path for version injection (e.g., "main.version")
}

type RuntimeConfig struct {
	Service string `yaml:"service"`
}

type ServiceInfo struct {
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description"`
	Template      string         `yaml:"template"`
	Path          string         `yaml:"-"`
	Config        map[string]any `yaml:"-"`
	BuildConfig   BuildConfig    `yaml:"build"`
	Runtime       RuntimeConfig  `yaml:"runtime"`
	EnvVars       []EnvVars      `yaml:"env_vars"`
	Secrets       []Secrets      `yaml:"secrets"`
	Region        string         `yaml:"region"`
	Regions       []string       `yaml:"regions"` // For multi-region deployments
	IsMultiRegion bool           `yaml:"-"`       // True if this was expanded from a multi-region config
	Project       string         `yaml:"project"`
	License       string         `yaml:"license"`
	Provider      string         `yaml:"provider"`
	RegistryName  string         `yaml:"registry_name"`
	DependsOn     []string       `yaml:"depends_on"` // Services this service depends on
}

// DisplayName returns the service name with region suffix for multi-region deployments.
func (s *ServiceInfo) DisplayName() string {
	if s.IsMultiRegion && s.Region != "" {
		return s.Name + " (" + s.Region + ")"
	}
	return s.Name
}

// RecipeKey returns the recipe lookup key for this service.
// This matches the format used to index recipes: "provider-service" or just "provider".
// It uses the Template field (from "type" in config) which contains the full key.
func (s *ServiceInfo) RecipeKey() string {
	// Template contains the type (e.g., "gcp-cloud-run", "homebrew")
	if s.Template != "" {
		return s.Template
	}
	return s.Provider
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
	rt := configutil.MapFromAny(config["runtime"])
	runtime := RuntimeConfig{}

	if rt["service"] != nil {
		if svc, ok := rt["service"].(string); ok {
			runtime.Service = svc
		}
	}

	// env vars conversions
	evs := configutil.MapFromAny(config["env_vars"])
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
	secrets := configutil.MapFromAny(config["secrets"])
	var secretVars []Secrets
	for k, v := range secrets {
		secretVars = append(secretVars, Secrets{Name: k, Value: v.(string)})
	}

	// Parse build config
	buildConfig := parseBuildConfig(config)

	// Template can be specified as "template" or "type"
	template := configutil.GetString(config, "template", "")
	if template == "" {
		template = configutil.GetString(config, "type", "")
	}

	// Provider can be explicit or derived from type
	provider := configutil.GetString(config, "provider", "")
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
		Name:         configutil.GetString(config, "name", ""),
		Description:  configutil.GetString(config, "description", ""),
		Template:     template,
		Path:         path,
		Config:       config,
		BuildConfig:  buildConfig,
		Runtime:      runtime,
		Region:       configutil.GetString(config, "region", ""),
		Regions:      configutil.GetStringSlice(config, "regions"),
		Project:      configutil.GetString(config, "project", ""),
		License:      configutil.GetString(config, "license", ""),
		Provider:     provider,
		RegistryName: configutil.GetString(config, "registry_name", ""),
		DependsOn:    configutil.GetStringSlice(config, "depends_on"),
		EnvVars:      envVars,
		Secrets:      secretVars,
	}
}

// ExpandMultiRegion expands a service with multiple regions into separate ServiceInfo instances.
// If the service has a Regions array, it creates one instance per region.
// If the service only has a single Region, it returns the original service unchanged.
func ExpandMultiRegion(svc ServiceInfo) []ServiceInfo {
	// If no regions array, return as-is
	if len(svc.Regions) == 0 {
		return []ServiceInfo{svc}
	}

	// Expand into multiple services, one per region
	expanded := make([]ServiceInfo, 0, len(svc.Regions))
	for _, region := range svc.Regions {
		instance := svc          // copy
		instance.Region = region // set specific region
		instance.Regions = nil   // clear regions array
		instance.IsMultiRegion = true
		expanded = append(expanded, instance)
	}

	return expanded
}

func parseBuildConfig(config map[string]any) BuildConfig {
	buildMap := configutil.MapFromAny(config["build"])
	if len(buildMap) == 0 {
		return BuildConfig{}
	}

	bc := BuildConfig{
		Language:   configutil.GetString(buildMap, "language", ""),
		Version:    configutil.GetString(buildMap, "version", ""),
		Cmd:        configutil.GetString(buildMap, "cmd", ""),
		VersionVar: configutil.GetString(buildMap, "version_var", ""),
	}

	// Parse build env vars
	buildEnvVars := configutil.MapFromAny(buildMap["env_vars"])
	for k, v := range buildEnvVars {
		if val, ok := v.(string); ok {
			bc.EnvVars = append(bc.EnvVars, EnvVars{Name: k, Value: val})
		}
	}

	// Parse build flags (e.g., ldflags: ["-s", "-w"])
	flagsMap := configutil.MapFromAny(buildMap["flags"])
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
