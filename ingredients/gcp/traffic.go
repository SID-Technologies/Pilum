package gcp

import (
	"fmt"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateRouteToLatestCommand sends 100% of traffic to the latest revision.
//
// Runs after every Cloud Run deploy. Once traffic is pinned to a named
// revision (which is what a rollback does), `gcloud run deploy` creates new
// revisions that receive no traffic until someone runs --to-latest. Without
// this step the deploy after a rollback reports success while production keeps
// serving the rolled-back revision.
//
// Returns nil (step skipped) for a no_traffic deploy: routing would undo it.
func GenerateRouteToLatestCommand(svc serviceinfo.ServiceInfo) []string {
	if ParseCloudRunConfig(svc.Config).NoTraffic {
		return nil
	}

	cmd := []string{
		"gcloud", "run", "services", "update-traffic", svc.Name,
		"--to-latest",
		"--region", svc.Region,
	}
	return withProject(cmd, svc.Project)
}

// GenerateRollbackTrafficCommand pins 100% of traffic to an existing revision.
func GenerateRollbackTrafficCommand(svc serviceinfo.ServiceInfo, revision string) []string {
	cmd := []string{
		"gcloud", "run", "services", "update-traffic", svc.Name,
		fmt.Sprintf("--to-revisions=%s=100", revision),
		"--region", svc.Region,
	}
	return withProject(cmd, svc.Project)
}

// GenerateListRevisionsCommand lists a Cloud Run service's revisions as JSON.
func GenerateListRevisionsCommand(svc serviceinfo.ServiceInfo) []string {
	cmd := []string{
		"gcloud", "run", "revisions", "list",
		"--service", svc.Name,
		"--region", svc.Region,
		"--format", "json",
	}
	return withProject(cmd, svc.Project)
}

// GenerateListJobExecutionsCommand lists a Cloud Run Job's executions as JSON.
// Jobs have no revisions; each execution records the image it ran.
func GenerateListJobExecutionsCommand(svc serviceinfo.ServiceInfo) []string {
	cmd := []string{
		"gcloud", "run", "jobs", "executions", "list",
		"--job", svc.Name,
		"--region", svc.Region,
		"--format", "json",
	}
	return withProject(cmd, svc.Project)
}

// GenerateJobSetImageCommand points a Cloud Run Job at a different image,
// keeping the rest of its current configuration.
func GenerateJobSetImageCommand(svc serviceinfo.ServiceInfo, image string) []string {
	cmd := []string{
		"gcloud", "run", "jobs", "update", svc.Name,
		"--image", image,
		"--region", svc.Region,
	}
	return withProject(cmd, svc.Project)
}

func withProject(cmd []string, project string) []string {
	if project != "" {
		cmd = append(cmd, "--project", project)
	}
	return cmd
}
