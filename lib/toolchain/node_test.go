package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Plans shouldn't depend on which package managers this machine has.
	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	pnpmStorePath = func(string) (string, error) { return "/home/ci/.local/share/pnpm/store/v10", nil }
	os.Exit(m.Run())
}

func TestCanonicalNodeAliases(t *testing.T) {
	t.Parallel()

	for _, lang := range []string{"nodejs", "JavaScript", "js", "typescript", "ts"} {
		require.Equal(t, "node", Canonical(lang), lang)
	}
	require.Equal(t, "bun", Canonical("bun"), "bun has its own template")
}

func TestNodeRootIsTheWorkspaceLockfile(t *testing.T) {
	t.Parallel()

	root := tree(t, "pnpm-lock.yaml", "package.json", "apps/web/package.json")
	tc, got, ok := Resolve(filepath.Join(root, "apps/web"), root, "typescript")
	require.True(t, ok)
	require.Equal(t, "node", tc.Name())
	require.Equal(t, root, got)

	_, got, ok = Resolve(filepath.Join(root, "apps/web"), root, "bun")
	require.True(t, ok, "bun services use the node toolchain")
	require.Equal(t, root, got)
}

func TestNodeWithoutLanguageNeedsPackageJSON(t *testing.T) {
	t.Parallel()

	root := tree(t, "package-lock.json", "package.json", "services/api/main.go")
	_, _, ok := Resolve(filepath.Join(root, "services/api"), root, "")
	require.False(t, ok)
}

func TestNodePlanPerPackageManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []string
		tool  string
		cmd   string
	}{
		{"pnpm", []string{"pnpm-lock.yaml"}, "pnpm", "pnpm fetch --store-dir /home/ci/.local/share/pnpm/store/v10"},
		{"bun", []string{"bun.lock"}, "bun", "bun install --frozen-lockfile --ignore-scripts"},
		{"bun binary lockfile", []string{"bun.lockb"}, "bun", "bun install --frozen-lockfile --ignore-scripts"},
		{"yarn 1", []string{"yarn.lock"}, "yarn", "yarn install --frozen-lockfile --ignore-scripts --non-interactive"},
		{"yarn berry", []string{"yarn.lock", ".yarnrc.yml"}, "yarn", "yarn install --immutable --mode=skip-build"},
		{"npm", []string{"package-lock.json"}, "npm", "npm ci --ignore-scripts --no-audit --no-fund"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := tree(t, append(tt.files, "package.json")...)
			plan, ok := node{}.Plan(Group{Root: root})
			require.True(t, ok)
			require.Equal(t, tt.tool, plan.Tool)
			require.Equal(t, tt.cmd, plan.Cmd)
			require.Equal(t, root, plan.Dir)
		})
	}
}

func TestYarnBerryFromLockfileMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "yarn.lock"), []byte("__metadata:\n  version: 8\n"), 0o600))
	require.True(t, isYarnBerry(root))
}

func TestNodePlanIsolatesManifestsAndLockfiles(t *testing.T) {
	t.Parallel()

	root := tree(t,
		"pnpm-lock.yaml", "pnpm-workspace.yaml", ".npmrc", "package.json",
		"apps/web/package.json", "apps/web/src/index.ts",
		"packages/ui/package.json",
		"patches/react@19.patch",
		"node_modules/react/package.json",
		"apps/web/dist/package.json",
	)
	plan, ok := node{}.Plan(Group{Root: root})
	require.True(t, ok)

	require.Subset(t, plan.Isolate, []string{
		"pnpm-lock.yaml", "pnpm-workspace.yaml", ".npmrc", "package.json",
		"apps/web/package.json", "packages/ui/package.json", "patches/react@19.patch",
	})
	require.NotContains(t, plan.Isolate, "apps/web/src/index.ts", "sources aren't needed to install")
	require.NotContains(t, plan.Isolate, "node_modules/react/package.json")
	require.NotContains(t, plan.Isolate, "apps/web/dist/package.json")
}

func TestNodePlanSkipsMissingTool(t *testing.T) {
	// Not parallel: swaps lookPath.
	saved := lookPath
	t.Cleanup(func() { lookPath = saved })
	lookPath = func(string) (string, error) { return "", os.ErrNotExist }

	root := tree(t, "bun.lock", "package.json")
	_, ok := node{}.Plan(Group{Root: root})
	require.False(t, ok)
}
