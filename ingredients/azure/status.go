package azure

import (
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateStatusCommand generates an az CLI command to describe a Container App.
func GenerateStatusCommand(svc serviceinfo.ServiceInfo) []string {
	return []string{
		"az", "containerapp", "show",
		"--name", svc.Name,
		"--resource-group", svc.Project,
		"--output", "json",
	}
}

// GenerateLogsCommand generates an az CLI command to read Container App logs.
func GenerateLogsCommand(svc serviceinfo.ServiceInfo, follow bool) []string {
	cmd := []string{
		"az", "containerapp", "logs", "show",
		"--name", svc.Name,
		"--resource-group", svc.Project,
		"--type", "console",
	}
	if follow {
		cmd = append(cmd, "--follow")
	}
	return cmd
}
