// Package templates provides embedded build configuration templates.
package templates

import (
	"embed"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed builds/*.yaml
var buildsFS embed.FS

// BuildConfig represents a language-specific build configuration.
type BuildConfig struct {
	Language string            `yaml:"language"`
	Version  string            `yaml:"version"`
	Cmd      string            `yaml:"cmd"`
	EnvVars  map[string]string `yaml:"env_vars"`
	Flags    map[string]any    `yaml:"flags"`
}

// GetAvailableLanguages returns a list of supported build languages.
func GetAvailableLanguages() []string {
	entries, err := buildsFS.ReadDir("builds")
	if err != nil {
		return nil
	}

	var languages []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") {
			lang := strings.TrimSuffix(name, ".yaml")
			languages = append(languages, lang)
		}
	}

	sort.Strings(languages)
	return languages
}

// GetBuildConfig loads the build configuration for a specific language.
func GetBuildConfig(language string) (*BuildConfig, error) {
	filename := filepath.Join("builds", language+".yaml")
	data, err := buildsFS.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config BuildConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetBuildConfigYAML returns the raw YAML content for a language's build config.
func GetBuildConfigYAML(language string) (string, error) {
	filename := filepath.Join("builds", language+".yaml")
	data, err := buildsFS.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
