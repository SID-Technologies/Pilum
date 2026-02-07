package gcp

import (
	"github.com/sid-technologies/pilum/lib/configutil"
)

// CloudRunJobConfig holds GCP Cloud Run Job specific configuration.
type CloudRunJobConfig struct {
	Timeout         string // Max execution time per task (e.g., "600s", "10m")
	MaxRetries      int    // Number of retries on task failure (-1 for not set)
	Parallelism     int    // Maximum concurrent tasks (-1 for not set)
	TaskCount       int    // Total number of tasks to run (-1 for not set)
	CPU             string // CPUs per task (e.g., "1", "2")
	Memory          string // Memory per task (e.g., "512Mi", "2Gi")
	ServiceAccount  string // Custom service account email for job execution
	ExecuteOnDeploy bool   // Automatically execute job after deployment
}

// ParseCloudRunJobConfig extracts Cloud Run Job configuration from the raw service config.
// Returns a CloudRunJobConfig with sensible defaults for unset values.
func ParseCloudRunJobConfig(config map[string]any) CloudRunJobConfig {
	jobMap := configutil.MapFromAny(config["job"])
	if len(jobMap) == 0 {
		return CloudRunJobConfig{
			Timeout:     "600s",
			MaxRetries:  -1,
			Parallelism: -1,
			TaskCount:   -1,
		}
	}

	return CloudRunJobConfig{
		Timeout:         configutil.GetString(jobMap, "timeout", "600s"),
		MaxRetries:      configutil.GetInt(jobMap, "max_retries", -1),
		Parallelism:     configutil.GetInt(jobMap, "parallelism", -1),
		TaskCount:       configutil.GetInt(jobMap, "task_count", -1),
		CPU:             configutil.GetString(jobMap, "cpu", ""),
		Memory:          configutil.GetString(jobMap, "memory", ""),
		ServiceAccount:  configutil.GetString(jobMap, "service_account", ""),
		ExecuteOnDeploy: configutil.GetBool(jobMap, "execute_on_deploy", false),
	}
}
