package build

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateBuildCommand creates a build command from service configuration.
// Returns the command to execute and the image name for downstream use.
//
// Image naming precedence:
//   - service.ImageName (from pilum.yaml `image_name:`) overrides service.Name
//     for the image part of the reference. Use this when the Pilum service
//     name and the image name should differ (e.g., a platform-shared image
//     where the service is `sid-otel-collector` but the image is just
//     `otel-collector`).
//   - tag (CLI `--tag`) overrides service.Version (from pilum.yaml `version:`)
//     which in turn overrides the default `latest`. This lets CI inject a
//     git-sha tag while still allowing pilum.yaml to pin a stable platform
//     version for deliberate-release types like gcp-artifact-registry-image.
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

	imageBaseName := imageBaseName(service)

	// Construct image name using provider-specific formatting.
	// The registry parameter is for CLI overrides only (full registry URL).
	// If empty, use service configuration with provider-specific paths.
	var imageName string
	if registry != "" && !isRegistryName(registry, service) {
		// CLI override with full registry path
		imageName = fmt.Sprintf("%s/%s", registry, imageBaseName)
	} else {
		// Use provider-specific formatting
		imageName = generateProviderImageName(service, imageBaseName)
	}

	imageName = fmt.Sprintf("%s:%s", imageName, resolveTag(service, tag))

	// Wrap in shell for execution
	fullCmd := []string{"/bin/sh", "-c", command}

	return fullCmd, imageName
}

// imageBaseName returns the leaf segment of the image reference (the part
// AFTER the registry/repo). Defaults to the Pilum service name; overridable
// via `image_name:` in pilum.yaml for types like gcp-artifact-registry-image
// where the service identifier and the image identifier should differ.
func imageBaseName(service serviceinfo.ServiceInfo) string {
	if service.ImageName != "" {
		return service.ImageName
	}

	return service.Name
}

// resolveTag picks the image tag with CLI > pilum.yaml > "latest" precedence.
// CLI wins so CI can inject git-sha tags; pilum.yaml `version:` wins over
// "latest" so platform-image types can pin a deliberate version in source.
func resolveTag(service serviceinfo.ServiceInfo, cliTag string) string {
	if cliTag != "" {
		return cliTag
	}

	if service.Version != "" {
		return service.Version
	}

	return "latest"
}

// isRegistryName checks if the registry param matches the service's RegistryName
// (meaning it was passed through from service config, not a CLI override).
func isRegistryName(registry string, service serviceinfo.ServiceInfo) bool {
	return registry == service.RegistryName
}

// generateProviderImageName creates the full image name using provider-specific formatting.
// `imageBase` is the leaf segment (caller-resolved from ImageName or Name).
func generateProviderImageName(service serviceinfo.ServiceInfo, imageBase string) string {
	switch service.Provider {
	case "gcp":
		if service.Region != "" && service.Project != "" {
			registryName := service.RegistryName
			if registryName == "" {
				registryName = service.Project
			}
			return fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s",
				service.Region, service.Project, registryName, imageBase)
		}
	case "aws":
		if service.RegistryName != "" && service.Region != "" {
			return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s",
				service.RegistryName, service.Region, imageBase)
		}
	case "azure":
		if service.RegistryName != "" {
			return fmt.Sprintf("%s.azurecr.io/%s", service.RegistryName, imageBase)
		}
	case "github":
		if service.RegistryName != "" {
			return fmt.Sprintf("ghcr.io/%s/%s", service.RegistryName, imageBase)
		}
	case "dockerhub":
		return fmt.Sprintf("docker.io/%s", imageBase)
	default:
		// Unknown provider, use image base name only
	}
	return imageBase
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
