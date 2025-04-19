package serviceinfo

import (
	"github.com/sid-technologies/centurion/lib/errors"
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

type BuildConfig struct {
	Language string      `yaml:"language"`
	Version  string      `yaml:"version"`
	Cmd      string      `yaml:"cmd"`
	EnvVars  []EnvVars   `yaml:"env_vars"`
	Flags    []BuildFlag `yaml:"flags"`
}

type RuntimeConfig struct {
	Service string `yaml:"service"`
}

type ServiceInfo struct {
	Name         string         `yaml:"name"`
	Template     string         `yaml:"template"`
	Path         string         `yaml:"-"`
	Config       map[string]any `yaml:"-"`
	BuildConfig  BuildConfig    `yaml:"build"`
	Runtime      RuntimeConfig  `yaml:"runtime"`
	EnvVars      []EnvVars      `yaml:"env_vars"`
	Secrets      []Secrets      `yaml:"secrets"`
	Region       string         `yaml:"region"`
	Project      string         `yaml:"project"`
	Provider     string         `yaml:"provider"`
	RegistryName string         `yaml:"registry_name"`
}

func (s *ServiceInfo) Validate() error {
	if s.Name == "" {
		return errors.New("missing required field: name")
	}
	if s.Template == "" {
		return errors.New("missing required field: template")
	}
	if s.Region == "" {
		return errors.New("missing required field: region")
	}
	if s.Project == "" {
		return errors.New("missing required field: project")
	}
	if s.Provider == "" {
		return errors.New("missing required field: provider")
	}

	// add more rules as needed
	return nil
}

func NewServiceInfo(config map[string]any, path string) *ServiceInfo {
	rt := mapFromAny(config["runtime"])
	// service  := mapFromAny(config["service"])
	runtime := RuntimeConfig{}

	if rt["service"] != nil {
		runtime.Service = rt["service"].(string)
	}

	// env vars conversions
	evs := mapFromAny(config["env_vars"])
	var envVars []EnvVars
	for k, v := range evs {
		key := k
		val := v.(string)
		envVars = append(envVars, EnvVars{Name: key, Value: val})
	}

	// secrets conversion
	secrets := mapFromAny(config["secrets"])
	var secretVars []Secrets
	for k, v := range secrets {
		secretVars = append(secretVars, Secrets{Name: k, Value: v.(string)})
	}

	return &ServiceInfo{
		Name:     getString(config, "name", ""),
		Template: getString(config, "template", ""),
		Path:     path,
		Config:   config,
		Runtime:  runtime,
		Region:   getString(config, "region", ""),
		Project:  getString(config, "project", ""),
		Provider: getString(config, "provider", ""),
		EnvVars:  envVars,
		Secrets:  secretVars,
	}
}
