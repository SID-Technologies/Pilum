package serviceinfo

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/providers"
	"github.com/sid-technologies/pilum/lib/suggest"
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

type HomebrewConfig struct {
	TapURL     string `yaml:"tap_url"`     // Full repository URL for tap (e.g., https://github.com/org/Homebrew-tap)
	ProjectURL string `yaml:"project_url"` // Full repository URL for releases (e.g., https://github.com/org/project)
	TokenEnv   string `yaml:"token_env"`   // Environment variable name for auth token
}

type CloudRunConfig struct {
	MinInstances  *int   `yaml:"min_instances"`  // nil = don't set, 0 = scale to zero
	MaxInstances  *int   `yaml:"max_instances"`  // nil = don't set
	CPUThrottling *bool  `yaml:"cpu_throttling"` // nil = don't set (uses GCP default)
	Memory        string `yaml:"memory"`         // e.g., "2048Mi", "512Mi"
	CPU           string `yaml:"cpu"`            // e.g., "1", "2"
	Concurrency   int    `yaml:"concurrency"`    // max concurrent requests per instance
	Timeout       int    `yaml:"timeout"`        // request timeout in seconds
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
	CloudRunConfig CloudRunConfig `yaml:"cloud_run"`
	EnvVars        []EnvVars      `yaml:"env_vars"`
	Secrets        []Secrets      `yaml:"secrets"`
	Region         string         `yaml:"region"`
	Regions        []string       `yaml:"regions"` // For multi-region deployments
	IsMultiRegion  bool           `yaml:"-"`       // True if this was expanded from a multi-region config
	Project        string         `yaml:"project"`
	License        string         `yaml:"license"`
	Provider       string         `yaml:"provider"`
	Service        string         `yaml:"service"` // Deployment target (e.g., cloud-run, gke, lambda)
	RegistryName   string         `yaml:"registry_name"`
	DependsOn      []string       `yaml:"depends_on"` // Services this service depends on
}

// DisplayName returns the service name with region suffix for multi-region deployments.
func (s *ServiceInfo) DisplayName() string {
	if s.IsMultiRegion && s.Region != "" {
		return s.Name + " (" + s.Region + ")"
	}
	return s.Name
}

// RecipeKey returns the key used to match this service to a recipe.
// Format: "{provider}-{service}" when service is set, otherwise just "{provider}".
func (s *ServiceInfo) RecipeKey() string {
	if s.Service != "" {
		return s.Provider + "-" + s.Service
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

	// Validate provider is supported
	if !providers.IsValidProvider(s.Provider) {
		suggestion := suggest.FormatSuggestion(s.Provider, providers.GetProviders())
		if suggestion != "" {
			return errors.New("unknown provider '%s' - %s", s.Provider, suggestion)
		}
		return errors.New("unknown provider '%s'", s.Provider)
	}

	// Validate service is supported for this provider (only if service is specified)
	if s.Service != "" && !providers.IsValidService(s.Provider, s.Service) {
		validServices := providers.GetServices(s.Provider)
		if len(validServices) > 0 {
			suggestion := suggest.FormatSuggestion(s.Service, validServices)
			if suggestion != "" {
				return errors.New("unknown service '%s' for provider '%s' - %s", s.Service, s.Provider, suggestion)
			}
			return errors.New("unknown service '%s' for provider '%s' (valid: %v)", s.Service, s.Provider, validServices)
		}
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

	// Parse cloud run config if present
	cloudRunConfig := parseCloudRunConfig(config)

	return &ServiceInfo{
		Name:           getString(config, "name", ""),
		Description:    getString(config, "description", ""),
		Template:       template,
		Path:           path,
		Config:         config,
		BuildConfig:    buildConfig,
		Runtime:        runtime,
		HomebrewConfig: homebrewConfig,
		CloudRunConfig: cloudRunConfig,
		Region:         getString(config, "region", ""),
		Regions:        getStringSlice(config, "regions"),
		Project:        getString(config, "project", ""),
		License:        getString(config, "license", ""),
		Provider:       provider,
		Service:        getString(config, "service", ""),
		RegistryName:   getString(config, "registry_name", ""),
		DependsOn:      getStringSlice(config, "depends_on"),
		EnvVars:        envVars,
		Secrets:        secretVars,
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

func parseCloudRunConfig(config map[string]any) CloudRunConfig {
	crMap := mapFromAny(config["cloud_run"])
	if len(crMap) == 0 {
		return CloudRunConfig{}
	}

	cfg := CloudRunConfig{
		Memory:      getString(crMap, "memory", ""),
		CPU:         getString(crMap, "cpu", ""),
		Concurrency: getInt(crMap, "concurrency", 0),
		Timeout:     getInt(crMap, "timeout", 0),
	}

	// Handle pointer fields (nil = not set, value = explicitly set)
	if v, ok := crMap["min_instances"]; ok {
		if intVal, ok := v.(int); ok {
			cfg.MinInstances = &intVal
		}
	}
	if v, ok := crMap["max_instances"]; ok {
		if intVal, ok := v.(int); ok {
			cfg.MaxInstances = &intVal
		}
	}
	if v, ok := crMap["cpu_throttling"]; ok {
		if boolVal, ok := v.(bool); ok {
			cfg.CPUThrottling = &boolVal
		}
	}

	return cfg
}

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
