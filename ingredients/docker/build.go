package docker

import (
	"fmt"
	"strings"

	service "github.com/sid-technologies/pilum/lib/service_info"
)

func GenerateDockerBuildCommand(service service.ServiceInfo, imageName, templatePath string) []string {
	cmd := []string{
		"docker", "build",
		"-t", imageName,
	}

	// Also tag :latest locally when the primary tag is versioned.
	// Push only publishes the explicit tag the remote :latest pointer still moves exclusively via CI.
	if latest := latestAlias(imageName); latest != "" {
		cmd = append(cmd, "-t", latest)
	}

	cmd = append(cmd, "--build-arg", "SERVICE_NAME="+service.Path)

	if len(service.EnvVars) > 0 {
		var envArgs []string
		for _, env := range service.EnvVars {
			envArgs = append(envArgs, fmt.Sprintf("%s=%s", env.Name, env.Value))
		}
		cmd = append(cmd, "--build-arg", strings.Join(envArgs, ","))
	}

	cmd = append(cmd, "-f", templatePath, ".")

	return cmd
}

// latestAlias returns imageName with its tag swapped for :latest, or "" when there is nothing to alias (no tag, or the tag already is latest). The last
// ":" is only a tag separator if no "/" follows it — a ":" before a slash is a registry port (e.g. localhost:5000/img).
func latestAlias(imageName string) string {
	idx := strings.LastIndex(imageName, ":")
	if idx == -1 || strings.Contains(imageName[idx:], "/") {
		return ""
	}

	if imageName[idx+1:] == "latest" {
		return ""
	}

	return imageName[:idx] + ":latest"
}
