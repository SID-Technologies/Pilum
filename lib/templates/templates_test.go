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

func TestCanonicalAliases(t *testing.T) {
	t.Parallel()

	require.Equal(t, "node", Canonical("nodejs"))
	require.Equal(t, "node", Canonical(" TypeScript "))
	require.Equal(t, "go", Canonical("golang"))
	require.Equal(t, "bun", Canonical("bun"))
	require.Equal(t, "zig", Canonical("Zig"), "unknown languages pass through")
}

func TestAliasesResolveTemplates(t *testing.T) {
	t.Parallel()

	require.Equal(t, 1536, MemoryForLanguage("nodejs"), "the examples' spelling finds the node template")
	require.Contains(t, GetAvailableLanguages(), "bun")

	cfg, err := GetBuildConfig("bun")
	require.NoError(t, err)
	require.Equal(t, "bun run build", cfg.Cmd)
}

func TestTier2Templates(t *testing.T) {
	t.Parallel()

	for _, lang := range []string{"php", "ruby", "c", "cpp", "c++", "java", "kotlin", "dotnet", "csharp"} {
		cfg, err := GetBuildConfig(lang)
		require.NoError(t, err, lang)
		require.NotEmpty(t, cfg.Cmd, lang)
		require.Positive(t, cfg.Resources.Memory, lang)
	}
}
