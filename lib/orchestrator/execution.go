package orchestrator

import (
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator/workers"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"
)

// executeTasksParallel runs tasks concurrently with a worker pool.
func (p *Pipeline) executeTasksParallel(tasks []stepTask) error {
	var wg sync.WaitGroup
	resultChan := make(chan types.TaskResult, len(tasks))
	semaphore := make(chan struct{}, p.getWorkerCount())

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

			result := p.executeTask(task.service, task.step)
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

// getWorkerCount returns the number of workers to use.
// When auto (MaxWorkers=0), uses one worker per CPU core, capped by the number of services.
func (p *Pipeline) getWorkerCount() int {
	if p.options.MaxWorkers > 0 {
		output.Info("Using %d workers (explicit)", p.options.MaxWorkers)
		return p.options.MaxWorkers
	}

	cpus := runtime.NumCPU()
	w := cpus
	if w > len(p.services) {
		w = len(p.services)
	}

	output.Info("Using %d workers (auto: %d CPUs detected, %d services)", w, cpus, len(p.services))
	return w
}

// executeTask runs a single task.
func (p *Pipeline) executeTask(svc serviceinfo.ServiceInfo, step *recepie.RecipeStep) types.TaskResult {
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
