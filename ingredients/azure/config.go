package azure

import (
	"github.com/sid-technologies/pilum/lib/configutil"
)

// ContainerAppConfig holds Azure Container Apps specific configuration.
type ContainerAppConfig struct {
	Environment     string // Container Apps environment name
	MinReplicas     int    // Minimum replicas (0 for scale to zero, -1 for not set)
	MaxReplicas     int    // Maximum replicas (-1 for not set)
	CPU             string // CPU cores (e.g., "0.25", "0.5", "1")
	Memory          string // Memory (e.g., "0.5Gi", "1Gi")
	IngressPort     int    // Container port for ingress (-1 for not set)
	IngressExternal bool   // Whether ingress is externally accessible
}

// ParseContainerAppConfig extracts Container App configuration from the raw service config.
func ParseContainerAppConfig(config map[string]any) ContainerAppConfig {
	caMap := configutil.MapFromAny(config["container_app"])
	if len(caMap) == 0 {
		return ContainerAppConfig{
			MinReplicas: -1,
			MaxReplicas: -1,
			IngressPort: -1,
		}
	}

	return ContainerAppConfig{
		Environment:     configutil.GetString(caMap, "environment", ""),
		MinReplicas:     configutil.GetInt(caMap, "min_replicas", -1),
		MaxReplicas:     configutil.GetInt(caMap, "max_replicas", -1),
		CPU:             configutil.GetString(caMap, "cpu", ""),
		Memory:          configutil.GetString(caMap, "memory", ""),
		IngressPort:     configutil.GetInt(caMap, "ingress_port", -1),
		IngressExternal: configutil.GetBool(caMap, "ingress_external", true),
	}
}
