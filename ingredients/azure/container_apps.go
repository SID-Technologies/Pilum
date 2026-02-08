package azure

import (
	"fmt"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateDeployCommand generates an `az containerapp update` command.
func GenerateDeployCommand(svc serviceinfo.ServiceInfo, imageName string) []string {
	cfg := ParseContainerAppConfig(svc.Config)

	cmd := []string{
		"az",
		"containerapp",
		"update",
		"--name", svc.Name,
		"--resource-group", svc.Project,
		"--image", imageName,
	}

	// Add environment if set
	if cfg.Environment != "" {
		cmd = append(cmd, "--environment", cfg.Environment)
	}

	// Add min replicas if set
	if cfg.MinReplicas >= 0 {
		cmd = append(cmd, fmt.Sprintf("--min-replicas=%d", cfg.MinReplicas))
	}

	// Add max replicas if set
	if cfg.MaxReplicas > 0 {
		cmd = append(cmd, fmt.Sprintf("--max-replicas=%d", cfg.MaxReplicas))
	}

	// Add CPU if set
	if cfg.CPU != "" {
		cmd = append(cmd, "--cpu", cfg.CPU)
	}

	// Add memory if set
	if cfg.Memory != "" {
		cmd = append(cmd, "--memory", cfg.Memory)
	}

	// Add ingress settings
	if cfg.IngressPort > 0 {
		cmd = append(cmd, "--target-port", fmt.Sprintf("%d", cfg.IngressPort))
		if cfg.IngressExternal {
			cmd = append(cmd, "--ingress", "external")
		} else {
			cmd = append(cmd, "--ingress", "internal")
		}
	}

	return cmd
}
