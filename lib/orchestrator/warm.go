package orchestrator

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator/workers"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/path"
	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/toolchain"
	"github.com/sid-technologies/pilum/lib/types"
)

// warmStepName labels the warm phase in output and dry-run JSON.
const warmStepName = "warm build cache"

// minAutoGroup is the smallest group worth warming automatically: warming a
// lone service is the same work its build would do, only earlier.
const minAutoGroup = 2

// warmGroup is one warm command and the services that share it.
type warmGroup struct {
	cmd      string
	dir      string // build.warm: relative to the project root, "" is the root; auto: absolute
	env      map[string]string
	timeout  int
	language string
	services []string
	auto     bool     // planned from the detected toolchain, not build.warm
	isolate  []string // files copied to a temp dir for cmd to run in; see toolchain.Plan
}

// label names the group in output, e.g. "go (10 services)".
func (g *warmGroup) label() string {
	name := g.language
	if name == "" {
		name = strings.Fields(g.cmd)[0]
	}
	if g.dir != "" && !g.auto {
		name += " in " + g.dir
	}
	detail := g.services[0]
	if len(g.services) > 1 {
		detail = fmt.Sprintf("%d services", len(g.services))
	}
	if g.auto {
		detail += ", auto"
	}
	return fmt.Sprintf("%s (%s)", name, detail)
}

// autoBucket gathers services that one detected toolchain could warm together.
type autoBucket struct {
	tc       toolchain.Toolchain
	root     string
	group    warmGroup
	svc      serviceinfo.ServiceInfo
	dirs     []string
	services []string
}

// warmGroups collects what to warm for the services that will run a build
// step: their distinct build.warm commands and, for services that don't set
// one, a command planned from the detected toolchain. Services share a run
// only when its command, directory and build env all match: the env is part
// of the key because a cache filled for one GOOS/GOARCH misses for another.
func (p *Pipeline) warmGroups(maxSteps int) []*warmGroup {
	var groups []*warmGroup
	explicit := make(map[string]*warmGroup)
	var buckets []*autoBucket
	byAutoKey := make(map[string]*autoBucket)
	bound := warmBound()

	for _, svc := range p.services {
		if svc.BuildConfig.WarmDisabled {
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

		cmd := strings.TrimSpace(svc.BuildConfig.Warm)
		if cmd != "" {
			key := warmKey(cmd, svc.BuildConfig.WarmDir, env)
			g, exists := explicit[key]
			if !exists {
				g = &warmGroup{cmd: cmd, dir: svc.BuildConfig.WarmDir, env: env, language: svc.BuildConfig.Language}
				explicit[key] = g
				groups = append(groups, g)
			}
			g.timeout = max(g.timeout, timeout)
			g.services = append(g.services, svc.DisplayName())
			continue
		}

		// A service built only inside its Dockerfile has no host build to
		// speed up.
		if strings.TrimSpace(svc.BuildConfig.Cmd) == "" {
			continue
		}
		dir, err := filepath.Abs(svc.Path)
		if err != nil {
			continue
		}
		tc, root, ok := toolchain.Resolve(dir, bound, svc.BuildConfig.Language)
		if !ok {
			continue
		}
		key := warmKey(tc.Name()+"\x00"+tc.Variant(svc), root, env)
		b, exists := byAutoKey[key]
		if !exists {
			b = &autoBucket{tc: tc, root: root, svc: svc, group: warmGroup{env: env, language: tc.Name(), auto: true}}
			byAutoKey[key] = b
			buckets = append(buckets, b)
		}
		b.group.timeout = max(b.group.timeout, timeout)
		b.dirs = append(b.dirs, dir)
		b.services = append(b.services, svc.DisplayName())
	}

	for _, b := range buckets {
		if len(b.services) < minAutoGroup {
			continue
		}
		plan, ok := b.tc.Plan(toolchain.Group{Root: b.root, Dirs: b.dirs, Service: b.svc})
		if !ok {
			continue
		}
		g := b.group
		g.cmd = plan.Cmd
		g.dir = plan.Dir
		g.isolate = plan.Isolate
		g.services = b.services
		g.env = maps.Clone(g.env)
		maps.Copy(g.env, plan.Env)
		groups = append(groups, &g)
	}
	return groups
}

// warmBound is the highest directory toolchain detection searches: the
// repository holding the working directory.
func warmBound() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return path.RepoRoot(cwd)
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
			err := p.runWarm(g)
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
	dir := displayDir(g.dir)
	switch {
	case len(g.isolate) > 0:
		return fmt.Sprintf("(in a copy of %s from %s: %s)", strings.Join(g.isolate, ", "), dir, g.cmd)
	case dir == "" || dir == ".":
		return g.cmd
	default:
		return fmt.Sprintf("(cd %s && %s)", dir, g.cmd)
	}
}

// displayDir shows an absolute dir relative to the working directory when
// it's inside it.
func displayDir(dir string) string {
	if !filepath.IsAbs(dir) {
		return dir
	}
	cwd, err := os.Getwd()
	if err != nil {
		return dir
	}
	rel, err := filepath.Rel(cwd, dir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return dir
	}
	return rel
}

// runWarm runs a group's command, in a temporary copy of its files when the
// group is isolated.
func (p *Pipeline) runWarm(g *warmGroup) error {
	dir := g.dir
	if len(g.isolate) > 0 {
		tmp, err := isolatedCopy(g.dir, g.isolate)
		if err != nil {
			return err
		}
		defer func() { _ = os.RemoveAll(tmp) }()
		dir = tmp
	}
	_, err := workers.CommandWorker(p.warmTask(g, dir))
	return err
}

// isolatedCopy copies files, relative to root, into a new temporary directory
// with the same layout. Missing files are skipped: they're optional configs.
// Both sides go through os.Root, so a path can't escape either directory.
func isolatedCopy(root string, files []string) (string, error) {
	tmp, err := os.MkdirTemp("", "pilum-warm-")
	if err != nil {
		return "", errors.Wrap(err, "creating warm directory")
	}
	err = copyInto(root, tmp, files)
	if err != nil {
		_ = os.RemoveAll(tmp)
		return "", err
	}
	return tmp, nil
}

func copyInto(srcDir, dstDir string, files []string) error {
	src, err := os.OpenRoot(srcDir)
	if err != nil {
		return errors.Wrap(err, "opening %s", srcDir)
	}
	defer src.Close()
	dst, err := os.OpenRoot(dstDir)
	if err != nil {
		return errors.Wrap(err, "opening %s", dstDir)
	}
	defer dst.Close()

	for _, rel := range files {
		data, err := src.ReadFile(rel)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return errors.Wrap(err, "reading %s", rel)
		}
		err = dst.MkdirAll(filepath.Dir(rel), 0o755)
		if err != nil {
			return errors.Wrap(err, "copying %s", rel)
		}
		err = dst.WriteFile(rel, data, 0o600)
		if err != nil {
			return errors.Wrap(err, "copying %s", rel)
		}
	}
	return nil
}

func (p *Pipeline) warmTask(g *warmGroup, dir string) *workers.TaskInfo {
	mode := "root"
	if dir != "" {
		mode = "service_dir"
	}
	task := workers.NewTaskInfo(g.cmd, dir, g.label(), mode, g.env, nil, g.timeout, p.options.Debug, 0)
	// A failed warm leaves nothing for a retry to fix that the builds won't
	// also hit, and NewTaskInfo turns 0 into its default of 3.
	task.Retries = 0
	return task
}
