package gcp

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

func GenerateGCPDeployCommand(svc serviceinfo.ServiceInfo, imageName string) []string {
	cfg := ParseCloudRunConfig(svc.Config)

	cmd := []string{
		"gcloud",
		"run",
		"deploy",
		svc.Name,
		"--region", svc.Region,
		"--platform", "managed",
	}

	if cfg.AllowUnauthenticated {
		cmd = append(cmd, "--allow-unauthenticated")
	} else {
		cmd = append(cmd, "--no-allow-unauthenticated")
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

	if cfg.Ingress != "" {
		cmd = append(cmd, "--ingress", cfg.Ingress)
	}

	if cfg.VPCConnector != "" {
		cmd = append(cmd, "--vpc-connector", cfg.VPCConnector)

		egress := cfg.VPCEgress
		if egress == "" {
			egress = "all-traffic"
		}

		cmd = append(cmd, "--vpc-egress", egress)
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

func ingressContainerPort(cfg CloudRunConfig) int {
	if cfg.Port > 0 {
		return cfg.Port
	}

	return 8080
}

// joinEnvVars formats env vars for gcloud's --set-env-vars flag.
// If any value contains a comma (the default delimiter), it uses
// gcloud's alternative-delimiter syntax `^DELIM^KEY1=VAL1DELIM KEY2=VAL2`.
// See `gcloud topic escaping`.
func joinEnvVars(envs []serviceinfo.EnvVars) string {
	parts := make([]string, 0, len(envs))
	for _, env := range envs {
		parts = append(parts, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	if needsAltDelimiter(envs) {
		delim := pickDelimiter(envs)
		return fmt.Sprintf("^%s^%s", delim, strings.Join(parts, delim))
	}

	return strings.Join(parts, ",")
}

// needsAltDelimiter reports whether any env value contains the default
// comma separator — gcloud parses --set-env-vars as comma-delimited KEY=VAL
// pairs, so a comma in any value breaks the default syntax.
func needsAltDelimiter(envs []serviceinfo.EnvVars) bool {
	for _, env := range envs {
		if strings.Contains(env.Value, ",") {
			return true
		}
	}

	return false
}

// pickDelimiter returns the first single-byte separator not present in any
// env value. Tried in order of "least likely to appear" to keep emitted
// commands readable.
func pickDelimiter(envs []serviceinfo.EnvVars) string {
	candidates := []string{"|", "@", ";", "~", "#", "%"}
	for _, c := range candidates {
		if !anyValueContains(envs, c) {
			return c
		}
	}
	// All common delimiters present — fall back to a control char that's
	// effectively never in real config (file separator).
	return "\x1c"
}

func anyValueContains(envs []serviceinfo.EnvVars, s string) bool {
	for _, env := range envs {
		if strings.Contains(env.Value, s) {
			return true
		}
	}

	return false
}

func joinSecretRefs(secrets []serviceinfo.Secrets) string {
	parts := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		parts = append(parts, fmt.Sprintf("%s=%s", secret.Name, toGcloudSecretRef(secret.Value)))
	}

	return strings.Join(parts, ",")
}
