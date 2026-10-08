package examples

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDockerfileFor(t *testing.T) {
	t.Parallel()

	for _, lang := range []string{"go", "node", "nodejs", "typescript", "python", "rust", "bun", "java", "dotnet", "csharp"} {
		data, ok := DockerfileFor(lang)
		require.True(t, ok, lang)
		require.Contains(t, string(data), "FROM", lang)
	}

	_, ok := DockerfileFor("zig")
	require.False(t, ok)
}
