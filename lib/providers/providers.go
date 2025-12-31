package providers

import (
	"gopkg.in/yaml.v3"

	_ "embed"
)

//go:embed providers.yaml
var providersYAML []byte

// ProviderInfo contains metadata about a provider.
type ProviderInfo struct {
	Name     string   `yaml:"name"`
	Services []string `yaml:"services"`
}

// Config is the root structure of the providers.yaml file.
type Config struct {
	Providers map[string]ProviderInfo `yaml:"providers"`
}

var config *Config

func init() {
	config = &Config{}
	if err := yaml.Unmarshal(providersYAML, config); err != nil {
		panic("failed to parse embedded providers.yaml: " + err.Error())
	}
}

// IsValidProvider returns true if the provider is supported.
func IsValidProvider(provider string) bool {
	_, exists := config.Providers[provider]
	return exists
}

// IsValidService returns true if the service is supported for the given provider.
func IsValidService(provider, service string) bool {
	p, exists := config.Providers[provider]
	if !exists {
		return false
	}
	// Empty services list means any service is valid (or no service required)
	if len(p.Services) == 0 {
		return service == ""
	}
	for _, s := range p.Services {
		if s == service {
			return true
		}
	}
	return false
}

// GetProviders returns a list of all supported provider names.
func GetProviders() []string {
	providers := make([]string, 0, len(config.Providers))
	for name := range config.Providers {
		providers = append(providers, name)
	}
	return providers
}

// GetServices returns a list of supported services for a provider.
func GetServices(provider string) []string {
	p, exists := config.Providers[provider]
	if !exists {
		return nil
	}
	return p.Services
}

// GetAllRecipeKeys returns all valid provider-service combinations.
func GetAllRecipeKeys() []string {
	var keys []string
	for provider, info := range config.Providers {
		if len(info.Services) == 0 {
			keys = append(keys, provider)
		} else {
			for _, service := range info.Services {
				keys = append(keys, provider+"-"+service)
			}
		}
	}
	return keys
}

// GetProviderName returns the display name for a provider.
func GetProviderName(provider string) string {
	p, exists := config.Providers[provider]
	if !exists {
		return provider
	}
	return p.Name
}
