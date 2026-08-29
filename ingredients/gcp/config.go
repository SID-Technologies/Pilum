package gcp

import (
	"strings"

	"github.com/sid-technologies/pilum/lib/configutil"
)

// CloudRunConfig holds GCP Cloud Run specific configuration.
type CloudRunConfig struct {
	MinInstances      int      // Minimum number of instances (0 for scale to zero, -1 for not set)
	MaxInstances      int      // Maximum number of instances (-1 for not set)
	CPUThrottling     bool     // Enable CPU throttling
	Memory            string   // Memory limit (e.g., "512Mi", "1Gi")
	CPU               string   // CPU limit (e.g., "1", "2")
	Concurrency       int      // Max concurrent requests per instance (-1 for not set)
	TimeoutSeconds    int      // Request timeout in seconds (-1 for not set)
	CloudSQLInstances []string // Cloud SQL instance connections (e.g., "project:region:instance")
	Port              int      // Port is the ingress container's listening port. Required by Cloud Run in multi-container mode

	// AllowUnauthenticated exposes the service to the public internet.Defaults to FALSE.
	AllowUnauthenticated bool

	// Ingress is who may REACH the service: "all", "internal-and-cloud-load-balancing", or "internal".
	Ingress string

	// VPCConnector routes the service's outbound traffic through a Serverless
	// VPC Access connector. Empty leaves the setting untouched.
	VPCConnector string

	// VPCEgress selects WHICH traffic takes the connector: "all-traffic" or "private-ranges-only".
	VPCEgress string
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
		MinInstances:         configutil.GetInt(cloudRunMap, "min_instances", -1),
		MaxInstances:         configutil.GetInt(cloudRunMap, "max_instances", -1),
		CPUThrottling:        configutil.GetBool(cloudRunMap, "cpu_throttling", false),
		Memory:               configutil.GetString(cloudRunMap, "memory", ""),
		CPU:                  configutil.GetString(cloudRunMap, "cpu", ""),
		Concurrency:          configutil.GetInt(cloudRunMap, "concurrency", -1),
		TimeoutSeconds:       configutil.GetInt(cloudRunMap, "timeout_seconds", -1),
		CloudSQLInstances:    configutil.GetStringSlice(cloudRunMap, "cloudsql_instances"),
		Port:                 configutil.GetInt(cloudRunMap, "port", 0),
		Ingress:              configutil.GetString(cloudRunMap, "ingress", ""),
		VPCConnector:         configutil.GetString(cloudRunMap, "vpc_connector", ""),
		VPCEgress:            configutil.GetString(cloudRunMap, "vpc_egress", ""),
		AllowUnauthenticated: configutil.GetBool(cloudRunMap, "allow_unauthenticated", false),
	}
}

// toGcloudSecretRef converts a secret reference to gcloud --set-secrets format.
// Accepts either:
//   - Full resource path: "projects/my-project/secrets/my-secret/versions/latest" → "my-secret:latest"
//   - Already formatted:  "my-secret:latest" → "my-secret:latest" (pass-through)
func toGcloudSecretRef(value string) string {
	// Already in gcloud format (contains colon, no slash path)
	if strings.Contains(value, ":") && !strings.Contains(value, "/") {
		return value
	}

	// Parse full resource path: projects/{project}/secrets/{name}/versions/{version}
	parts := strings.Split(value, "/")
	if len(parts) >= 6 && parts[2] == "secrets" && parts[4] == "versions" {
		secretName := parts[3]
		version := parts[5]
		return secretName + ":" + version
	}

	// Unrecognized format — pass through as-is
	return value
}
