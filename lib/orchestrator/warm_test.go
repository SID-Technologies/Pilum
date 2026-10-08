package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"

	"github.com/stretchr/testify/require"
)

// warmRecipe is a build step followed by a deploy step.
func warmRecipe(buildCmd string) []recipe.Info {
	return []recipe.Info{{
		Provider: "test",
		Recipe: recipe.Recipe{
			Name:     "test-recipe",
			Provider: "test",
			Steps: []recipe.Step{
				{Name: "build binary", Command: buildCmd, ExecutionMode: "root", Tags: []string{"build"}, Timeout: 30},
				{Name: "deploy", Command: "true", ExecutionMode: "root", Tags: []string{"deploy"}, Timeout: 30},
			},
		},
	}}
}

func warmService(name, language, warm string, env ...serviceinfo.EnvVars) serviceinfo.ServiceInfo {
	return serviceinfo.ServiceInfo{
		Name:        name,
		Provider:    "test",
		BuildConfig: serviceinfo.BuildConfig{Language: language, Warm: warm, EnvVars: env},
	}
}

func TestWarmGroupsDedupesPerCommandDirAndEnv(t *testing.T) {
	t.Parallel()

	linux := serviceinfo.EnvVars{Name: "GOOS", Value: "linux"}
	var services []serviceinfo.ServiceInfo
	for _, name := range []string{"api", "auth", "billing", "search"} {
		services = append(services, warmService(name, "go", "go build ./...", linux))
	}
	for _, name := range []string{"web", "admin", "docs"} {
		services = append(services, warmService(name, "node", "pnpm install --frozen-lockfile"))
	}
	// Same command, different target: its cache entries differ, so it warms separately.
	services = append(services, warmService("cli", "go", "go build ./...", serviceinfo.EnvVars{Name: "GOOS", Value: "darwin"}))
	// Nothing to warm.
	services = append(services, warmService("static", "", ""))

	p := NewPipeline(services, warmRecipe("true"), types.PipelineOptions{})
	groups := p.warmGroups(p.findMaxSteps())

	require.Len(t, groups, 3)
	require.Equal(t, "go build ./...", groups[0].cmd)
	require.ElementsMatch(t, []string{"api", "auth", "billing", "search"}, groups[0].services)
	require.Equal(t, "go (4 services)", groups[0].label())
	require.Equal(t, map[string]string{"GOOS": "linux"}, groups[0].env)
	require.Equal(t, 30, groups[0].timeout, "the build step's timeout")

	require.Equal(t, "pnpm install --frozen-lockfile", groups[1].cmd)
	require.Len(t, groups[1].services, 3)

	require.Equal(t, []string{"cli"}, groups[2].services)
	require.Equal(t, "go (cli)", groups[2].label())
}

func TestWarmGroupsSkippedWithoutABuildStep(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{warmService("api", "go", "go build ./...")}
	p := NewPipeline(services, warmRecipe("true"), types.PipelineOptions{OnlyTags: []string{"deploy"}})
	require.Empty(t, p.warmGroups(p.findMaxSteps()), "a deploy-only run builds nothing")
}

func TestWarmGroupsSeparateDirs(t *testing.T) {
	t.Parallel()

	a := warmService("a", "go", "go build ./...")
	a.BuildConfig.WarmDir = "services/a"
	b := warmService("b", "go", "go build ./...")
	b.BuildConfig.WarmDir = "services/b"

	p := NewPipeline([]serviceinfo.ServiceInfo{a, b}, warmRecipe("true"), types.PipelineOptions{})
	groups := p.warmGroups(p.findMaxSteps())
	require.Len(t, groups, 2)
	require.Equal(t, "go in services/a (a)", groups[0].label())
	require.Equal(t, "(cd services/a && go build ./...)", warmCommand(groups[0]))
}

// readLog returns the lines appended to path.
func readLog(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return strings.Fields(string(data))
}

func TestPipelineWarmsEachLanguageOnceBeforeBuilds(t *testing.T) {
	t.Parallel()

	log := filepath.Join(t.TempDir(), "log")
	var services []serviceinfo.ServiceInfo
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		services = append(services, warmService(name, "go", "echo warm-go >> "+log))
	}
	for _, name := range []string{"x", "y"} {
		services = append(services, warmService(name, "node", "echo warm-node >> "+log))
	}

	p := NewPipeline(services, warmRecipe("echo build >> "+log), types.PipelineOptions{Timeout: 10})
	require.NoError(t, p.Run())

	lines := readLog(t, log)
	require.Len(t, lines, 2+len(services))
	require.ElementsMatch(t, []string{"warm-go", "warm-node"}, lines[:2], "both languages warm before any build")
	for _, line := range lines[2:] {
		require.Equal(t, "build", line)
	}
}

func TestPipelineWarmFailureDoesNotFailTheRun(t *testing.T) {
	t.Parallel()

	log := filepath.Join(t.TempDir(), "log")
	services := []serviceinfo.ServiceInfo{warmService("a", "go", "exit 1")}

	p := NewPipeline(services, warmRecipe("echo build >> "+log), types.PipelineOptions{Timeout: 10})
	require.NoError(t, p.Run())
	require.Equal(t, []string{"build"}, readLog(t, log))
}

func TestPipelineNoWarm(t *testing.T) {
	t.Parallel()

	log := filepath.Join(t.TempDir(), "log")
	services := []serviceinfo.ServiceInfo{warmService("a", "go", "echo warm >> "+log)}

	p := NewPipeline(services, warmRecipe("echo build >> "+log), types.PipelineOptions{Timeout: 10, NoWarm: true})
	require.NoError(t, p.Run())
	require.Equal(t, []string{"build"}, readLog(t, log))
}

func TestPipelineWarmRunsInWarmDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	svc := warmService("a", "go", "pwd > "+log)
	svc.BuildConfig.WarmDir = dir

	p := NewPipeline([]serviceinfo.ServiceInfo{svc}, warmRecipe("true"), types.PipelineOptions{Timeout: 10})
	require.NoError(t, p.Run())

	want, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	got, err := filepath.EvalSymlinks(readLog(t, log)[0])
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestPipelineDryRunShowsWarm(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		warmService("a", "go", "go build ./..."),
		warmService("b", "go", "go build ./..."),
	}
	p := NewPipeline(services, warmRecipe("true"), types.PipelineOptions{DryRun: true})
	require.NoError(t, p.Run())

	require.Equal(t, types.DryRunEntry{Service: "a,b", Step: warmStepName, Command: "go build ./..."}, p.dryRunResults[0])
}

// goModule writes a Go module with a main package per service under root and
// returns root.
func goModule(t *testing.T, services ...string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/m\n\ngo 1.21\n"), 0o600))
	for _, svc := range services {
		dir := filepath.Join(root, "services", svc)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600))
	}
	return root
}

func autoGoService(root, name string, env ...serviceinfo.EnvVars) serviceinfo.ServiceInfo {
	return serviceinfo.ServiceInfo{
		Name:        name,
		Provider:    "test",
		Path:        filepath.Join(root, "services", name),
		BuildConfig: serviceinfo.BuildConfig{Language: "go", Cmd: "go build -o ./dist", EnvVars: env},
	}
}

func TestWarmGroupsAutoGo(t *testing.T) {
	t.Parallel()

	root := goModule(t, "api", "auth", "cli")
	linux := serviceinfo.EnvVars{Name: "GOOS", Value: "linux"}
	services := []serviceinfo.ServiceInfo{
		autoGoService(root, "api", linux),
		autoGoService(root, "auth", linux),
		// Different target, so a different cache: alone, it isn't worth warming.
		autoGoService(root, "cli", serviceinfo.EnvVars{Name: "GOOS", Value: "darwin"}),
	}

	p := NewPipeline(services, warmRecipe("true"), types.PipelineOptions{})
	groups := p.warmGroups(p.findMaxSteps())

	require.Len(t, groups, 1)
	g := groups[0]
	require.True(t, g.auto)
	require.Equal(t, "go build ./services/api ./services/auth", g.cmd)
	require.Equal(t, root, g.dir)
	require.Equal(t, map[string]string{"GOOS": "linux"}, g.env)
	require.Equal(t, "go (2 services, auto)", g.label())
}

func TestWarmGroupsAutoSkips(t *testing.T) {
	t.Parallel()

	root := goModule(t, "a", "b", "c", "d")

	optedOut := autoGoService(root, "a")
	optedOut.BuildConfig.WarmDisabled = true
	dockerOnly := autoGoService(root, "b")
	dockerOnly.BuildConfig.Cmd = ""
	explicit := autoGoService(root, "c")
	explicit.BuildConfig.Warm = "echo custom"

	p := NewPipeline([]serviceinfo.ServiceInfo{optedOut, dockerOnly, explicit, autoGoService(root, "d")},
		warmRecipe("true"), types.PipelineOptions{})
	groups := p.warmGroups(p.findMaxSteps())

	require.Len(t, groups, 1, "only the explicit warm: d is alone, a opted out, b builds in its Dockerfile")
	require.False(t, groups[0].auto)
	require.Equal(t, "echo custom", groups[0].cmd)
}

func TestPipelineAutoWarmCompilesWithoutTouchingTheProject(t *testing.T) {
	t.Parallel()

	root := goModule(t, "api", "auth")
	before := snapshot(t, root)

	services := []serviceinfo.ServiceInfo{autoGoService(root, "api"), autoGoService(root, "auth")}
	p := NewPipeline(services, warmRecipe("true"), types.PipelineOptions{Timeout: 120})
	require.NoError(t, p.Run())

	require.Equal(t, before, snapshot(t, root), "warming writes only to Go's cache")
}

// snapshot lists every file under root.
func snapshot(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(p string, _ os.DirEntry, err error) error {
		files = append(files, p)
		return err
	})
	require.NoError(t, err)
	return files
}

func TestIsolatedCopy(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "apps/web"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "lock"), []byte("l"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "apps/web/package.json"), []byte("{}"), 0o600))

	tmp, err := isolatedCopy(root, []string{"lock", "apps/web/package.json", "missing.rc"})
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmp) })

	data, err := os.ReadFile(filepath.Join(tmp, "apps/web/package.json"))
	require.NoError(t, err)
	require.Equal(t, "{}", string(data))
	require.FileExists(t, filepath.Join(tmp, "lock"))
	require.NoFileExists(t, filepath.Join(tmp, "missing.rc"), "optional files are skipped")
}

func TestRunWarmIsolatedRunsInACopyAndCleansUp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "lock"), []byte("l"), 0o600))
	log := filepath.Join(t.TempDir(), "log")

	p := NewPipeline(nil, nil, types.PipelineOptions{})
	g := &warmGroup{cmd: "pwd > " + log + " && test -f lock", dir: root, isolate: []string{"lock"}, timeout: 10, services: []string{"web"}}
	require.NoError(t, p.runWarm(g))

	ran := readLog(t, log)[0]
	require.NotEqual(t, root, ran)
	require.NoDirExists(t, ran, "the copy is removed")
}

func TestPipelineDryRunShowsAutoWarm(t *testing.T) {
	t.Parallel()

	root := goModule(t, "api", "auth")
	services := []serviceinfo.ServiceInfo{autoGoService(root, "api"), autoGoService(root, "auth")}
	p := NewPipeline(services, warmRecipe("true"), types.PipelineOptions{DryRun: true})
	require.NoError(t, p.Run())

	require.Equal(t, types.DryRunEntry{
		Service: "api,auth",
		Step:    warmStepName,
		Command: "(cd " + root + " && go build ./services/api ./services/auth)",
	}, p.dryRunResults[0])
}
