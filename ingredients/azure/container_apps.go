package azure

import (
	"fmt"
	"slices"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/shellutil"
)

func GenerateDeployCommand(svc serviceinfo.ServiceInfo, imageName string) string {
	cfg := ParseContainerAppConfig(svc.Config)
	q := shellutil.Quote

	up := []string{
		"az", "containerapp", "up",
		"--name", q(svc.Name),
		"--resource-group", q(svc.Project),
		"--image", q(imageName),
	}
	if cfg.Environment != "" {
		up = append(up, "--environment", q(cfg.Environment))
	}
	if len(svc.EnvVars) > 0 {
		up = append(up, "--env-vars")
		up = append(up, envVarArgs(svc.EnvVars)...)
	}
	if cfg.IngressPort > 0 {
		up = append(up, "--target-port", fmt.Sprintf("%d", cfg.IngressPort))
		if cfg.IngressExternal {
			up = append(up, "--ingress", "external")
		} else {
			up = append(up, "--ingress", "internal")
		}
	}

	// Resource limits are update-only flags; emit the chaser command only
	// when at least one is configured.
	update := []string{
		"az", "containerapp", "update",
		"--name", q(svc.Name),
		"--resource-group", q(svc.Project),
	}
	hasUpdate := false
	if cfg.MinReplicas >= 0 {
		update = append(update, fmt.Sprintf("--min-replicas=%d", cfg.MinReplicas))
		hasUpdate = true
	}
	if cfg.MaxReplicas > 0 {
		update = append(update, fmt.Sprintf("--max-replicas=%d", cfg.MaxReplicas))
		hasUpdate = true
	}
	if cfg.CPU != "" {
		update = append(update, "--cpu", q(cfg.CPU))
		hasUpdate = true
	}
	if cfg.Memory != "" {
		update = append(update, "--memory", q(cfg.Memory))
		hasUpdate = true
	}

	cmd := strings.Join(up, " ")
	if hasUpdate {
		cmd += " && " + strings.Join(update, " ")
	}
	return cmd
}

// envVarArgs renders env vars as sorted, quoted KEY=VALUE arguments.
func envVarArgs(envs []serviceinfo.EnvVars) []string {
	args := make([]string, 0, len(envs))
	for _, env := range envs {
		args = append(args, env.Name+"="+env.Value)
	}
	slices.Sort(args)
	for i, arg := range args {
		args[i] = shellutil.Quote(arg)
	}
	return args
}
