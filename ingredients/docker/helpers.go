package docker

import (
	"fmt"

	"github.com/sid-technologies/pilum/lib/errors"
	service "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateImageName creates a full image name with tag using provider-specific formatting.
// The registry parameter is for CLI overrides only (full registry URL).
// If empty or matching service.RegistryName, uses provider-specific paths.
// Returns an error if the provider is unknown or required fields are missing.
func GenerateImageName(svc service.ServiceInfo, registry string, tag string) (string, error) {
	var suffix string
	if tag != "" {
		suffix = tag
	} else {
		suffix = "latest"
	}

	// If registry is a full URL override (not just the registry name from service config)
	if registry != "" && registry != svc.RegistryName {
		return fmt.Sprintf("%s/%s:%s", registry, svc.Name, suffix), nil
	}

	// Use provider-specific formatting
	imageName, err := generateProviderImageName(svc)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", imageName, suffix), nil
}

// generateProviderImageName creates the image name using provider-specific formatting.
// Returns an error if the provider is unknown or required fields are missing.
func generateProviderImageName(svc service.ServiceInfo) (string, error) {
	switch svc.Provider {
	case "aws":
		if svc.RegistryName == "" {
			return "", errors.New("service '%s': AWS provider requires registry_name (account ID)", svc.Name)
		}
		if svc.Region == "" {
			return "", errors.New("service '%s': AWS provider requires region", svc.Name)
		}
		return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s",
			svc.RegistryName, svc.Region, svc.Name), nil
	case "gcp":
		if svc.Region == "" {
			return "", errors.New("service '%s': GCP provider requires region", svc.Name)
		}
		if svc.Project == "" {
			return "", errors.New("service '%s': GCP provider requires project", svc.Name)
		}
		registryName := svc.RegistryName
		if registryName == "" {
			registryName = svc.Project
		}
		return fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s",
			svc.Region, svc.Project, registryName, svc.Name), nil
	case "azure":
		if svc.RegistryName == "" {
			return "", errors.New("service '%s': Azure provider requires registry_name", svc.Name)
		}
		return fmt.Sprintf("%s.azurecr.io/%s", svc.RegistryName, svc.Name), nil
	case "dockerhub":
		return fmt.Sprintf("docker.io/%s", svc.Name), nil
	case "gitlab":
		if svc.RegistryName == "" {
			return "", errors.New("service '%s': GitLab provider requires registry_name", svc.Name)
		}
		return fmt.Sprintf("%s.gitlab.io/%s", svc.RegistryName, svc.Name), nil
	case "github":
		if svc.RegistryName == "" {
			return "", errors.New("service '%s': GitHub provider requires registry_name", svc.Name)
		}
		return fmt.Sprintf("ghcr.io/%s/%s", svc.RegistryName, svc.Name), nil
	case "homebrew":
		// Homebrew doesn't use Docker images
		return "", errors.New("service '%s': Homebrew provider does not use Docker images", svc.Name)
	default:
		return "", errors.New("service '%s': unknown provider '%s'", svc.Name, svc.Provider)
	}
}
