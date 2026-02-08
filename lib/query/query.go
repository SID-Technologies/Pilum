// Package query provides types and helpers for querying cloud service status and logs.
package query

import (
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// ServiceStatus holds normalized status information for a deployed service.
type ServiceStatus struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Region   string `json:"region,omitempty"`
	Status   string `json:"status"`
	Image    string `json:"image,omitempty"`
	URL      string `json:"url,omitempty"`
	Replicas string `json:"replicas,omitempty"`
	Updated  string `json:"updated,omitempty"`
}

// cloudProviders is the set of providers that have queryable cloud services.
var cloudProviders = map[string]bool{
	"gcp":        true,
	"aws":        true,
	"azure":      true,
	"cloudflare": true,
}

// IsCloudProvider returns true if the provider has queryable cloud services.
func IsCloudProvider(provider string) bool {
	return cloudProviders[provider]
}

// FilterCloudServices returns only services whose provider is a cloud provider.
func FilterCloudServices(services []serviceinfo.ServiceInfo) []serviceinfo.ServiceInfo {
	var cloud []serviceinfo.ServiceInfo
	for _, svc := range services {
		if IsCloudProvider(svc.Provider) {
			cloud = append(cloud, svc)
		}
	}
	return cloud
}
