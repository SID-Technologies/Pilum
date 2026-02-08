package gcp

import (
	"fmt"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateStatusCommand generates a gcloud command to describe a Cloud Run service.
func GenerateStatusCommand(svc serviceinfo.ServiceInfo) []string {
	cmd := []string{
		"gcloud", "run", "services", "describe", svc.Name,
		"--region", svc.Region,
		"--format", "json",
	}
	if svc.Project != "" {
		cmd = append(cmd, "--project", svc.Project)
	}
	return cmd
}

// GenerateJobStatusCommand generates a gcloud command to describe a Cloud Run Job.
func GenerateJobStatusCommand(svc serviceinfo.ServiceInfo) []string {
	cmd := []string{
		"gcloud", "run", "jobs", "describe", svc.Name,
		"--region", svc.Region,
		"--format", "json",
	}
	if svc.Project != "" {
		cmd = append(cmd, "--project", svc.Project)
	}
	return cmd
}

// GenerateLogsCommand generates a gcloud command to read logs for a Cloud Run service.
func GenerateLogsCommand(svc serviceinfo.ServiceInfo, lines int, _ bool) []string {
	cmd := []string{
		"gcloud", "run", "services", "logs", "read", svc.Name,
		"--region", svc.Region,
	}
	if lines > 0 {
		cmd = append(cmd, fmt.Sprintf("--limit=%d", lines))
	}
	if svc.Project != "" {
		cmd = append(cmd, "--project", svc.Project)
	}
	return cmd
}

// GenerateJobLogsCommand generates a gcloud command to list executions of a Cloud Run Job.
func GenerateJobLogsCommand(svc serviceinfo.ServiceInfo, lines int, _ bool) []string {
	cmd := []string{
		"gcloud", "run", "jobs", "executions", "list",
		"--job", svc.Name,
		"--region", svc.Region,
	}
	if lines > 0 {
		cmd = append(cmd, fmt.Sprintf("--limit=%d", lines))
	}
	if svc.Project != "" {
		cmd = append(cmd, "--project", svc.Project)
	}
	return cmd
}
