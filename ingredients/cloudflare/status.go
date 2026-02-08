// Package cloudflare provides command generators for Cloudflare services.
package cloudflare

import (
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateStatusCommand generates a wrangler command to list Pages deployments.
func GenerateStatusCommand(svc serviceinfo.ServiceInfo) []string {
	return []string{
		"npx", "wrangler", "pages", "deployment", "list",
		"--project-name", svc.Name,
	}
}

// GenerateLogsCommand generates a wrangler command to list Pages deployments.
func GenerateLogsCommand(svc serviceinfo.ServiceInfo) []string {
	return []string{
		"npx", "wrangler", "pages", "deployment", "list",
		"--project-name", svc.Name,
	}
}
