package orchestrator

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator/workers"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/sysinfo"
	"github.com/sid-technologies/pilum/lib/templates"
	"github.com/sid-technologies/pilum/lib/types"
)

// executeTasksParallel runs tasks concurrently with a worker pool.
func (p *Pipeline) executeTasksParallel(tasks []stepTask) error {
	var wg sync.WaitGroup
	resultChan := make(chan types.TaskResult, len(tasks))
	plan := p.planWorkers(tasks)
	semaphore := make(chan struct{}, plan.workers)

	// Create and start spinner manager
	spinner := output.NewSpinnerManager()

	// Add all spinners first (so they're all visible)
	for _, t := range tasks {
		spinner.AddSpinner(t.service.DisplayName(), t.step.Name, p.output.MaxNameLen)
	}

	spinner.Start()

	for _, t := range tasks {
		wg.Add(1)
		task := t // capture

		go func() {
			defer wg.Done()
			semaphore <- struct{}{}        // acquire
			defer func() { <-semaphore }() // release

			startTime := time.Now()

			result := p.executeTask(task.service, task.step, plan.threads)
			result.Duration = time.Since(startTime)

			spinner.Complete(task.service.DisplayName(), result.Success, result.Duration, result.Error)

			resultChan <- result
		}()
	}

	wg.Wait()
	spinner.Stop()
	spinner.RenderFinal()
	close(resultChan)

	// Collect results
	var failed []string
	for result := range resultChan {
		p.resultsMu.Lock()
		p.results = append(p.results, result)
		p.resultsMu.Unlock()

		if !result.Success {
			failed = append(failed, result.ServiceName)
		}
	}

	if len(failed) > 0 {
		return errors.New("step failed for: %s", strings.Join(failed, ", "))
	}

	return nil
}

const (
	// defaultBuildCPUs is how many cores one build is assumed to use when
	// resources.cpu is unset. go build, docker build (BuildKit) and bundlers
	// are all internally parallel, so charging one core per build (Bazel's
	// single-threaded default) oversubscribes the machine N-fold.
	defaultBuildCPUs = 2

	// maxIOWorkers caps steps that mostly wait on the network (push, deploy,
	// execute). They use little local CPU, so they get their own, larger
	// limit instead of competing for cores (as Pants does for remote work).
	maxIOWorkers = 16
)

// ioBoundTags mark steps that wait on a registry or cloud API.
var ioBoundTags = []string{"push", "deploy", "execute"}

// workerPlan is how a step's tasks are scheduled.
type workerPlan struct {
	workers int
	// threads is GOMAXPROCS for each task: the core budget split across
	// concurrent builds, so N builds don't each assume the whole machine.
	// 0 leaves GOMAXPROCS alone.
	threads int
}

// planWorkers decides concurrency for one step's tasks.
//
// CPU-bound steps (tagged "build", or untagged) are budgeted by cores and
// memory, where each build costs resources.cpu cores (default 2). Steps that
// invoke docker are sized from the docker daemon, which on Docker Desktop is
// a VM smaller than the host. I/O-bound steps get a flat, larger limit.
// --max-workers overrides the worker count for every step.
func (p *Pipeline) planWorkers(tasks []stepTask) workerPlan {
	cpuBound := p.isCPUBound(tasks)

	cpus := runtime.NumCPU()
	memMB := sysinfo.TotalMemoryMB()
	source := "host"
	if cpuBound && p.dockerResources.CPUs > 0 && p.tasksInvokeDocker(tasks) {
		cpus = p.dockerResources.CPUs
		if p.dockerResources.MemoryMB > 0 {
			memMB = p.dockerResources.MemoryMB
		}
		source = "docker"
	}

	if p.options.MaxWorkers > 0 {
		plan := workerPlan{workers: p.options.MaxWorkers}
		if cpuBound {
			plan.threads = max(1, cpus/plan.workers)
		}
		output.Info("Using %d workers (explicit)", plan.workers)
		return plan
	}

	if !cpuBound {
		w := max(1, min(len(tasks), maxIOWorkers))
		output.Info("Using %d workers (network-bound step, %d tasks)", w, len(tasks))
		return workerPlan{workers: w}
	}

	perBuild := min(p.maxBuildCPUs(), cpus)
	w := cpus / perBuild

	// Memory-based limit: find the most expensive build language across services
	// and calculate how many can run concurrently within available memory.
	memWorkers := 0
	if memMB > 0 {
		if maxPerBuild := p.maxBuildMemoryMB(); maxPerBuild > 0 {
			// Reserve ~25% for OS and other processes (similar to Bazel's 0.67 factor)
			available := memMB * 75 / 100
			memWorkers = max(1, available/maxPerBuild)
			w = min(w, memWorkers)
		}
	}

	w = max(1, min(w, len(tasks)))
	threads := max(1, cpus/w)

	output.Info("Using %d workers (%s: %d CPUs ÷ %d per build, %dMB RAM → %d by memory, %d tasks; GOMAXPROCS=%d)",
		w, source, cpus, perBuild, memMB, memWorkers, len(tasks), threads)
	return workerPlan{workers: w, threads: threads}
}

// isCPUBound reports whether a step's tasks compete for local cores.
// Untagged steps are treated as CPU-bound, the safe assumption.
func (p *Pipeline) isCPUBound(tasks []stepTask) bool {
	for _, t := range tasks {
		if p.stepHasAnyTag(t.step, []string{"build"}) || !p.stepHasAnyTag(t.step, ioBoundTags) {
			return true
		}
	}
	return false
}

// tasksInvokeDocker reports whether any task runs the docker CLI.
func (p *Pipeline) tasksInvokeDocker(tasks []stepTask) bool {
	for _, t := range tasks {
		if invokesDocker(p.generateCommand(t.service, t.step)) {
			return true
		}
	}
	return false
}

// maxBuildCPUs returns the highest declared resources.cpu across services,
// or defaultBuildCPUs when none is set.
func (p *Pipeline) maxBuildCPUs() int {
	maxCPU := 0
	for _, svc := range p.services {
		maxCPU = max(maxCPU, svc.BuildConfig.Resources.CPU)
	}
	if maxCPU == 0 {
		return defaultBuildCPUs
	}
	return maxCPU
}

// maxBuildMemoryMB returns the highest estimated build memory across all services.
// Uses explicit resources.memory from pilum.yaml if set, otherwise language defaults
// from the embedded build templates.
func (p *Pipeline) maxBuildMemoryMB() int {
	maxMem := 0
	for _, svc := range p.services {
		mem := svc.BuildConfig.Resources.Memory
		if mem == 0 {
			mem = templates.MemoryForLanguage(svc.BuildConfig.Language)
		}
		if mem > maxMem {
			maxMem = mem
		}
	}
	return maxMem
}

// executeTask runs a single task. threads > 0 sets GOMAXPROCS for the command
// unless the step, service, or caller's environment already sets it.
func (p *Pipeline) executeTask(svc serviceinfo.ServiceInfo, step *recipe.Step, threads int) types.TaskResult {
	result := types.TaskResult{
		ServiceName: svc.DisplayName(),
		StepName:    step.Name,
	}

	cmd := p.generateCommand(svc, step)
	if cmd == nil {
		result.Success = true
		return result
	}

	// Determine working directory
	cwd := ""
	execMode := step.ExecutionMode
	if execMode == "" {
		execMode = "root"
	}
	if execMode == "service_dir" {
		cwd = svc.Path
	}

	// Get timeout and retries
	timeout := p.options.Timeout
	if step.Timeout > 0 {
		timeout = step.Timeout
	}
	retries := p.options.Retries
	if step.Retries > 0 {
		retries = step.Retries
	}

	// Build env vars from step and service
	envVars := make(map[string]string)
	for _, ev := range svc.BuildConfig.EnvVars {
		envVars[ev.Name] = ev.Value
	}
	for k, v := range step.EnvVars {
		envVars[k] = v
	}
	if _, set := envVars["GOMAXPROCS"]; threads > 0 && !set && os.Getenv("GOMAXPROCS") == "" {
		envVars["GOMAXPROCS"] = strconv.Itoa(threads)
	}

	taskInfo := workers.NewTaskInfo(
		cmd,
		cwd,
		svc.Name,
		execMode,
		envVars,
		step.BuildFlags,
		timeout,
		p.options.Debug,
		retries,
	)

	success, err := workers.CommandWorker(taskInfo)
	result.Success = success
	result.Error = err

	return result
}
