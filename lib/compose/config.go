package compose

import (
	"fmt"

	"github.com/sid-technologies/pilum/lib/configutil"
	"github.com/sid-technologies/pilum/lib/errors"

	"github.com/spf13/viper"
)

// Dependency represents an infrastructure dependency declared in .pilum.yml.
type Dependency struct {
	Type        string            // "postgres", "mysql", "redis", "mongo", "mariadb", "custom"
	Name        string            // Compose service name (defaults to Type)
	Image       string            // Override image or required for "custom"
	Port        int               // Host port override
	User        string            // DB user
	Password    string            // DB password
	Database    string            // DB name
	Healthcheck string            // Health check command (required for "custom")
	Environment map[string]string // Extra environment variables
	Secrets     []string          // Service secret names this dependency provides
}

// LoadDependencies reads compose.dependencies from .pilum.yml via Viper.
// Returns nil if the compose section doesn't exist (not an error).
func LoadDependencies() ([]Dependency, error) {
	raw := viper.Get("compose")
	if raw == nil {
		return nil, nil
	}

	composeMap, ok := raw.(map[string]any)
	if !ok {
		return nil, nil
	}

	depsRaw, ok := composeMap["dependencies"]
	if !ok {
		return nil, nil
	}

	items, ok := depsRaw.([]any)
	if !ok {
		return nil, nil
	}

	var deps []Dependency
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		dep, err := parseDependency(m)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("compose.dependencies[%d]", i))
		}
		deps = append(deps, dep)
	}

	return deps, nil
}

func parseDependency(m map[string]any) (Dependency, error) {
	typeName := configutil.GetString(m, "type", "")
	if typeName == "" {
		return Dependency{}, errors.New("missing required field: type")
	}

	dep := Dependency{
		Type:     typeName,
		Name:     configutil.GetString(m, "name", typeName),
		Image:    configutil.GetString(m, "image", ""),
		Port:     configutil.GetInt(m, "port", 0),
		User:     configutil.GetString(m, "user", ""),
		Password: configutil.GetString(m, "password", ""),
		Database: configutil.GetString(m, "database", ""),
		Secrets:  configutil.GetStringSlice(m, "secrets"),
	}

	// Custom healthcheck command
	dep.Healthcheck = configutil.GetString(m, "healthcheck", "")

	// Extra environment variables
	if envRaw, ok := m["environment"]; ok {
		if envMap, ok := envRaw.(map[string]any); ok {
			dep.Environment = make(map[string]string, len(envMap))
			for k, v := range envMap {
				if s, ok := v.(string); ok {
					dep.Environment[k] = s
				}
			}
		}
	}

	// Validate custom type
	if typeName == "custom" && dep.Image == "" {
		return Dependency{}, errors.New("custom dependency requires an image")
	}

	// Validate known type
	if typeName != "custom" {
		if _, ok := GetPreset(typeName); !ok {
			return Dependency{}, errors.New("unknown dependency type %q (supported: %v)", typeName, SupportedTypes())
		}
	}

	return dep, nil
}
