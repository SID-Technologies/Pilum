package recepie

import (
	"reflect"

	"github.com/sid-technologies/pilum/lib/configutil"
	"github.com/sid-technologies/pilum/lib/errors"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// Field defines a configuration field for a recipe.
// Used for both required and optional fields.
type Field struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`    // string, int, bool, list
	Default     string `yaml:"default"` // default value if not provided
}

// RequiredField is an alias for Field for backwards compatibility.
type RequiredField = Field

// Recipe defines a deployment workflow.
type Recipe struct {
	Name           string       `yaml:"name"`
	Description    string       `yaml:"description"`
	Provider       string       `yaml:"provider"`
	Service        string       `yaml:"service"`
	RequiredFields []Field      `yaml:"required_fields"`
	OptionalFields []Field      `yaml:"optional_fields"`
	Steps          []RecipeStep `yaml:"steps"`
}

// RecipeStep defines a single step in a recipe.
type RecipeStep struct {
	Name          string            `yaml:"name"`
	Command       any               `yaml:"command,omitempty"` // string or []string
	ExecutionMode string            `yaml:"execution_mode"`
	EnvVars       map[string]string `yaml:"env_vars,omitempty"`
	BuildFlags    map[string]any    `yaml:"build_flags,omitempty"`
	Timeout       int               `yaml:"timeout,omitempty"`
	Debug         bool              `yaml:"debug,omitempty"`
	Retries       int               `yaml:"retries,omitempty"`
	Tags          []string          `yaml:"tags,omitempty"` // Tags for filtering (e.g., "deploy", "build")
}

// ValidateService checks if a service has all required fields for this recipe.
func (r *Recipe) ValidateService(svc *serviceinfo.ServiceInfo) error {
	for _, field := range r.RequiredFields {
		value := getServiceField(svc, field.Name)
		if value == "" && field.Default == "" {
			return errors.New("recipe '%s' requires field '%s': %s",
				r.Name, field.Name, field.Description)
		}
	}
	return nil
}

// GetRequiredFields returns the list of required fields with descriptions.
func (r *Recipe) GetRequiredFields() []Field {
	return r.RequiredFields
}

// GetOptionalFields returns the list of optional fields with descriptions.
func (r *Recipe) GetOptionalFields() []Field {
	return r.OptionalFields
}

// getServiceField extracts a field value from ServiceInfo by name.
// Supports nested field names like "homebrew.tap_url".
func getServiceField(svc *serviceinfo.ServiceInfo, fieldName string) string {
	// Map common field names to ServiceInfo struct fields
	fieldMap := map[string]func(*serviceinfo.ServiceInfo) string{
		"name":          func(s *serviceinfo.ServiceInfo) string { return s.Name },
		"description":   func(s *serviceinfo.ServiceInfo) string { return s.Description },
		"project":       func(s *serviceinfo.ServiceInfo) string { return s.Project },
		"license":       func(s *serviceinfo.ServiceInfo) string { return s.License },
		"region":        func(s *serviceinfo.ServiceInfo) string { return s.Region },
		"provider":      func(s *serviceinfo.ServiceInfo) string { return s.Provider },
		"template":      func(s *serviceinfo.ServiceInfo) string { return s.Template },
		"registry_name": func(s *serviceinfo.ServiceInfo) string { return s.RegistryName },
		"homebrew.project_url": func(s *serviceinfo.ServiceInfo) string {
			return configutil.GetNestedString(s.Config, "homebrew", "project_url")
		},
		"homebrew.tap_url": func(s *serviceinfo.ServiceInfo) string {
			return configutil.GetNestedString(s.Config, "homebrew", "tap_url")
		},
		"homebrew.token_env": func(s *serviceinfo.ServiceInfo) string {
			return configutil.GetNestedString(s.Config, "homebrew", "token_env")
		},
	}

	if getter, exists := fieldMap[fieldName]; exists {
		return getter(svc)
	}

	// Try to get from the raw config map
	if svc.Config != nil {
		if val, exists := svc.Config[fieldName]; exists {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}

	// Try reflection as fallback for any other fields
	v := reflect.ValueOf(svc).Elem()
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == fieldName || field.Name == fieldName {
			fieldVal := v.Field(i)
			if fieldVal.Kind() == reflect.String {
				return fieldVal.String()
			}
		}
	}

	return ""
}
