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

// Probe is an HTTP GET readiness/startup probe.
type Probe struct {
	Path                string `yaml:"path"`
	Port                int    `yaml:"port"`
	InitialDelaySeconds int    `yaml:"initial_delay_seconds"`
	PeriodSeconds       int    `yaml:"period_seconds"`
	TimeoutSeconds      int    `yaml:"timeout_seconds"`
	FailureThreshold    int    `yaml:"failure_threshold"`
}

// Sidecar is an auxiliary container co-deployed with the ingress container.
// Cloud Run sidecars share the network namespace with the ingress container.
type Sidecar struct {
	Name         string    `yaml:"name"`
	Image        string    `yaml:"image"`
	Port         int       `yaml:"port"`
	Memory       string    `yaml:"memory"`
	CPU          string    `yaml:"cpu"`
	EnvVars      []EnvVars `yaml:"env_vars"`
	Secrets      []Secrets `yaml:"secrets"`
	Args         []string  `yaml:"args"`
	Command      []string  `yaml:"command"`
	DependsOn    []string  `yaml:"depends_on"`
	StartupProbe *Probe    `yaml:"startup_probe"`
}

type ServiceInfo struct {
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description"`
	Type          string         `yaml:"-"`
	Template      string         `yaml:"template"`
	Path          string         `yaml:"-"`
	Config        map[string]any `yaml:"-"`
	BuildConfig   BuildConfig    `yaml:"build"`
	Runtime       RuntimeConfig  `yaml:"runtime"`
	EnvVars       []EnvVars      `yaml:"env_vars"`
	Secrets       []Secrets      `yaml:"secrets"`
	Region        string         `yaml:"region"`
	Regions       []string       `yaml:"regions"`
	IsMultiRegion bool           `yaml:"-"`
	Project       string         `yaml:"project"`
	License       string         `yaml:"license"`
	Provider      string         `yaml:"provider"`
	RegistryName  string         `yaml:"registry_name"`
	DependsOn     []string       `yaml:"depends_on"`
	Sidecars      []Sidecar      `yaml:"sidecars"`

	// ImageName + Version are used by gcp-artifact-registry-image to override
	// the defaults derived from Name. Ignored by deploy-only types.
	ImageName string `yaml:"image_name"`
	Version   string `yaml:"version"`

	// Image is the pre-built image ref for gcp-cloud-run-from-image deploys.
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

	envVars := mergeEnvVarSources(config)

	secrets := configutil.MapFromAny(config["secrets"])
	var secretVars []Secrets
	for k, v := range secrets {
		secretVars = append(secretVars, Secrets{Name: k, Value: v.(string)})
	}

	buildConfig := parseBuildConfig(config)

	serviceType := configutil.GetString(config, "type", "")
	template := configutil.GetString(config, "template", "")

	provider := configutil.GetString(config, "provider", "")
	if provider == "" {
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

// mergeEnvVarSources merges top-level env_vars with target-specific nested
// blocks (cloud_run.env_vars, etc.). Top-level wins on key conflicts.
func mergeEnvVarSources(config map[string]any) []EnvVars {
	merged := make(map[string]string)

	for _, nestedKey := range []string{"cloud_run", "container_app", "lambda"} {
		nested := configutil.MapFromAny(config[nestedKey])
		for k, v := range configutil.MapFromAny(nested["env_vars"]) {
			val, ok := v.(string)
			if !ok {
				continue
			}

			merged[k] = val
		}
	}

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
