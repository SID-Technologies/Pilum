package docker

import (
	"fmt"
	"strings"

	service "github.com/sid-technologies/centurion/lib/service_info"
)

func GenerateDockerBuildCommand(service service.ServiceInfo, imageName, templatePath string) []string {
	cmd := []string{
		"docker", "build",
		"-t", imageName,
		"--build-arg", "SERVICE_NAME=" + service.Name,
	}

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
