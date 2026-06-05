package gcp

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateGCPDeployCommand builds the gcloud invocation for a Cloud Run deploy.
// With sidecars, emits per-container flag groups; gcloud requires all
// service-level flags before the first --container.
func GenerateGCPDeployCommand(svc serviceinfo.ServiceInfo, imageName string) []string {
	cfg := ParseCloudRunConfig(svc.Config)

	cmd := []string{
		"gcloud",
		"run",
		"deploy",
		svc.Name,
		"--region", svc.Region,
		"--platform", "managed",
		"--allow-unauthenticated",
	}

	cmd = appendServiceLevelFlags(cmd, cfg, svc.Project)

	if len(svc.Sidecars) == 0 {
		return appendSingleContainerFlags(cmd, svc, imageName, cfg)
	}

	return appendMultiContainerFlags(cmd, svc, imageName, cfg)
}

func appendServiceLevelFlags(cmd []string, cfg CloudRunConfig, project string) []string {
	if cfg.CPUThrottling {
		cmd = append(cmd, "--cpu-throttling")
	}

	if cfg.MinInstances >= 0 {
		cmd = append(cmd, fmt.Sprintf("--min-instances=%d", cfg.MinInstances))
	}

	if cfg.MaxInstances > 0 {
		cmd = append(cmd, fmt.Sprintf("--max-instances=%d", cfg.MaxInstances))
	}

	if cfg.Concurrency > 0 {
		cmd = append(cmd, fmt.Sprintf("--concurrency=%d", cfg.Concurrency))
	}

	if cfg.TimeoutSeconds > 0 {
		cmd = append(cmd, fmt.Sprintf("--timeout=%d", cfg.TimeoutSeconds))
	}

	if len(cfg.CloudSQLInstances) > 0 {
		cmd = append(cmd, "--add-cloudsql-instances", strings.Join(cfg.CloudSQLInstances, ","))
	}

	if project != "" {
		cmd = append(cmd, "--project", project)
	}

	return cmd
}

func appendSingleContainerFlags(cmd []string, svc serviceinfo.ServiceInfo, imageName string, cfg CloudRunConfig) []string {
	cmd = append(cmd, "--image", imageName)

	if cfg.Memory != "" {
		cmd = append(cmd, "--memory", cfg.Memory)
	}

	if cfg.CPU != "" {
		cmd = append(cmd, "--cpu", cfg.CPU)
	}

	if len(svc.EnvVars) > 0 {
		cmd = append(cmd, "--set-env-vars", joinEnvVars(svc.EnvVars))
	}

	if len(svc.Secrets) > 0 {
		cmd = append(cmd, "--set-secrets", joinSecretRefs(svc.Secrets))
	}

	return cmd
}

func appendMultiContainerFlags(cmd []string, svc serviceinfo.ServiceInfo, imageName string, cfg CloudRunConfig) []string {
	cmd = append(cmd, "--container", ingressContainerName(svc))
	cmd = append(cmd, "--image", imageName)
	cmd = append(cmd, fmt.Sprintf("--port=%d", ingressContainerPort(cfg)))

	if cfg.Memory != "" {
		cmd = append(cmd, "--memory", cfg.Memory)
	}

	if cfg.CPU != "" {
		cmd = append(cmd, "--cpu", cfg.CPU)
	}

	if len(svc.EnvVars) > 0 {
		cmd = append(cmd, "--set-env-vars", joinEnvVars(svc.EnvVars))
	}

	if len(svc.Secrets) > 0 {
		cmd = append(cmd, "--set-secrets", joinSecretRefs(svc.Secrets))
	}

	for _, sc := range svc.Sidecars {
		cmd = appendSidecarFlags(cmd, sc)
	}

	return cmd
}

func appendSidecarFlags(cmd []string, sc serviceinfo.Sidecar) []string {
	cmd = append(cmd, "--container", sc.Name)
	cmd = append(cmd, "--image", sc.Image)

	if sc.Memory != "" {
		cmd = append(cmd, "--memory", sc.Memory)
	}

	if sc.CPU != "" {
		cmd = append(cmd, "--cpu", sc.CPU)
	}

	if len(sc.EnvVars) > 0 {
		cmd = append(cmd, "--set-env-vars", joinEnvVars(sc.EnvVars))
	}

	if len(sc.Secrets) > 0 {
		cmd = append(cmd, "--set-secrets", joinSecretRefs(sc.Secrets))
	}

	if len(sc.DependsOn) > 0 {
		cmd = append(cmd, fmt.Sprintf("--depends-on=%s", strings.Join(sc.DependsOn, ",")))
	}

	if len(sc.Args) > 0 {
		cmd = append(cmd, "--args="+strings.Join(sc.Args, ","))
	}

	if len(sc.Command) > 0 {
		cmd = append(cmd, "--command="+strings.Join(sc.Command, ","))
	}

	return cmd
}

func ingressContainerName(svc serviceinfo.ServiceInfo) string {
	return svc.Name
}

// ingressContainerPort defaults to 8080 to match single-container Cloud Run.
func ingressContainerPort(cfg CloudRunConfig) int {
	if cfg.Port > 0 {
		return cfg.Port
	}

	return 8080
}

func joinEnvVars(envs []serviceinfo.EnvVars) string {
	parts := make([]string, 0, len(envs))
	for _, env := range envs {
		parts = append(parts, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	return strings.Join(parts, ",")
}

func joinSecretRefs(secrets []serviceinfo.Secrets) string {
	parts := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		parts = append(parts, fmt.Sprintf("%s=%s", secret.Name, toGcloudSecretRef(secret.Value)))
	}

	return strings.Join(parts, ",")
}
