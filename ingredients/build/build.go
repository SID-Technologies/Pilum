package build

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// ResolveImageName returns the fully-qualified image reference.
// CLI tag > service.Version > "latest". registry override > provider path.
func ResolveImageName(service serviceinfo.ServiceInfo, registry, tag string) string {
	imageBase := imageBaseName(service)

	var imageName string
	if registry != "" && !isRegistryName(registry, service) {
		imageName = fmt.Sprintf("%s/%s", registry, imageBase)
	} else {
		imageName = generateProviderImageName(service, imageBase)
	}

	return fmt.Sprintf("%s:%s", imageName, resolveTag(service, tag))
}

// GenerateBuildCommand returns the source-compile command and the image name.
// Returns (nil, imageName) for Dockerfile-only services so docker build / push
// downstream still have a tag.
func GenerateBuildCommand(service serviceinfo.ServiceInfo, registry, tag string) ([]string, string) {
	imageName := ResolveImageName(service, registry, tag)

	buildCmd := service.BuildConfig.Cmd
	if buildCmd == "" {
		return nil, imageName
	}

	command := buildCmd

	for _, flag := range service.BuildConfig.Flags {
		if len(flag.Values) == 0 {
			continue
		}

		vals := strings.Join(flag.Values, " ")
		command = fmt.Sprintf("%s -%s='%s'", command, flag.Name, vals)
	}

	return []string{"/bin/sh", "-c", command}, imageName
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
