package gcp

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateGCPDeployCommand builds the gcloud invocation that deploys a Cloud
// Run service. When the service declares no sidecars, this emits the
// single-container deploy that Pilum has always emitted. When sidecars are
// present, this emits a multi-container deploy using gcloud's `--container`
// flag groups — flag ordering is significant: every service-level flag must
// appear before the first `--container` flag.
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

// appendServiceLevelFlags adds every flag that's NOT scoped to a specific
// container. gcloud requires all of these before any `--container` flag,
// otherwise multi-container deploys fail with a flag-ordering error. Kept
// shared between the single- and multi-container paths so the same flags
// emit identically in both shapes.
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

// appendSingleContainerFlags preserves the historical single-container shape
// of the deploy command — flags like --image, --memory, --set-env-vars are
// emitted at the top level (not scoped under a --container=NAME).
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

// appendMultiContainerFlags emits one `--container=NAME` group per container,
// starting with the ingress (the main service container) and then each
// sidecar. Cloud Run requires exactly one ingress container with an explicit
// --port (no default port applies when sidecars are present).
//
// Container-level flags (--image, --memory, --cpu, --set-env-vars, etc.)
// MUST appear after their `--container=NAME` declaration; they bind to the
// most-recently-named container until the next --container flips the scope.
func appendMultiContainerFlags(cmd []string, svc serviceinfo.ServiceInfo, imageName string, cfg CloudRunConfig) []string {
	// Ingress container — same identity as the single-container case, just
	// emitted under a named scope.
	cmd = append(cmd, "--container", ingressContainerName(svc))
	cmd = append(cmd, "--image", imageName)

	// Cloud Run requires the ingress container's port to be set explicitly
	// when sidecars are present. We use the configured concurrency-port
	// (defaulting to 8080) — operators who run on a different port should
	// already have set it via env or flag elsewhere; this matches the
	// implicit single-container default.
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

// appendSidecarFlags emits a single sidecar's --container group.
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

// ingressContainerName returns the name we use for the main app container in
// a multi-container revision. Defaulting to the service name keeps the
// `--depends-on` references customers write in sidecar configs predictable:
// they reference `<service-name>` to wait on the ingress container.
func ingressContainerName(svc serviceinfo.ServiceInfo) string {
	return svc.Name
}

// ingressContainerPort returns the port the ingress container listens on.
// Cloud Run requires this be set explicitly in multi-container mode; in
// single-container mode it defaults implicitly to 8080. We mirror that
// default unless an operator has overridden it via the recipe.
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
