package docker

import (
	"fmt"

	service "github.com/sid-technologies/centurion/lib/service_info"
)

func GenerateImageName(service service.ServiceInfo, registry string, tag string) string {
	var imageName string
	var suffix string

	if tag != "" {
		suffix = ":" + tag
	} else {
		suffix = ":latest"
	}

	if registry != "" {
		imageName = fmt.Sprintf("%s/%s", registry, service.Name)
		return fmt.Sprintf("%s:%s", imageName, suffix)
	}

	switch service.Provider {
	case "aws":
		imageName = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s",
			service.RegistryName, // Assuming RegistryName is the account ID
			service.Region,
			service.Name,
		)
	case "gcp":
		imageName = fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s",
			service.Region,
			service.Project,
			service.RegistryName, // Use the registry name for the GCP repo
			service.Name,
		)
	case "azure":
		imageName = fmt.Sprintf("%s.azurecr.io/%s",
			service.RegistryName, // Use the registry name for Azure CR
			service.Name,
		)
	case "dockerhub":
		imageName = fmt.Sprintf("docker.io/%s", service.Name) // You may want to include username/repo
	case "gitlab":
		imageName = fmt.Sprintf("%s.gitlab.io/%s",
			service.RegistryName, // You may want to include the specific GitLab instance or organization
			service.Name,
		)
	case "github":
		imageName = fmt.Sprintf("ghcr.io/%s/%s",
			service.RegistryName, // Use the GitHub username or organization
			service.Name,
		)

	default:
		imageName = service.Name
	}

	return fmt.Sprintf("%s:%s", imageName, suffix)
}
