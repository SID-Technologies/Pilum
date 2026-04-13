package compose

import (
	"sort"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// Service represents a single service in a docker-compose file.
type Service struct {
	Image         string         `yaml:"image"`
	ContainerName string         `yaml:"container_name"`
	Restart       string         `yaml:"restart"`
	Environment   []string       `yaml:"environment,omitempty"`
	Ports         []string       `yaml:"ports,omitempty"`
	Networks      []string       `yaml:"networks,omitempty"`
	DependsOn     map[string]any `yaml:"depends_on,omitempty"`
	Healthcheck   *Healthcheck   `yaml:"healthcheck,omitempty"`
	MemLimit      string         `yaml:"mem_limit,omitempty"`
	CPUs          string         `yaml:"cpus,omitempty"`
	Volumes       []string       `yaml:"volumes,omitempty"`
}

// Healthcheck represents a docker-compose healthcheck configuration.
type Healthcheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	Retries     int      `yaml:"retries"`
	StartPeriod string   `yaml:"start_period"`
}

// File represents a complete docker-compose.yml structure.
type File struct {
	Services map[string]*Service `yaml:"services"`
	Networks map[string]any      `yaml:"networks"`
	Volumes  map[string]any      `yaml:"volumes,omitempty"`
}

// Result is the JSON output structure for --json mode.
type Result struct {
	File     string         `json:"file"`
	Services []string       `json:"services"`
	Ports    map[string]int `json:"ports"`
}

// Options holds all configuration for compose file generation.
type Options struct {
	Registry     string
	Tag          string
	Dependencies []Dependency
	DisableDeps  bool // --no-deps flag
}

// secretBinding maps a service secret to the dependency that provides it.
type secretBinding struct {
	depName    string // compose service name of the dependency
	connString string // rendered connection string to inject
}

// Generate builds a complete compose File from discovered services and options.
func Generate(services []serviceinfo.ServiceInfo, opts Options) *File {
	compose := &File{
		Services: make(map[string]*Service),
		Networks: map[string]any{
			"app-network": map[string]string{"driver": "bridge"},
		},
		Volumes: make(map[string]any),
	}

	// Build secret→dependency binding map
	secretMap := make(map[string]secretBinding)

	if !opts.DisableDeps {
		for _, dep := range opts.Dependencies {
			svc, volumeNames := BuildDependencyService(dep)
			compose.Services[dep.Name] = svc

			for _, vn := range volumeNames {
				compose.Volumes[vn] = map[string]any{}
			}

			// Map each secret to this dependency
			connStr := RenderConnectionString(dep)
			for _, secretName := range dep.Secrets {
				secretMap[secretName] = secretBinding{
					depName:    dep.Name,
					connString: connStr,
				}
			}
		}
	}

	// Drop empty volumes map so it doesn't render in YAML
	if len(compose.Volumes) == 0 {
		compose.Volumes = nil
	}

	// Sort services for deterministic port assignment
	sorted := make([]serviceinfo.ServiceInfo, len(services))
	copy(sorted, services)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	// Detect migration service
	var migrationName string
	for _, svc := range sorted {
		if strings.Contains(strings.ToLower(svc.Name), "migration") {
			migrationName = svc.Name
			break
		}
	}

	// Build app services
	for i, svc := range sorted {
		cs := BuildServiceEntry(svc, i, opts)
		isMigration := strings.Contains(strings.ToLower(svc.Name), "migration")

		if isMigration {
			cs.Restart = "no"
		}

		// Wire dependencies via secret matching
		wireSecrets(cs, svc, secretMap)

		// Migration ordering
		if !isMigration && migrationName != "" {
			if cs.DependsOn == nil {
				cs.DependsOn = make(map[string]any)
			}
			cs.DependsOn[migrationName] = map[string]string{
				"condition": "service_completed_successfully",
			}
		}

		compose.Services[svc.Name] = cs
	}

	return compose
}

// wireSecrets injects dependency connection strings into environment variables
// and adds depends_on entries for matching secrets.
func wireSecrets(cs *Service, svc serviceinfo.ServiceInfo, secretMap map[string]secretBinding) {
	if len(secretMap) == 0 {
		return
	}

	depNames := make(map[string]bool)

	for _, secret := range svc.Secrets {
		binding, ok := secretMap[secret.Name]
		if !ok {
			continue
		}

		depNames[binding.depName] = true

		// Replace the placeholder env var with the rendered connection string
		if binding.connString != "" {
			replacement := secret.Name + "=" + binding.connString
			for i, env := range cs.Environment {
				if strings.HasPrefix(env, secret.Name+"=") {
					cs.Environment[i] = replacement
					break
				}
			}
		}
	}

	// Add depends_on for each matched dependency
	if len(depNames) > 0 {
		if cs.DependsOn == nil {
			cs.DependsOn = make(map[string]any)
		}
		for name := range depNames {
			cs.DependsOn[name] = map[string]string{
				"condition": "service_healthy",
			}
		}
	}
}
