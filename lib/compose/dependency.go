package compose

import (
	"fmt"
	"sort"
)

// MergeDefaults fills empty Dependency fields from the preset defaults.
// Fields explicitly set on the Dependency are not overridden.
func MergeDefaults(dep *Dependency) {
	preset, ok := GetPreset(dep.Type)
	if !ok {
		return // custom type — no defaults to apply
	}

	if dep.Image == "" {
		dep.Image = preset.Image
	}
	if dep.Port == 0 {
		dep.Port = preset.DefaultPort
	}
	if dep.User == "" {
		dep.User = preset.DefaultUser
	}
	if dep.Password == "" {
		dep.Password = preset.DefaultPassword
	}
	if dep.Database == "" {
		dep.Database = preset.DefaultDatabase
	}
}

// BuildDependencyService creates a compose Service for an infrastructure dependency.
// Returns the service and a list of volume names to register at the top level.
func BuildDependencyService(dep Dependency) (*Service, []string) {
	MergeDefaults(&dep)

	svc := &Service{
		Image:         dep.Image,
		ContainerName: dep.Name,
		Restart:       "unless-stopped",
		Networks:      []string{"app-network"},
	}

	// Port mapping
	if dep.Port > 0 {
		svc.Ports = []string{fmt.Sprintf("%d:%d", dep.Port, dep.Port)}
	}

	// Environment — from preset or custom
	var env []string
	preset, isPreset := GetPreset(dep.Type)
	if isPreset && preset.Env != nil {
		env = preset.Env(dep)
	}
	for k, v := range dep.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	if len(env) > 0 {
		sort.Strings(env)
		svc.Environment = env
	}

	// Health check
	var healthCmd string
	if dep.Healthcheck != "" {
		healthCmd = dep.Healthcheck
	} else if isPreset && preset.HealthCmd != nil {
		healthCmd = preset.HealthCmd(dep)
	}
	if healthCmd != "" {
		svc.Healthcheck = &Healthcheck{
			Test:        []string{"CMD-SHELL", healthCmd},
			Interval:    "10s",
			Timeout:     "5s",
			Retries:     5,
			StartPeriod: "10s",
		}
	}

	// Volumes
	var volumeNames []string
	if isPreset && preset.Volumes != nil {
		svc.Volumes = preset.Volumes(dep)
		// Extract volume names (the part before the colon) for top-level registration
		for _, v := range svc.Volumes {
			for i, c := range v {
				if c == ':' {
					name := v[:i]
					// Only register named volumes, not bind mounts (starting with . or /)
					if len(name) > 0 && name[0] != '.' && name[0] != '/' {
						volumeNames = append(volumeNames, name)
					}
					break
				}
			}
		}
	}

	return svc, volumeNames
}

// RenderConnectionString renders the connection string for a dependency.
// Returns "" for custom types or if no template is defined.
func RenderConnectionString(dep Dependency) string {
	MergeDefaults(&dep)

	preset, ok := GetPreset(dep.Type)
	if !ok || preset.ConnStringTmpl == nil {
		return ""
	}

	return preset.ConnStringTmpl(dep)
}
