package gcp

import (
	"fmt"
	"strings"

	service "github.com/sid-technologies/centurion/lib/service_info"
)

func GenerateGCPDeployCommand(service service.ServiceInfo, imageName string) []string {
	cmd := []string{
		"gcloud",
		"run",
		"deploy",
		service.Name,
		"--image", imageName,
		"--region", service.Region,
		"--platform", "managed",
		"--allow-unauthenticated",
	}

	// Add secrets if provided
	if len(service.Secrets) > 0 {
		var secretsStrs []string
		for _, secret := range service.Secrets {
			secretStr := fmt.Sprintf("%s=%s", secret.Name, secret.Value)
			secretsStrs = append(secretsStrs, secretStr)
		}
		// Join the secrets into a single string
		secretsJoined := strings.Join(secretsStrs, ",")
		cmd = append(cmd, "--set-secrets", secretsJoined)
	}

	return cmd
}
