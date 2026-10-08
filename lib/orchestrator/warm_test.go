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
