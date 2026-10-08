package orchestrator

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sid-technologies/pilum/lib/orchestrator/workers"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"
)

// warmStepName labels the warm phase in output and dry-run JSON.
const warmStepName = "warm build cache"

// warmGroup is one build.warm command and the services that share it.
type warmGroup struct {
	cmd      string
	dir      string // relative to the project root; "" is the root
	env      map[string]string
	timeout  int
	language string
	services []string
}

// label names the group in output, e.g. "go (10 services)".
func (g *warmGroup) label() string {
	name := g.language
	if name == "" {
		name = strings.Fields(g.cmd)[0]
	}
	if g.dir != "" {
		name += " in " + g.dir
	}
	if len(g.services) == 1 {
		return fmt.Sprintf("%s (%s)", name, g.services[0])
	}
	return fmt.Sprintf("%s (%d services)", name, len(g.services))
}

// warmGroups collects the distinct build.warm commands of services that will
// run a build step. Services agree on a command when its text, directory and
// build env all match: the env is part of the key because a cache filled for
// one GOOS/GOARCH misses for another.
func (p *Pipeline) warmGroups(maxSteps int) []*warmGroup {
	var groups []*warmGroup
	byKey := make(map[string]*warmGroup)

	for _, svc := range p.services {
		cmd := strings.TrimSpace(svc.BuildConfig.Warm)
		if cmd == "" {
			continue
		}
		step := p.firstBuildStep(svc, maxSteps)
		if step == nil {
			continue
		}

		env := warmEnv(svc, step)
		timeout := p.options.Timeout
		if step.Timeout > 0 {
			timeout = step.Timeout
		}

		key := warmKey(cmd, svc.BuildConfig.WarmDir, env)
		g, exists := byKey[key]
		if !exists {
			g = &warmGroup{cmd: cmd, dir: svc.BuildConfig.WarmDir, env: env, language: svc.BuildConfig.Language}
			byKey[key] = g
			groups = append(groups, g)
		}
		g.timeout = max(g.timeout, timeout)
		g.services = append(g.services, svc.DisplayName())
	}
	return groups
}

// firstBuildStep returns the service's first build step that will run, or nil
// when none will (say, under --only-tags deploy) and warming would be wasted.
func (p *Pipeline) firstBuildStep(svc serviceinfo.ServiceInfo, maxSteps int) *recipe.Step {
	rec, exists := p.getRecipeForService(svc)
	if !exists {
		return nil
	}
	for i := 0; i < maxSteps && i < len(rec.Steps); i++ {
		step := &rec.Steps[i]
		if p.shouldSkipStep(step) || !p.stepHasAnyTag(step, []string{"build"}) {
			continue
		}
		return step
	}
	return nil
}

// warmEnv is the env the service's build step runs with, so the warm run
// fills the cache entries the build will look up.
func warmEnv(svc serviceinfo.ServiceInfo, step *recipe.Step) map[string]string {
	env := make(map[string]string)
	for _, ev := range svc.BuildConfig.EnvVars {
		env[ev.Name] = ev.Value
	}
	maps.Copy(env, step.EnvVars)
	return env
}

func warmKey(cmd, dir string, env map[string]string) string {
	parts := []string{dir, cmd}
	for _, k := range slices.Sorted(maps.Keys(env)) {
		parts = append(parts, k+"="+env[k])
	}
	return strings.Join(parts, "\x00")
}

// warm runs every distinct build.warm command, concurrently, before any
// service's build starts.
//
// Builds running in parallel against a cold cache all miss on the same
// shared dependencies and each compiles them; a cache can be shared safely
// without deduplicating the work in flight. One warm run per command fills
// the cache once, so the builds that follow only compile what is theirs.
//
// Warming is an optimization, so a failure is reported and the pipeline goes
// on: the builds still run, cold, and surface any real error themselves.
func (p *Pipeline) warm(maxSteps int) {
	if p.options.NoWarm {
		return
	}
	groups := p.warmGroups(maxSteps)
	if len(groups) == 0 {
		return
	}

	p.output.PrintPhaseHeader("Warming build caches")

	if p.options.DryRun {
		for _, g := range groups {
			p.output.PrintDryRun(g.label(), warmStepName, warmCommand(g))
			p.dryRunResults = append(p.dryRunResults, types.DryRunEntry{
				Service: strings.Join(g.services, ","),
				Step:    warmStepName,
				Command: warmCommand(g),
			})
		}
		return
	}

	spinner := output.NewSpinnerManager()
	for _, g := range groups {
		spinner.AddSpinner(g.label(), warmStepName, p.output.MaxNameLen)
	}
	spinner.Start()

	var wg sync.WaitGroup
	var failed atomic.Bool
	for _, g := range groups {
		wg.Go(func() {
			start := time.Now()
			_, err := workers.CommandWorker(p.warmTask(g))
			if err != nil {
				failed.Store(true)
			}
			spinner.Complete(g.label(), err == nil, time.Since(start), err)
		})
	}
	wg.Wait()
	spinner.Stop()
	spinner.RenderFinal()

	if failed.Load() {
		output.Warning("Warming failed; continuing with cold builds")
	}
}

// warmCommand is the command as a dry run shows it, with its directory.
func warmCommand(g *warmGroup) string {
	if g.dir == "" {
		return g.cmd
	}
	return fmt.Sprintf("(cd %s && %s)", g.dir, g.cmd)
}

func (p *Pipeline) warmTask(g *warmGroup) *workers.TaskInfo {
	mode := "root"
	if g.dir != "" {
		mode = "service_dir"
	}
	task := workers.NewTaskInfo(g.cmd, g.dir, g.label(), mode, g.env, nil, g.timeout, p.options.Debug, 0)
	// A failed warm leaves nothing for a retry to fix that the builds won't
	// also hit, and NewTaskInfo turns 0 into its default of 3.
	task.Retries = 0
	return task
}
