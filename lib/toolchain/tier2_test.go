package toolchain

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseComposerVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		out          string
		major, minor int
		ok           bool
	}{
		{"Composer version 2.8.4 2024-12-11 11:57:47", 2, 8, true},
		{"Composer 2.5.0 2022-12-20", 2, 5, true},
		{"Composer version 1.10.27", 1, 10, true},
		{"command not found", 0, 0, false},
	}
	for _, tt := range tests {
		major, minor, ok := parseComposerVersion(tt.out)
		require.Equal(t, tt.ok, ok, tt.out)
		require.Equal(t, tt.major, major, tt.out)
		require.Equal(t, tt.minor, minor, tt.out)
	}
}

func TestComposerPlan(t *testing.T) {
	// Not parallel: swaps composerVersion.
	saved := composerVersion
	t.Cleanup(func() { composerVersion = saved })

	root := tree(t, "composer.json", "composer.lock", "services/api/composer.json")
	tc, got, ok := Resolve(filepath.Join(root, "services/api"), root, "php")
	require.True(t, ok)
	require.Equal(t, root, got)

	composerVersion = func(string) (int, int, bool) { return 2, 8, true }
	plan, ok := tc.Plan(Group{Root: got})
	require.True(t, ok)
	require.Equal(t, "composer install --download-only --no-scripts --no-plugins --no-interaction --quiet", plan.Cmd)
	require.Empty(t, plan.Isolate, "--download-only installs nothing")

	composerVersion = func(string) (int, int, bool) { return 2, 4, true }
	_, ok = tc.Plan(Group{Root: got})
	require.False(t, ok, "--download-only arrived in Composer 2.5")
}

func TestRubyRecognizedButNotWarmed(t *testing.T) {
	t.Parallel()

	root := tree(t, "Gemfile", "Gemfile.lock")
	tc, got, ok := Resolve(root, root, "")
	require.True(t, ok)
	require.Equal(t, "bundler", tc.Name())

	_, ok = tc.Plan(Group{Root: got})
	require.False(t, ok, "Bundler has no download-only install")
}

func TestCAndCppAreNotAutoWarmed(t *testing.T) {
	t.Parallel()

	root := tree(t, "CMakeLists.txt", "vcpkg.json")
	for _, lang := range []string{"c", "cpp", "c++"} {
		_, _, ok := Resolve(root, root, lang)
		require.False(t, ok, lang)
	}
}
