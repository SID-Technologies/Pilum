package gcp

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateDeployJobCommand generates the gcloud command to create or update a Cloud Run Job.
func GenerateDeployJobCommand(svc serviceinfo.ServiceInfo, imageName string) []string {
	cfg := ParseCloudRunJobConfig(svc.Config)

	cmd := []string{
		"gcloud",
		"run",
		"jobs",
		"deploy",
		svc.Name,
		"--image", imageName,
		"--region", svc.Region,
	}

	// Add task timeout if set
	if cfg.Timeout != "" {
		cmd = append(cmd, "--task-timeout", cfg.Timeout)
	}

	// Add max retries if set (>= 0)
	if cfg.MaxRetries >= 0 {
		cmd = append(cmd, fmt.Sprintf("--max-retries=%d", cfg.MaxRetries))
	}

	// Add parallelism if set (> 0)
	if cfg.Parallelism > 0 {
		cmd = append(cmd, fmt.Sprintf("--parallelism=%d", cfg.Parallelism))
	}

	// Add task count if set (> 0)
	if cfg.TaskCount > 0 {
		cmd = append(cmd, fmt.Sprintf("--tasks=%d", cfg.TaskCount))
	}

	// Add CPU if set
	if cfg.CPU != "" {
		cmd = append(cmd, "--cpu", cfg.CPU)
	}

	// Add memory if set
	if cfg.Memory != "" {
		cmd = append(cmd, "--memory", cfg.Memory)
	}

	// Add service account if set
	if cfg.ServiceAccount != "" {
		cmd = append(cmd, "--service-account", cfg.ServiceAccount)
	}

	// Add Cloud SQL instances if set
	if len(cfg.CloudSQLInstances) > 0 {
		cmd = append(cmd, "--add-cloudsql-instances", strings.Join(cfg.CloudSQLInstances, ","))
	}

	// Add secrets if provided
	if len(svc.Secrets) > 0 {
		var secretsStrs []string
		for _, secret := range svc.Secrets {
			secretStr := fmt.Sprintf("%s=%s", secret.Name, toGcloudSecretRef(secret.Value))
			secretsStrs = append(secretsStrs, secretStr)
		}
		secretsJoined := strings.Join(secretsStrs, ",")
		cmd = append(cmd, "--set-secrets", secretsJoined)
	}

	// Add project if set
	if svc.Project != "" {
		cmd = append(cmd, "--project", svc.Project)
	}

	return cmd
}

// GenerateExecuteJobCommand generates the gcloud command to execute a Cloud Run Job.
// Returns nil if execute_on_deploy is false (caller treats nil as skip).
func GenerateExecuteJobCommand(svc serviceinfo.ServiceInfo) []string {
	cfg := ParseCloudRunJobConfig(svc.Config)

	if !cfg.ExecuteOnDeploy {
		return nil
	}

	cmd := []string{
		"gcloud",
		"run",
		"jobs",
		"execute",
		svc.Name,
		"--region", svc.Region,
		"--wait",
	}

	// Add project if set
	if svc.Project != "" {
		cmd = append(cmd, "--project", svc.Project)
	}

	return cmd
}
