package serviceinfo

import (
	"github.com/sid-technologies/pilum/lib/configutil"
	"github.com/sid-technologies/pilum/lib/errors"
)

type EnvVars struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Secrets struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type BuildFlag struct {
	Name   string   `yaml:"name"`   // e.g., "ldflags", "gcflags"
	Values []string `yaml:"values"` // e.g., ["-s", "-w"]
}

type BuildResources struct {
	Memory int `yaml:"memory"` // Estimated build memory in MB (0 = use language default)
	CPU    int `yaml:"cpu"`    // Estimated CPU cores needed (0 = 1)
}

type BuildConfig struct {
	Language   string         `yaml:"language"`
	Version    string         `yaml:"version"`
	Cmd        string         `yaml:"cmd"`
	EnvVars    []EnvVars      `yaml:"env_vars"`
	Flags      []BuildFlag    `yaml:"flags"`
	VersionVar string         `yaml:"version_var"` // Go variable path for version injection (e.g., "main.version")
	Resources  BuildResources `yaml:"resources"`
}

type RuntimeConfig struct {
	Service string `yaml:"service"`
}

// Probe describes a container readiness/startup probe. Mirrors the subset of
// the Cloud Run / Knative probe schema we support today (HTTP GET probe).
type Probe struct {
	Path                string `yaml:"path"`                  // HTTP path to probe (e.g., "/")
	Port                int    `yaml:"port"`                  // Port to probe; if 0, falls back to the container's port
	InitialDelaySeconds int    `yaml:"initial_delay_seconds"` // Wait this long before first probe
	PeriodSeconds       int    `yaml:"period_seconds"`        // Probe every N seconds
	TimeoutSeconds      int    `yaml:"timeout_seconds"`       // Fail the probe after N seconds
	FailureThreshold    int    `yaml:"failure_threshold"`     // Mark container unready after N consecutive failures
}

// Sidecar describes an auxiliary container that runs alongside the main app
// container in a Cloud Run multi-container revision.
//
// On Cloud Run, sidecars share the network namespace with the ingress
// container (so `localhost:PORT` reaches the sidecar from the ingress
// container and vice-versa). One ingress container per revision; up to 10
// containers total. Cloud Run pulls each image independently from its
// registry at startup — sidecars are NOT layered on the main image.
type Sidecar struct {
	Name         string    `yaml:"name"`          // Required: container name (unique within the service)
	Image        string    `yaml:"image"`         // Required: fully-qualified image reference
	Port         int       `yaml:"port"`          // Optional: port the sidecar listens on (only required if exposing via ingress, which sidecars typically don't)
	Memory       string    `yaml:"memory"`        // Optional: memory limit (e.g., "128Mi")
	CPU          string    `yaml:"cpu"`           // Optional: CPU limit (e.g., "0.25")
	EnvVars      []EnvVars `yaml:"env_vars"`      // Optional: env vars (YAML map: KEY: VALUE)
	Secrets      []Secrets `yaml:"secrets"`       // Optional: secret refs (YAML map: ENV_NAME: secret-ref)
	Args         []string  `yaml:"args"`          // Optional: override the container's CMD/args
	Command      []string  `yaml:"command"`       // Optional: override the container's ENTRYPOINT
	DependsOn    []string  `yaml:"depends_on"`    // Optional: container-startup ordering — names of OTHER containers in this revision that must be ready before this one starts
	StartupProbe *Probe    `yaml:"startup_probe"` // Optional: HTTP startup probe
}

type ServiceInfo struct {
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description"`
	Type          string         `yaml:"-"` // Recipe type (e.g., "gcp-cloud-run") — from "type" YAML key
	Template      string         `yaml:"template"`
	Path          string         `yaml:"-"`
	Config        map[string]any `yaml:"-"`
	BuildConfig   BuildConfig    `yaml:"build"`
	Runtime       RuntimeConfig  `yaml:"runtime"`
	EnvVars       []EnvVars      `yaml:"env_vars"`
	Secrets       []Secrets      `yaml:"secrets"`
	Region        string         `yaml:"region"`
	Regions       []string       `yaml:"regions"` // For multi-region deployments
	IsMultiRegion bool           `yaml:"-"`       // True if this was expanded from a multi-region config
	Project       string         `yaml:"project"`
	License       string         `yaml:"license"`
	Provider      string         `yaml:"provider"`
	RegistryName  string         `yaml:"registry_name"`
	DependsOn     []string       `yaml:"depends_on"` // Services this service depends on
	Sidecars      []Sidecar      `yaml:"sidecars"`   // Auxiliary containers co-deployed in the same Cloud Run revision

	// ImageName is used by the gcp-artifact-registry-image type — a build/push-only
	// service that produces a versioned image referenced by other services'
	// sidecars. Ignored by deploy-shaped types.
	ImageName string `yaml:"image_name"`
	Version   string `yaml:"version"`

	// Image is the fully-qualified image reference for deploy-only types like
	// gcp-cloud-run-from-image. When set, the deploy step uses this image as-is
	// rather than constructing one from build artifacts. Ignored by types that
	// build their own image (gcp-cloud-run, gcp-artifact-registry-image).
	Image string `yaml:"image"`
}

// DisplayName returns the service name with region suffix for multi-region deployments.
func (s *ServiceInfo) DisplayName() string {
	if s.IsMultiRegion && s.Region != "" {
		return s.Name + " (" + s.Region + ")"
	}
	return s.Name
}

// RecipeKey returns the recipe lookup key for this service.
// This matches the format used to index recipes: "provider-service" or just "provider".
// It uses the Type field (from "type" in config) which contains the full key.
func (s *ServiceInfo) RecipeKey() string {
	if s.Type != "" {
		return s.Type
	}
	return s.Provider
}

func (s *ServiceInfo) Validate() error {
	// Minimal base validation - provider-specific validation is done by recipes
	if s.Name == "" {
		return errors.New("missing required field: name")
	}
	if s.Provider == "" {
		return errors.New("missing required field: provider")
	}
	return nil
}

func NewServiceInfo(config map[string]any, path string) *ServiceInfo {
	rt := configutil.MapFromAny(config["runtime"])
	runtime := RuntimeConfig{}

	if rt["service"] != nil {
		if svc, ok := rt["service"].(string); ok {
			runtime.Service = svc
		}
	}

	// env vars merge: top-level `env_vars:` PLUS any target-specific nested
	// blocks (e.g. `cloud_run.env_vars:`). Both sources contribute to the
	// single `svc.EnvVars` list that downstream code reads.
	//
	// Precedence: top-level wins on key conflicts. This nudges new pilum.yaml
	// files toward top-level (the portable location that applies regardless
	// of deploy target), but doesn't silently drop nested env vars from
	// legacy configs.
	envVars := mergeEnvVarSources(config)

	// secrets conversion
	secrets := configutil.MapFromAny(config["secrets"])
	var secretVars []Secrets
	for k, v := range secrets {
		secretVars = append(secretVars, Secrets{Name: k, Value: v.(string)})
	}

	// Parse build config
	buildConfig := parseBuildConfig(config)

	// Type is the recipe/deployment type (e.g., "gcp-cloud-run")
	serviceType := configutil.GetString(config, "type", "")

	// Template is the Dockerfile template name (e.g., "golang-api.v1.dockerfile")
	template := configutil.GetString(config, "template", "")

	// Provider can be explicit or derived from type
	provider := configutil.GetString(config, "provider", "")
	if provider == "" {
		// Derive provider from type if not explicitly set
		switch serviceType {
		case "gcp-cloud-run", "gcp-cloud-run-job", "gcp-artifact-registry-image", "gcp-cloud-run-from-image", "gcp":
			provider = "gcp"
		case "aws-lambda", "aws-ecs", "aws":
			provider = "aws"
		case "azure-container-apps", "azure":
			provider = "azure"
		case "homebrew":
			provider = "homebrew"
		case "npm":
			provider = "npm"
		case "cloudflare-pages", "cloudflare":
			provider = "cloudflare"
		default:
			// Unknown type, leave provider empty
		}
	}

	sidecars := parseSidecars(config)

	return &ServiceInfo{
		Name:         configutil.GetString(config, "name", ""),
		Description:  configutil.GetString(config, "description", ""),
		Type:         serviceType,
		Template:     template,
		Path:         path,
		Config:       config,
		BuildConfig:  buildConfig,
		Runtime:      runtime,
		Region:       configutil.GetString(config, "region", ""),
		Regions:      configutil.GetStringSlice(config, "regions"),
		Project:      configutil.GetString(config, "project", ""),
		License:      configutil.GetString(config, "license", ""),
		Provider:     provider,
		RegistryName: configutil.GetString(config, "registry_name", ""),
		DependsOn:    configutil.GetStringSlice(config, "depends_on"),
		EnvVars:      envVars,
		Secrets:      secretVars,
		Sidecars:     sidecars,
		ImageName:    configutil.GetString(config, "image_name", ""),
		Version:      configutil.GetString(config, "version", ""),
		Image:        configutil.GetString(config, "image", ""),
	}
}

// mergeEnvVarSources combines top-level `env_vars:` with any nested
// target-specific env_vars blocks into a single deterministic []EnvVars
// list. Top-level entries win on key conflicts.
//
// Background: historically the deploy command builders only read top-level
// EnvVars while the compose generator read both top-level AND nested
// (cloud_run.env_vars). That divergence caused nested env vars to silently
// disappear on real Cloud Run deploys while continuing to work in local
// docker-compose output. Merging at parse time gives every downstream
// reader the same complete view.
func mergeEnvVarSources(config map[string]any) []EnvVars {
	merged := make(map[string]string)

	// Nested first (lowest priority). Add any target-specific env_vars blocks
	// here as new deploy types gain them; today only cloud_run uses this shape.
	for _, nestedKey := range []string{"cloud_run", "container_app", "lambda"} {
		nested := configutil.MapFromAny(config[nestedKey])
		for k, v := range configutil.MapFromAny(nested["env_vars"]) {
			val, ok := v.(string)
			if !ok {
				continue // skip non-string entries; the top-level path is stricter
			}

			merged[k] = val
		}
	}

	// Top-level last so it overwrites any nested value with the same key.
	// Non-string values are skipped rather than fatal — the previous strict
	// path returned nil from NewServiceInfo, which would panic the only
	// production caller (which dereferences svc.Name without a nil check).
	for k, v := range configutil.MapFromAny(config["env_vars"]) {
		val, ok := v.(string)
		if !ok {
			continue
		}

		merged[k] = val
	}

	if len(merged) == 0 {
		return nil
	}

	envVars := make([]EnvVars, 0, len(merged))
	for k, v := range merged {
		envVars = append(envVars, EnvVars{Name: k, Value: v})
	}

	return envVars
}

// parseSidecars extracts the sidecar list from a raw config map. Returns nil
// when no `sidecars:` key is present or it's empty — callers branch on
// len(svc.Sidecars) == 0 to keep the single-container path unchanged.
func parseSidecars(config map[string]any) []Sidecar {
	raw, ok := config["sidecars"].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}

	sidecars := make([]Sidecar, 0, len(raw))

	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}

		sidecars = append(sidecars, parseSidecar(entry))
	}

	return sidecars
}

func parseSidecar(entry map[string]any) Sidecar {
	s := Sidecar{
		Name:      configutil.GetString(entry, "name", ""),
		Image:     configutil.GetString(entry, "image", ""),
		Port:      configutil.GetInt(entry, "port", 0),
		Memory:    configutil.GetString(entry, "memory", ""),
		CPU:       configutil.GetString(entry, "cpu", ""),
		Args:      configutil.GetStringSlice(entry, "args"),
		Command:   configutil.GetStringSlice(entry, "command"),
		DependsOn: configutil.GetStringSlice(entry, "depends_on"),
	}

	for k, v := range configutil.MapFromAny(entry["env_vars"]) {
		val, ok := v.(string)
		if !ok {
			continue
		}

		s.EnvVars = append(s.EnvVars, EnvVars{Name: k, Value: val})
	}

	for k, v := range configutil.MapFromAny(entry["secrets"]) {
		val, ok := v.(string)
		if !ok {
			continue
		}

		s.Secrets = append(s.Secrets, Secrets{Name: k, Value: val})
	}

	probeMap := configutil.MapFromAny(entry["startup_probe"])
	if len(probeMap) > 0 {
		s.StartupProbe = &Probe{
			Path:                configutil.GetString(probeMap, "path", ""),
			Port:                configutil.GetInt(probeMap, "port", 0),
			InitialDelaySeconds: configutil.GetInt(probeMap, "initial_delay_seconds", 0),
			PeriodSeconds:       configutil.GetInt(probeMap, "period_seconds", 0),
			TimeoutSeconds:      configutil.GetInt(probeMap, "timeout_seconds", 0),
			FailureThreshold:    configutil.GetInt(probeMap, "failure_threshold", 0),
		}
	}

	return s
}

// ExpandMultiRegion expands a service with multiple regions into separate ServiceInfo instances.
// If the service has a Regions array, it creates one instance per region.
// If the service only has a single Region, it returns the original service unchanged.
func ExpandMultiRegion(svc ServiceInfo) []ServiceInfo {
	// If no regions array, return as-is
	if len(svc.Regions) == 0 {
		return []ServiceInfo{svc}
	}

	// Expand into multiple services, one per region
	expanded := make([]ServiceInfo, 0, len(svc.Regions))
	for _, region := range svc.Regions {
		instance := svc          // copy
		instance.Region = region // set specific region
		instance.Regions = nil   // clear regions array
		instance.IsMultiRegion = true
		expanded = append(expanded, instance)
	}

	return expanded
}

func parseBuildConfig(config map[string]any) BuildConfig {
	buildMap := configutil.MapFromAny(config["build"])
	if len(buildMap) == 0 {
		return BuildConfig{}
	}

	// Parse resources
	resources := BuildResources{}
	resMap := configutil.MapFromAny(buildMap["resources"])
	if len(resMap) > 0 {
		resources.Memory = configutil.GetInt(resMap, "memory", 0)
		resources.CPU = configutil.GetInt(resMap, "cpu", 0)
	}

	bc := BuildConfig{
		Language:   configutil.GetString(buildMap, "language", ""),
		Version:    configutil.GetString(buildMap, "version", ""),
		Cmd:        configutil.GetString(buildMap, "cmd", ""),
		VersionVar: configutil.GetString(buildMap, "version_var", ""),
		Resources:  resources,
	}

	// Parse build env vars
	buildEnvVars := configutil.MapFromAny(buildMap["env_vars"])
	for k, v := range buildEnvVars {
		if val, ok := v.(string); ok {
			bc.EnvVars = append(bc.EnvVars, EnvVars{Name: k, Value: val})
		}
	}

	// Parse build flags (e.g., ldflags: ["-s", "-w"])
	flagsMap := configutil.MapFromAny(buildMap["flags"])
	for flagName, flagVal := range flagsMap {
		var values []string
		switch v := flagVal.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					values = append(values, s)
				}
			}
		case []string:
			values = v
		case string:
			values = []string{v}
		}
		if len(values) > 0 {
			bc.Flags = append(bc.Flags, BuildFlag{Name: flagName, Values: values})
		}
	}

	return bc
}
