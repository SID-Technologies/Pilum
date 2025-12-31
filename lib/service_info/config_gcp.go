package serviceinfo

// CloudRunConfig contains GCP Cloud Run specific configuration.
type CloudRunConfig struct {
	MinInstances  *int   `yaml:"min_instances"`  // nil = don't set, 0 = scale to zero
	MaxInstances  *int   `yaml:"max_instances"`  // nil = don't set
	CPUThrottling *bool  `yaml:"cpu_throttling"` // nil = don't set (uses GCP default)
	Memory        string `yaml:"memory"`         // e.g., "2048Mi", "512Mi"
	CPU           string `yaml:"cpu"`            // e.g., "1", "2"
	Concurrency   int    `yaml:"concurrency"`    // max concurrent requests per instance
	Timeout       int    `yaml:"timeout"`        // request timeout in seconds
}

// parseCloudRunConfig parses Cloud Run configuration from config map.
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
