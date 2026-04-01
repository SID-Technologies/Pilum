package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryForLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		language string
		expected int
	}{
		{"go", 1024},
		{"rust", 2048},
		{"node", 1536},
		{"python", 512},
		{"unknown", DefaultBuildMemoryMB},
		{"", DefaultBuildMemoryMB},
	}

	for _, tt := range tests {
		require.Equal(t, tt.expected, MemoryForLanguage(tt.language),
			"unexpected memory for language %q", tt.language)
	}
}

func TestBuildConfigHasResources(t *testing.T) {
	t.Parallel()

	// Every supported language template should have resources.memory set
	for _, lang := range GetAvailableLanguages() {
		config, err := GetBuildConfig(lang)
		require.NoError(t, err, "failed to load config for %s", lang)
		require.Greater(t, config.Resources.Memory, 0,
			"language %s should declare resources.memory in its build template", lang)
	}
}
