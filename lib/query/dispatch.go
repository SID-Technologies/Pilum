package query

import (
	"encoding/json"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/ingredients/azure"
	"github.com/sid-technologies/pilum/ingredients/cloudflare"
	"github.com/sid-technologies/pilum/ingredients/gcp"
	"github.com/sid-technologies/pilum/lib/errors"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateStatusCommand returns the CLI command to query a service's status.
func GenerateStatusCommand(svc serviceinfo.ServiceInfo) ([]string, error) {
	switch svc.Type {
	case "gcp-cloud-run":
		return gcp.GenerateStatusCommand(svc), nil
	case "gcp-cloud-run-job":
		return gcp.GenerateJobStatusCommand(svc), nil
	case "azure-container-apps":
		return azure.GenerateStatusCommand(svc), nil
	case "aws-lambda":
		return aws.GenerateStatusCommand(svc), nil
	case "cloudflare-pages":
		return cloudflare.GenerateStatusCommand(svc), nil
	default:
		// Fall back to provider for templates that match the provider name
		switch svc.Provider {
		case "gcp":
			return gcp.GenerateStatusCommand(svc), nil
		case "azure":
			return azure.GenerateStatusCommand(svc), nil
		case "aws":
			return aws.GenerateStatusCommand(svc), nil
		case "cloudflare":
			return cloudflare.GenerateStatusCommand(svc), nil
		}
		return nil, errors.New("unsupported template for status: %s", svc.Type)
	}
}

// GenerateLogsCommand returns the CLI command to query a service's logs.
func GenerateLogsCommand(svc serviceinfo.ServiceInfo, lines int, follow bool) ([]string, error) {
	switch svc.Type {
	case "gcp-cloud-run":
		return gcp.GenerateLogsCommand(svc, lines, follow), nil
	case "gcp-cloud-run-job":
		return gcp.GenerateJobLogsCommand(svc, lines, follow), nil
	case "azure-container-apps":
		return azure.GenerateLogsCommand(svc, follow), nil
	case "aws-lambda":
		return aws.GenerateLogsCommand(svc, follow), nil
	case "cloudflare-pages":
		return cloudflare.GenerateLogsCommand(svc), nil
	default:
		switch svc.Provider {
		case "gcp":
			return gcp.GenerateLogsCommand(svc, lines, follow), nil
		case "azure":
			return azure.GenerateLogsCommand(svc, follow), nil
		case "aws":
			return aws.GenerateLogsCommand(svc, follow), nil
		case "cloudflare":
			return cloudflare.GenerateLogsCommand(svc), nil
		}
		return nil, errors.New("unsupported template for logs: %s", svc.Type)
	}
}

// ParseStatusResponse parses provider-specific JSON output into a ServiceStatus.
func ParseStatusResponse(jsonOutput string, svc serviceinfo.ServiceInfo) ServiceStatus {
	switch svc.Type {
	case "gcp-cloud-run", "gcp-cloud-run-job":
		return parseGCPStatus(jsonOutput, svc)
	case "azure-container-apps":
		return parseAzureStatus(jsonOutput, svc)
	case "aws-lambda":
		return parseAWSStatus(jsonOutput, svc)
	default:
		switch svc.Provider {
		case "gcp":
			return parseGCPStatus(jsonOutput, svc)
		case "azure":
			return parseAzureStatus(jsonOutput, svc)
		case "aws":
			return parseAWSStatus(jsonOutput, svc)
		}
		return ServiceStatus{
			Name:     svc.DisplayName(),
			Provider: svc.Provider,
			Region:   svc.Region,
			Status:   "Unknown",
		}
	}
}

func parseGCPStatus(jsonOutput string, svc serviceinfo.ServiceInfo) ServiceStatus {
	status := ServiceStatus{
		Name:     svc.DisplayName(),
		Provider: svc.Provider,
		Region:   svc.Region,
		Status:   "Unknown",
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &data); err != nil {
		return status
	}

	// Extract status from conditions
	if conditions, ok := extractSlice(data, "status", "conditions"); ok {
		for _, c := range conditions {
			cond, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if getStr(cond, "type") == "Ready" {
				if getStr(cond, "status") == "True" {
					status.Status = "Ready"
				} else {
					status.Status = "NotReady"
				}
			}
		}
	}

	// Extract URL
	if statusMap, ok := data["status"].(map[string]any); ok {
		if url, ok := statusMap["url"].(string); ok {
			status.URL = url
		}
	}

	// Extract image from template
	if tmpl, ok := extractMap(data, "spec", "template", "spec", "containers"); ok {
		if containers, ok := tmpl.([]any); ok && len(containers) > 0 {
			if container, ok := containers[0].(map[string]any); ok {
				if image, ok := container["image"].(string); ok {
					status.Image = image
				}
			}
		}
	}

	// Extract update time
	if meta, ok := data["metadata"].(map[string]any); ok {
		if annotations, ok := meta["annotations"].(map[string]any); ok {
			if ts, ok := annotations["run.googleapis.com/lastModifiedTimestamp"].(string); ok {
				status.Updated = ts
			}
		}
	}

	return status
}

func parseAzureStatus(jsonOutput string, svc serviceinfo.ServiceInfo) ServiceStatus {
	status := ServiceStatus{
		Name:     svc.DisplayName(),
		Provider: svc.Provider,
		Region:   svc.Region,
		Status:   "Unknown",
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &data); err != nil {
		return status
	}

	if props, ok := data["properties"].(map[string]any); ok {
		if state, ok := props["provisioningState"].(string); ok {
			status.Status = state
		}
		if cfg, ok := props["configuration"].(map[string]any); ok {
			if ingress, ok := cfg["ingress"].(map[string]any); ok {
				if fqdn, ok := ingress["fqdn"].(string); ok {
					status.URL = "https://" + fqdn
				}
			}
		}
		if tmpl, ok := props["template"].(map[string]any); ok {
			if containers, ok := tmpl["containers"].([]any); ok && len(containers) > 0 {
				if c, ok := containers[0].(map[string]any); ok {
					if image, ok := c["image"].(string); ok {
						status.Image = image
					}
				}
			}
		}
	}

	if loc, ok := data["location"].(string); ok && status.Region == "" {
		status.Region = loc
	}

	return status
}

func parseAWSStatus(jsonOutput string, svc serviceinfo.ServiceInfo) ServiceStatus {
	status := ServiceStatus{
		Name:     svc.DisplayName(),
		Provider: svc.Provider,
		Region:   svc.Region,
		Status:   "Unknown",
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &data); err != nil {
		return status
	}

	if config, ok := data["Configuration"].(map[string]any); ok {
		if state, ok := config["State"].(string); ok {
			status.Status = state
		}
		if lastMod, ok := config["LastModified"].(string); ok {
			status.Updated = lastMod
		}
	}

	return status
}

// getStr safely extracts a string from a map.
func getStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// extractSlice navigates nested maps and returns a slice at the given path.
func extractSlice(data map[string]any, keys ...string) ([]any, bool) {
	current := any(data)
	for i, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		if i == len(keys)-1 {
			slice, ok := m[key].([]any)
			return slice, ok
		}
		current = m[key]
	}
	return nil, false
}

// extractMap navigates nested maps and returns the value at the given path.
func extractMap(data map[string]any, keys ...string) (any, bool) {
	current := any(data)
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current = m[key]
	}
	return current, current != nil
}
