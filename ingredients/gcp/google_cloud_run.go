package gcp

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

func GenerateGCPDeployCommand(svc serviceinfo.ServiceInfo, imageName string) []string {
	// Parse Cloud Run config from the raw service config
	cfg := ParseCloudRunConfig(svc.Config)

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

	// Add CPU throttling if enabled
	if cfg.CPUThrottling {
		cmd = append(cmd, "--cpu-throttling")
	}

	// Add min instances if set (>= 0)
	if cfg.MinInstances >= 0 {
		cmd = append(cmd, fmt.Sprintf("--min-instances=%d", cfg.MinInstances))
	}

	// Add max instances if set (> 0)
	if cfg.MaxInstances > 0 {
		cmd = append(cmd, fmt.Sprintf("--max-instances=%d", cfg.MaxInstances))
	}

	// Add memory if set
	if cfg.Memory != "" {
		cmd = append(cmd, "--memory", cfg.Memory)
	}

	// Add CPU if set
	if cfg.CPU != "" {
		cmd = append(cmd, "--cpu", cfg.CPU)
	}

	// Add concurrency if set (> 0)
	if cfg.Concurrency > 0 {
		cmd = append(cmd, fmt.Sprintf("--concurrency=%d", cfg.Concurrency))
	}

	// Add timeout if set (> 0)
	if cfg.TimeoutSeconds > 0 {
		cmd = append(cmd, fmt.Sprintf("--timeout=%d", cfg.TimeoutSeconds))
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

	// Add project if set
	if svc.Project != "" {
		cmd = append(cmd, "--project", svc.Project)
	}

	return cmd
}
