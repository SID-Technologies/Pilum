package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

// tree creates files (with parent dirs) under a new temp dir and returns it.
func tree(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		p := filepath.Join(root, f)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, nil, 0o600))
	}
	return root
}

func TestCanonical(t *testing.T) {
	t.Parallel()

	require.Equal(t, "go", Canonical("golang"))
	require.Equal(t, "go", Canonical(" Go "))
	require.Equal(t, "zig", Canonical("zig"), "unknown languages pass through")
}

func TestResolveGoModuleRoot(t *testing.T) {
	t.Parallel()

	root := tree(t, "go.mod", "services/api/main.go")
	tc, got, ok := Resolve(filepath.Join(root, "services/api"), root, "go")
	require.True(t, ok)
	require.Equal(t, "go", tc.Name())
	require.Equal(t, root, got)
}

func TestResolvePrefersGoWork(t *testing.T) {
	t.Parallel()

	root := tree(t, "go.work", "services/api/go.mod")
	_, got, ok := Resolve(filepath.Join(root, "services/api"), root, "golang")
	require.True(t, ok)
	require.Equal(t, root, got, "a workspace builds its modules as one")
}

func TestResolveStopsAtBound(t *testing.T) {
	t.Parallel()

	root := tree(t, "go.mod", "repo/services/api/main.go")
	_, _, ok := Resolve(filepath.Join(root, "repo/services/api"), filepath.Join(root, "repo"), "go")
	require.False(t, ok, "a go.mod above the repository says nothing about it")
}

func TestResolveWithoutLanguageNeedsManifestInServiceDir(t *testing.T) {
	t.Parallel()

	// A Node app inside a Go monorepo: the go.mod above it isn't its build.
	root := tree(t, "go.mod", "apps/web/package.json")
	_, _, ok := Resolve(filepath.Join(root, "apps/web"), root, "")
	require.False(t, ok)

	root = tree(t, "tools/cli/go.mod")
	_, got, ok := Resolve(filepath.Join(root, "tools/cli"), root, "")
	require.True(t, ok)
	require.Equal(t, filepath.Join(root, "tools/cli"), got)
}

func TestResolveUnknownLanguage(t *testing.T) {
	t.Parallel()

	root := tree(t, "go.mod")
	_, _, ok := Resolve(root, root, "zig")
	require.False(t, ok, "never warm by guesswork")
}

func goService(flags ...serviceinfo.BuildFlag) serviceinfo.ServiceInfo {
	return serviceinfo.ServiceInfo{Name: "svc", BuildConfig: serviceinfo.BuildConfig{Language: "go", Flags: flags}}
}

func TestGoPlanBuildsEveryServicePackage(t *testing.T) {
	t.Parallel()

	root := "/repo"
	plan, ok := golang{}.Plan(Group{
		Root:    root,
		Dirs:    []string{"/repo/services/api", "/repo/services/my app", "/repo/services/api"},
		Service: goService(),
	})
	require.True(t, ok)
	require.Equal(t, "go build ./services/api './services/my app'", plan.Cmd)
	require.Equal(t, root, plan.Dir)
	require.Empty(t, plan.Isolate)
}

func TestGoPlanSinglePackageDiscardsBinary(t *testing.T) {
	t.Parallel()

	// Two regions of one service: one package, which go build would write out.
	plan, ok := golang{}.Plan(Group{Root: "/repo", Dirs: []string{"/repo/api", "/repo/api"}, Service: goService()})
	require.True(t, ok)
	require.Equal(t, "go build -o "+os.DevNull+" ./api", plan.Cmd)
}

func TestGoPlanPassesCompileFlagsOnly(t *testing.T) {
	t.Parallel()

	svc := goService(
		serviceinfo.BuildFlag{Name: "ldflags", Values: []string{"-s", "-w"}},
		serviceinfo.BuildFlag{Name: "tags", Values: []string{"netgo"}},
		serviceinfo.BuildFlag{Name: "trimpath", Values: []string{"true"}},
	)
	plan, ok := golang{}.Plan(Group{Root: "/repo", Dirs: []string{"/repo/a", "/repo/b"}, Service: svc})
	require.True(t, ok)
	require.Equal(t, "go build -tags='netgo' -trimpath='true' ./a ./b", plan.Cmd)
	require.Equal(t, "-tags='netgo' -trimpath='true'", golang{}.Variant(svc))
}

func TestGoPlanRejectsDirOutsideRoot(t *testing.T) {
	t.Parallel()

	_, ok := golang{}.Plan(Group{Root: "/repo/a", Dirs: []string{"/repo/b"}, Service: goService()})
	require.False(t, ok)
}
