package gcp

import (
	"fmt"
	"strings"

	service "github.com/sid-technologies/pilum/lib/service_info"
)

func GenerateGCPDeployCommand(svc service.ServiceInfo, imageName string) []string {
	cmd := []string{
		"gcloud",
		"run",
		"deploy",
		svc.Name,
		"--image", imageName,
		"--region", svc.Region,
		"--platform", "managed",
		"--allow-unauthenticated",
	}

	// Always include project if set
	if svc.Project != "" {
		cmd = append(cmd, "--project", svc.Project)
	}

	// Add Cloud Run specific configuration
	cr := svc.CloudRunConfig

	if cr.MinInstances != nil {
		cmd = append(cmd, fmt.Sprintf("--min-instances=%d", *cr.MinInstances))
	}

	if cr.MaxInstances != nil {
		cmd = append(cmd, fmt.Sprintf("--max-instances=%d", *cr.MaxInstances))
	}

	if cr.CPUThrottling != nil {
		if *cr.CPUThrottling {
			cmd = append(cmd, "--cpu-throttling")
		} else {
			cmd = append(cmd, "--no-cpu-throttling")
		}
	}

	if cr.Memory != "" {
		cmd = append(cmd, "--memory", cr.Memory)
	}

	if cr.CPU != "" {
		cmd = append(cmd, "--cpu", cr.CPU)
	}

	if cr.Concurrency > 0 {
		cmd = append(cmd, fmt.Sprintf("--concurrency=%d", cr.Concurrency))
	}

	if cr.Timeout > 0 {
		cmd = append(cmd, fmt.Sprintf("--timeout=%d", cr.Timeout))
	}

	// Add secrets if provided
	if len(svc.Secrets) > 0 {
		var secretsStrs []string
		for _, secret := range svc.Secrets {
			secretStr := fmt.Sprintf("%s=%s", secret.Name, secret.Value)
			secretsStrs = append(secretsStrs, secretStr)
		}
		// Join the secrets into a single string
		secretsJoined := strings.Join(secretsStrs, ",")
		cmd = append(cmd, "--set-secrets", secretsJoined)
	}

	return cmd
}
