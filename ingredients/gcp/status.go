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

// GenerateLogsCommand generates a gcloud command to read logs for a Cloud Run
// service, or to stream them when follow is set.
func GenerateLogsCommand(svc serviceinfo.ServiceInfo, lines int, follow bool) []string {
	if follow {
		return tailLogsCommand("services", svc)
	}

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

// GenerateJobLogsCommand generates a gcloud command to list executions of a
// Cloud Run Job, or to stream its logs when follow is set.
func GenerateJobLogsCommand(svc serviceinfo.ServiceInfo, lines int, follow bool) []string {
	if follow {
		return tailLogsCommand("jobs", svc)
	}

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

// tailLogsCommand streams logs with `gcloud beta run {services|jobs} logs
// tail`, which needs gcloud's log-streaming component.
func tailLogsCommand(kind string, svc serviceinfo.ServiceInfo) []string {
	cmd := []string{
		"gcloud", "beta", "run", kind, "logs", "tail", svc.Name,
		"--region", svc.Region,
	}
	if svc.Project != "" {
		cmd = append(cmd, "--project", svc.Project)
	}
	return cmd
}
