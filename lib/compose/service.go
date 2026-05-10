package compose

import (
	"fmt"
	"sort"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// BuildServiceEntry converts a ServiceInfo into a compose Service.
// Does NOT handle dependency wiring (secret injection, depends_on) — that
// is handled by Generate after all services are built.
func BuildServiceEntry(svc serviceinfo.ServiceInfo, index int, opts Options) *Service {
	cs := &Service{
		Image:         resolveImageName(svc, opts),
		ContainerName: svc.Name,
		Restart:       "unless-stopped",
		Networks:      []string{"app-network"},
	}

	// Environment variables from pilum.yaml and cloud_run config
	var env []string

	// Secrets with dev defaults (non-dependency secrets only)
	for _, secret := range svc.Secrets {
		name := secret.Name
		switch name {
		case "JWT_SECRET":
			env = append(env, fmt.Sprintf("%s=dev-jwt-secret-do-not-use-in-production", name))
		case "ADMIN_KEY":
			env = append(env, fmt.Sprintf("%s=dev-admin-key", name))
		default:
			// Placeholder — dependency wiring in Generate will override matching secrets
			env = append(env, fmt.Sprintf("%s=${%s}", name, name))
		}
	}

	// Env vars from pilum.yaml
	for _, ev := range svc.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", ev.Name, ev.Value))
	}

	// Cloud Run env_vars from raw config
	if cr, ok := svc.Config["cloud_run"].(map[string]any); ok {
		if envVars, ok := cr["env_vars"].(map[string]any); ok {
			for name, value := range envVars {
				env = append(env, fmt.Sprintf("%s=%v", name, value))
			}
		}
	}

	if len(env) > 0 {
		sort.Strings(env)
		cs.Environment = env
	}

	// Health check from config
	if hc, ok := svc.Config["health_check"].(map[string]any); ok {
		hcPath := "/health"
		if p, ok := hc["path"].(string); ok {
			hcPath = p
		}
		startPeriod := "5s"
		if sp, ok := hc["initial_delay"].(string); ok {
			startPeriod = sp
		}
		cs.Healthcheck = &Healthcheck{
			Test:        []string{"CMD-SHELL", fmt.Sprintf("curl -f http://localhost:8080%s || exit 1", hcPath)},
			Interval:    "30s",
			Timeout:     "10s",
			Retries:     3,
			StartPeriod: startPeriod,
		}
	}

	// Runtime: memory + CPU limits
	if rt, ok := svc.Config["runtime"].(map[string]any); ok {
		if mem, ok := rt["memory"].(string); ok {
			if strings.HasSuffix(mem, "Mi") {
				mem = strings.Replace(mem, "Mi", "m", 1)
			}
			cs.MemLimit = mem
		}
		switch cpu := rt["cpu"].(type) {
		case string:
			cs.CPUs = cpu
		case int:
			cs.CPUs = fmt.Sprintf("%d", cpu)
		}
	}

	// Port: explicit from config or auto-assigned
	externalPort := 8080 + index
	if p, ok := svc.Config["port"].(int); ok {
		externalPort = p
	}
	cs.Ports = []string{fmt.Sprintf("%d:8080", externalPort)}

	return cs
}

func resolveImageName(svc serviceinfo.ServiceInfo, opts Options) string {
	switch {
	case opts.Registry != "":
		return fmt.Sprintf("%s/%s:%s", opts.Registry, svc.Name, opts.Tag)
	case svc.Provider == "gcp" && svc.Region != "" && svc.Project != "":
		// GCP Artifact Registry path: {region}-docker.pkg.dev/{project}/{repo}/{image}
		// `repo` (the Artifact Registry repository) comes from the service's
		// `registry_name` field. Fall back to `project` for services that
		// don't set one — mirrors ingredients/docker/helpers.go so the image
		// tag matches what `pilum build` produces. Previously we substituted
		// `project` for both slots, which broke `pilum compose` whenever a
		// service set `registry_name` (image tag mismatch with the build).
		registryName := svc.RegistryName
		if registryName == "" {
			registryName = svc.Project
		}
		return fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s:%s",
			svc.Region, svc.Project, registryName, svc.Name, opts.Tag)
	case svc.RegistryName != "":
		return fmt.Sprintf("%s/%s:%s", svc.RegistryName, svc.Name, opts.Tag)
	default:
		return fmt.Sprintf("%s:%s", svc.Name, opts.Tag)
	}
}
