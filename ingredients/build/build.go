package build

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateBuildCommand creates a build command from service configuration.
// Returns the command to execute and the image name for downstream use.
func GenerateBuildCommand(service serviceinfo.ServiceInfo, registry, tag string) ([]string, string) {
	buildCmd := service.BuildConfig.Cmd
	if buildCmd == "" {
		return nil, ""
	}

	// Start with the base command
	command := buildCmd

	// Add build flags (e.g., ldflags)
	for _, flag := range service.BuildConfig.Flags {
		if len(flag.Values) == 0 {
			continue
		}
		vals := strings.Join(flag.Values, " ")
		command = fmt.Sprintf("%s -%s='%s'", command, flag.Name, vals)
	}

	// Construct image name for downstream docker operations
	var imageName string
	if registry != "" {
		imageName = fmt.Sprintf("%s/%s", registry, service.Name)
	} else if service.Provider == "gcp" && service.Region != "" && service.Project != "" {
		imageName = fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s",
			service.Region, service.Project, service.Project, service.Name)
	} else {
		imageName = service.Name
	}

	if tag != "" {
		imageName = fmt.Sprintf("%s:%s", imageName, tag)
	} else {
		imageName = fmt.Sprintf("%s:latest", imageName)
	}

	// Wrap in shell for execution
	fullCmd := []string{"/bin/sh", "-c", command}

	return fullCmd, imageName
}

// GenerateBuildCommandString returns just the command string for display/dry-run.
func GenerateBuildCommandString(service serviceinfo.ServiceInfo) string {
	buildCmd := service.BuildConfig.Cmd
	if buildCmd == "" {
		return ""
	}

	command := buildCmd

	for _, flag := range service.BuildConfig.Flags {
		if len(flag.Values) == 0 {
			continue
		}
		vals := strings.Join(flag.Values, " ")
		command = fmt.Sprintf("%s -%s='%s'", command, flag.Name, vals)
	}

	return command
}
