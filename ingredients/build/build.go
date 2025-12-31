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

	// Construct image name using provider-specific formatting.
	// The registry parameter is for CLI overrides only (full registry URL).
	// If empty, use service configuration with provider-specific paths.
	var imageName string
	if registry != "" && !isRegistryName(registry, service) {
		// CLI override with full registry path
		imageName = fmt.Sprintf("%s/%s", registry, service.Name)
	} else {
		// Use provider-specific formatting
		imageName = generateProviderImageName(service)
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

// isRegistryName checks if the registry param matches the service's RegistryName
// (meaning it was passed through from service config, not a CLI override).
func isRegistryName(registry string, service serviceinfo.ServiceInfo) bool {
	return registry == service.RegistryName
}

// generateProviderImageName creates the full image name using provider-specific formatting.
func generateProviderImageName(service serviceinfo.ServiceInfo) string {
	switch service.Provider {
	case "gcp":
		if service.Region != "" && service.Project != "" {
			registryName := service.RegistryName
			if registryName == "" {
				registryName = service.Project
			}
			return fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s",
				service.Region, service.Project, registryName, service.Name)
		}
	case "aws":
		if service.RegistryName != "" && service.Region != "" {
			return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s",
				service.RegistryName, service.Region, service.Name)
		}
	case "azure":
		if service.RegistryName != "" {
			return fmt.Sprintf("%s.azurecr.io/%s", service.RegistryName, service.Name)
		}
	case "github":
		if service.RegistryName != "" {
			return fmt.Sprintf("ghcr.io/%s/%s", service.RegistryName, service.Name)
		}
	case "dockerhub":
		return fmt.Sprintf("docker.io/%s", service.Name)
	default:
		// Unknown provider, use service name only
	}
	return service.Name
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
