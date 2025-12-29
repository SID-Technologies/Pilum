package orchestrator

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sid-technologies/pilum/ingredients/build"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/recepie"
	"github.com/sid-technologies/pilum/lib/registry"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	workerqueue "github.com/sid-technologies/pilum/lib/worker_queue"
)

// TaskResult holds the result of a service task execution.
type TaskResult struct {
	ServiceName string
	StepName    string
	Success     bool
	Duration    time.Duration
	Error       error
}

// Runner executes deployment pipelines for multiple services.
type Runner struct {
	services   []serviceinfo.ServiceInfo
	recipes    map[string]recepie.Recipe
	imageNames map[string]string // service name -> image name
	options    RunnerOptions
	output     *OutputManager
	results    []TaskResult
	resultsMu  sync.Mutex
	registry   *registry.CommandRegistry
}

// stepTask represents a task for a specific service at a specific step.
type stepTask struct {
	service serviceinfo.ServiceInfo
	recipe  recepie.Recipe
	step    *recepie.RecipeStep
}

// RunnerOptions configures the runner.
type RunnerOptions struct {
	Tag          string
	Registry     string // Docker registry prefix (overrides service.yaml)
	TemplatePath string // Default template path for services that don't specify one
	Debug        bool
	Timeout      int
	Retries      int
	DryRun       bool
	MaxWorkers   int
	MaxSteps     int      // Maximum number of steps to run (0 = all)
	ExcludeTags  []string // Exclude steps with these tags (e.g., "deploy")
	OnlyTags     []string // Only run steps with these tags (e.g., "deploy")
}

// NewRunner creates a new deployment runner.
func NewRunner(services []serviceinfo.ServiceInfo, recipes []recepie.RecipeInfo, opts RunnerOptions) *Runner {
	// Initialize command registry with default handlers
	cmdRegistry := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(cmdRegistry)

	r := &Runner{
		services:   services,
		recipes:    make(map[string]recepie.Recipe),
		imageNames: make(map[string]string),
		options:    opts,
		output:     NewOutputManager(),
		registry:   cmdRegistry,
	}

	// Index recipes by provider
	for _, rec := range recipes {
		r.recipes[rec.Provider] = rec.Recipe
	}

	// Calculate max name length for output alignment (use DisplayName for multi-region)
	maxLen := 0
	for _, svc := range services {
		displayName := svc.DisplayName()
		if len(displayName) > maxLen {
			maxLen = len(displayName)
		}
	}
	r.output.SetMaxNameLength(maxLen + 2)

	return r
}

// Run executes the full deployment pipeline.
func (r *Runner) Run() error {
	if len(r.services) == 0 {
		fmt.Println("No services to deploy")
		return nil
	}

	// Find max steps
	maxSteps := r.findMaxSteps()
	if maxSteps == 0 {
		fmt.Println("No recipe steps found for services")
		return nil
	}

	r.output.PrintHeader(fmt.Sprintf("Deploying %d service(s)", len(r.services)))

	// Pre-calculate image names for all services
	for _, svc := range r.services {
		_, imageName := build.GenerateBuildCommand(svc, svc.RegistryName, r.options.Tag)
		r.imageNames[svc.Name] = imageName
	}

	// Execute step by step
	for stepIdx := 0; stepIdx < maxSteps; stepIdx++ {
		err := r.executeStep(stepIdx, maxSteps)
		if err != nil {
			return err
		}
	}

	r.output.PrintComplete(r.results)
	return nil
}

// findMaxSteps returns the max number of steps across all recipes.
func (r *Runner) findMaxSteps() int {
	maxSteps := 0
	for _, svc := range r.services {
		recipe, exists := r.recipes[svc.Provider]
		if !exists {
			continue
		}
		if len(recipe.Steps) > maxSteps {
			maxSteps = len(recipe.Steps)
		}
	}
	// Limit to MaxSteps if set
	if r.options.MaxSteps > 0 && r.options.MaxSteps < maxSteps {
		maxSteps = r.options.MaxSteps
	}
	return maxSteps
}

// shouldSkipStep checks if a step should be skipped based on tag filters.
func (r *Runner) shouldSkipStep(step *recepie.RecipeStep) bool {
	// If OnlyTags is set, step must have at least one matching tag
	if len(r.options.OnlyTags) > 0 {
		if !r.stepHasAnyTag(step, r.options.OnlyTags) {
			return true
		}
	}

	// If ExcludeTags is set, skip steps that have any excluded tag
	if len(r.options.ExcludeTags) > 0 {
		if r.stepHasAnyTag(step, r.options.ExcludeTags) {
			return true
		}
	}

	return false
}

// stepHasAnyTag checks if a step has any of the specified tags.
func (*Runner) stepHasAnyTag(step *recepie.RecipeStep, tags []string) bool {
	for _, stepTag := range step.Tags {
		stepTagLower := strings.ToLower(stepTag)
		for _, tag := range tags {
			if stepTagLower == strings.ToLower(tag) {
				return true
			}
		}
	}
	return false
}

// executeStep runs step N for all services that have it.
func (r *Runner) executeStep(stepIdx, totalSteps int) error {
	// Collect tasks for this step
	var tasks []stepTask
	stepNames := make(map[string]bool)

	for _, svc := range r.services {
		recipe, exists := r.recipes[svc.Provider]
		if !exists {
			continue
		}
		if stepIdx >= len(recipe.Steps) {
			continue
		}
		step := &recipe.Steps[stepIdx]
		// Skip steps based on name/tag filters
		if r.shouldSkipStep(step) {
			continue
		}
		stepNames[step.Name] = true
		tasks = append(tasks, stepTask{service: svc, recipe: recipe, step: step})
	}

	if len(tasks) == 0 {
		return nil
	}

	// Build step name
	stepName := r.buildStepName(stepNames)
	r.output.PrintStepHeader(stepIdx+1, totalSteps, stepName)

	// Show skipped services
	for _, svc := range r.services {
		recipe, exists := r.recipes[svc.Provider]
		if !exists {
			r.output.PrintSkipped(svc.DisplayName(), "no recipe")
		} else if stepIdx >= len(recipe.Steps) {
			r.output.PrintSkipped(svc.DisplayName(), "no step")
		}
	}

	if r.options.DryRun {
		for _, t := range tasks {
			cmd := r.generateCommand(t.service, t.step)
			r.output.PrintDryRun(t.service.DisplayName(), t.step.Name, cmd)
		}
		return nil
	}

	// Execute in parallel
	return r.executeTasksParallel(tasks)
}

// buildStepName creates display name from step names.
func (*Runner) buildStepName(names map[string]bool) string {
	if len(names) == 1 {
		for name := range names {
			return name
		}
	}
	var parts []string
	for name := range names {
		parts = append(parts, name)
	}
	return strings.Join(parts, " / ")
}

// executeTasksParallel runs tasks concurrently with a worker pool.
func (r *Runner) executeTasksParallel(tasks []stepTask) error {
	var wg sync.WaitGroup
	resultChan := make(chan TaskResult, len(tasks))
	semaphore := make(chan struct{}, r.getWorkerCount())

	// Create and start spinner manager
	spinner := NewSpinnerManager()

	// Add all spinners first (so they're all visible)
	for _, t := range tasks {
		spinner.AddSpinner(t.service.DisplayName(), t.step.Name, r.output.maxNameLen)
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

			result := r.executeTask(task.service, task.step)
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
		r.resultsMu.Lock()
		r.results = append(r.results, result)
		r.resultsMu.Unlock()

		if !result.Success {
			failed = append(failed, result.ServiceName)
		}
	}

	if len(failed) > 0 {
		return errors.New("step failed for: %s", strings.Join(failed, ", "))
	}

	return nil
}

// executeTask runs a single task.
func (r *Runner) executeTask(svc serviceinfo.ServiceInfo, step *recepie.RecipeStep) TaskResult {
	result := TaskResult{
		ServiceName: svc.DisplayName(),
		StepName:    step.Name,
	}

	cmd := r.generateCommand(svc, step)
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
	timeout := r.options.Timeout
	if step.Timeout > 0 {
		timeout = step.Timeout
	}
	retries := r.options.Retries
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

	taskInfo := workerqueue.NewTaskInfo(
		cmd,
		cwd,
		svc.Name,
		execMode,
		envVars,
		step.BuildFlags,
		timeout,
		r.options.Debug,
		retries,
	)

	success, err := workerqueue.CommandWorker(taskInfo)
	result.Success = success
	result.Error = err
	return result
}

// generateCommand creates the command for a step based on step name and provider.
func (r *Runner) generateCommand(svc serviceinfo.ServiceInfo, step *recepie.RecipeStep) any {
	// If step has explicit command, use it (with variable substitution)
	if step.Command != nil {
		return r.substituteVars(step.Command, svc)
	}

	// Look up handler from registry
	handler, found := r.registry.GetHandler(step.Name, svc.Provider)
	if !found {
		// Unknown step - let it pass (might be handled elsewhere)
		return nil
	}

	// Build context and execute handler
	// Use service's registry_name, fall back to template path from options if service doesn't specify
	templatePath := r.options.TemplatePath
	if templatePath == "" {
		templatePath = "./_templates"
	}
	ctx := registry.StepContext{
		Service:      svc,
		ImageName:    r.imageNames[svc.Name],
		Tag:          r.options.Tag,
		Registry:     svc.RegistryName,
		TemplatePath: templatePath,
	}

	return handler(ctx)
}

// substituteVars replaces ${var} patterns in commands.
func (r *Runner) substituteVars(cmd any, svc serviceinfo.ServiceInfo) any {
	replacer := strings.NewReplacer(
		"${name}", svc.Name,
		"${service.name}", svc.Name,
		"${provider}", svc.Provider,
		"${region}", svc.Region,
		"${project}", svc.Project,
		"${build.version}", r.options.Tag,
		"${tag}", r.options.Tag,
	)

	switch v := cmd.(type) {
	case string:
		return replacer.Replace(v)
	case []string:
		result := make([]string, len(v))
		for i, s := range v {
			result[i] = replacer.Replace(s)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			if s, ok := item.(string); ok {
				result[i] = replacer.Replace(s)
			} else {
				result[i] = item
			}
		}
		return result
	}
	return cmd
}

// getWorkerCount returns the number of workers to use.
func (r *Runner) getWorkerCount() int {
	if r.options.MaxWorkers > 0 {
		return r.options.MaxWorkers
	}
	// Default: use number of services or 4, whichever is smaller
	if len(r.services) < 4 {
		return len(r.services)
	}
	return 4
}
