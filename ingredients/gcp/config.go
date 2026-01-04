package gcp

import (
	"github.com/sid-technologies/pilum/lib/configutil"
)

// CloudRunConfig holds GCP Cloud Run specific configuration.
type CloudRunConfig struct {
	MinInstances   int    // Minimum number of instances (0 for scale to zero, -1 for not set)
	MaxInstances   int    // Maximum number of instances (-1 for not set)
	CPUThrottling  bool   // Enable CPU throttling
	Memory         string // Memory limit (e.g., "512Mi", "1Gi")
	CPU            string // CPU limit (e.g., "1", "2")
	Concurrency    int    // Max concurrent requests per instance (-1 for not set)
	TimeoutSeconds int    // Request timeout in seconds (-1 for not set)
}

// ParseCloudRunConfig extracts Cloud Run configuration from the raw service config.
// Returns a CloudRunConfig with sensible defaults for unset values.
func ParseCloudRunConfig(config map[string]any) CloudRunConfig {
	cloudRunMap := configutil.MapFromAny(config["cloud_run"])
	if len(cloudRunMap) == 0 {
		// Return config with -1 for int fields to indicate "not set"
		return CloudRunConfig{
			MinInstances:   -1,
			MaxInstances:   -1,
			Concurrency:    -1,
			TimeoutSeconds: -1,
		}
	}

	return CloudRunConfig{
		MinInstances:   configutil.GetInt(cloudRunMap, "min_instances", -1),
		MaxInstances:   configutil.GetInt(cloudRunMap, "max_instances", -1),
		CPUThrottling:  configutil.GetBool(cloudRunMap, "cpu_throttling", false),
		Memory:         configutil.GetString(cloudRunMap, "memory", ""),
		CPU:            configutil.GetString(cloudRunMap, "cpu", ""),
		Concurrency:    configutil.GetInt(cloudRunMap, "concurrency", -1),
		TimeoutSeconds: configutil.GetInt(cloudRunMap, "timeout_seconds", -1),
	}
}
