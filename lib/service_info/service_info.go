package serviceinfo

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/providers"
	"github.com/sid-technologies/pilum/lib/suggest"
)

// ServiceInfo contains all configuration for a service deployment.
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

// Validate performs basic validation on the service configuration.
func (s *ServiceInfo) Validate() error {
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

// NewServiceInfo creates a ServiceInfo from a config map and path.
func NewServiceInfo(config map[string]any, path string) *ServiceInfo {
	// Parse provider (can be explicit or derived from template)
	template := getString(config, "template", "")
	if template == "" {
		template = getString(config, "type", "")
	}

	provider := getString(config, "provider", "")
	if provider == "" {
		provider = deriveProviderFromTemplate(template)
	}

	// Parse env vars - return nil if invalid
	envVars, ok := parseEnvVars(config)
	if !ok {
		return nil
	}

	return &ServiceInfo{
		Name:           getString(config, "name", ""),
		Description:    getString(config, "description", ""),
		Template:       template,
		Path:           path,
		Config:         config,
		BuildConfig:    parseBuildConfig(config),
		Runtime:        parseRuntimeConfig(config),
		HomebrewConfig: parseHomebrewConfig(config),
		CloudRunConfig: parseCloudRunConfig(config),
		Region:         getString(config, "region", ""),
		Regions:        getStringSlice(config, "regions"),
		Project:        getString(config, "project", ""),
		License:        getString(config, "license", ""),
		Provider:       provider,
		Service:        getString(config, "service", ""),
		RegistryName:   getString(config, "registry_name", ""),
		DependsOn:      getStringSlice(config, "depends_on"),
		EnvVars:        envVars,
		Secrets:        parseSecrets(config),
	}
}

// deriveProviderFromTemplate infers the provider from template name.
func deriveProviderFromTemplate(template string) string {
	switch template {
	case "gcp-cloud-run", "gcp":
		return "gcp"
	case "aws-lambda", "aws-ecs", "aws":
		return "aws"
	case "azure-container-apps", "azure":
		return "azure"
	case "homebrew":
		return "homebrew"
	default:
		return ""
	}
}

// ExpandMultiRegion expands a service with multiple regions into separate ServiceInfo instances.
// If the service has a Regions array, it creates one instance per region.
// If the service only has a single Region, it returns the original service unchanged.
func ExpandMultiRegion(svc ServiceInfo) []ServiceInfo {
	if len(svc.Regions) == 0 {
		return []ServiceInfo{svc}
	}

	expanded := make([]ServiceInfo, 0, len(svc.Regions))
	for _, region := range svc.Regions {
		instance := svc
		instance.Region = region
		instance.Regions = nil
		instance.IsMultiRegion = true
		expanded = append(expanded, instance)
	}

	return expanded
}
