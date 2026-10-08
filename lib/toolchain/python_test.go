package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPythonRootAndTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []string
		tool  string
		cmd   string
	}{
		{"uv", []string{"uv.lock", "pyproject.toml", "services/api/pyproject.toml"}, "uv", "uv sync --frozen --no-install-workspace --quiet"},
		{"pip", []string{"requirements.txt", "services/api/app.py"}, "pip", "pip download --quiet -r requirements.txt -d .pilum-download"},
		{"uv wins over requirements", []string{"uv.lock", "requirements.txt", "pyproject.toml"}, "uv", "uv sync --frozen --no-install-workspace --quiet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := tree(t, append(tt.files, "services/api/main.py")...)
			tc, got, ok := Resolve(filepath.Join(root, "services/api"), root, "python")
			require.True(t, ok)
			require.Equal(t, root, got)

			plan, ok := tc.Plan(Group{Root: got})
			require.True(t, ok)
			require.Equal(t, tt.tool, plan.Tool)
			require.Equal(t, tt.cmd, plan.Cmd)
		})
	}
}

func TestPythonPoetryDoesNotWarm(t *testing.T) {
	t.Parallel()

	root := tree(t, "poetry.lock", "pyproject.toml")
	_, ok := python{}.Plan(Group{Root: root})
	require.False(t, ok, "Poetry has no download-only install")
}

func TestUVIsolatesEveryPyproject(t *testing.T) {
	t.Parallel()

	root := tree(t, "uv.lock", "pyproject.toml", ".python-version",
		"packages/core/pyproject.toml", "packages/core/src/core.py",
		".venv/lib/pyproject.toml")
	plan, ok := python{}.Plan(Group{Root: root})
	require.True(t, ok)
	require.Subset(t, plan.Isolate, []string{"uv.lock", ".python-version", "pyproject.toml", "packages/core/pyproject.toml"})
	require.NotContains(t, plan.Isolate, "packages/core/src/core.py")
	require.NotContains(t, plan.Isolate, ".venv/lib/pyproject.toml")
}

func TestPipIsolatesRequirementsAndConstraints(t *testing.T) {
	t.Parallel()

	root := tree(t, "requirements.txt", "requirements-dev.txt", "constraints.txt", "notes.txt", "venv/requirements.txt")
	plan, ok := python{}.Plan(Group{Root: root})
	require.True(t, ok)
	require.ElementsMatch(t, []string{"requirements.txt", "requirements-dev.txt", "constraints.txt"}, plan.Isolate)
}

func TestPythonWithoutLanguage(t *testing.T) {
	t.Parallel()

	root := tree(t, "uv.lock", "pyproject.toml", "services/api/pyproject.toml", "services/web/package.json")
	tc, _, ok := Resolve(filepath.Join(root, "services/api"), root, "")
	require.True(t, ok)
	require.Equal(t, "python", tc.Name())

	_, _, ok = Resolve(filepath.Join(root, "services/web"), root, "python")
	require.True(t, ok, "a declared language finds the workspace above")
}

func TestPipCommandFallsBackToModule(t *testing.T) {
	// Not parallel: swaps lookPath.
	saved := lookPath
	t.Cleanup(func() { lookPath = saved })
	lookPath = func(file string) (string, error) {
		if file == "python3" {
			return "/usr/bin/python3", nil
		}
		return "", os.ErrNotExist
	}

	pip, ok := pipCommand()
	require.True(t, ok)
	require.Equal(t, "python3 -m pip", pip)
}
