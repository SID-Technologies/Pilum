package orchestrator

import (
	"strings"

	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recipe"
	"github.com/sid-technologies/pilum/lib/registry"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/shellutil"
	"github.com/sid-technologies/pilum/lib/types"
)

// executeStep runs step N for all services that have it.
// displayStep is a running counter incremented only for steps that actually execute.
func (p *Pipeline) executeStep(stepIdx, runnableSteps int, displayStep *int) error {
	// Collect tasks for this step
	var tasks []stepTask
	stepNames := make(map[string]bool)

	for _, svc := range p.services {
		recipe, exists := p.getRecipeForService(svc)
		if !exists {
			continue
		}
		if stepIdx >= len(recipe.Steps) {
			continue
		}
		step := &recipe.Steps[stepIdx]
		// Skip steps based on name/tag filters
		if p.shouldSkipStep(step) {
			continue
		}
		stepNames[step.Name] = true
		tasks = append(tasks, stepTask{service: svc, recipe: recipe, step: step})
	}

	if len(tasks) == 0 {
		return nil
	}

	// Increment display step counter for this runnable step
	*displayStep++

	// Build step name
	stepName := p.buildStepName(stepNames)
	p.output.PrintStepHeader(*displayStep, runnableSteps, stepName)

	// Show skipped services
	for _, svc := range p.services {
		recipe, exists := p.getRecipeForService(svc)
		if !exists {
			p.output.PrintSkipped(svc.DisplayName(), "no recipe")
		} else if stepIdx >= len(recipe.Steps) {
			p.output.PrintSkipped(svc.DisplayName(), "no step")
		}
	}

	if p.options.DryRun {
		if p.stepRequiresWaves(tasks) {
			return p.dryRunWithWaves(tasks)
		}
		for _, t := range tasks {
			cmd := p.generateCommand(t.service, t.step)
			p.output.PrintDryRun(t.service.DisplayName(), t.step.Name, cmd)
			p.dryRunResults = append(p.dryRunResults, types.DryRunEntry{
				Service: t.service.DisplayName(),
				Step:    t.step.Name,
				Command: output.FormatCommand(cmd),
			})
		}
		return nil
	}

	// Check if this step requires wave-based execution
	if p.stepRequiresWaves(tasks) {
		return p.executeStepWithWaves(tasks)
	}

	// Execute in parallel
	return p.executeTasksParallel(tasks)
}

// shouldSkipStep checks if a step should be skipped based on tag filters.
func (p *Pipeline) shouldSkipStep(step *recipe.Step) bool {
	// If OnlyTags is set, step must have at least one matching tag
	if len(p.options.OnlyTags) > 0 {
		if !p.stepHasAnyTag(step, p.options.OnlyTags) {
			return true
		}
	}

	// If ExcludeTags is set, skip steps that have any excluded tag
	if len(p.options.ExcludeTags) > 0 {
		if p.stepHasAnyTag(step, p.options.ExcludeTags) {
			return true
		}
	}

	return false
}

// stepHasAnyTag checks if a step has any of the specified tags.
func (*Pipeline) stepHasAnyTag(step *recipe.Step, tags []string) bool {
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

// buildStepName creates display name from step names.
func (*Pipeline) buildStepName(names map[string]bool) string {
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

// generateCommand creates the command for a step based on step name and provider.
func (p *Pipeline) generateCommand(svc serviceinfo.ServiceInfo, step *recipe.Step) any {
	// If step has explicit command, use it (with variable substitution)
	if step.Command != nil {
		return p.substituteVars(step.Command, svc)
	}

	// Look up handler from registry
	handler, found := p.registry.GetHandler(step.Name, svc.Provider)
	if !found {
		// Unknown step - let it pass (might be handled elsewhere)
		return nil
	}

	// Build context and execute handler
	// Use service's registry_name, fall back to template path from options if service doesn't specify
	templatePath := p.options.TemplatePath
	if templatePath == "" {
		templatePath = "./_templates"
	}
	ctx := registry.StepContext{
		Service:      svc,
		ImageName:    p.imageNames[svc.Name],
		Tag:          p.options.Tag,
		Registry:     svc.RegistryName,
		TemplatePath: templatePath,
	}

	result := handler(ctx)

	// Guard against Go's nil interface trap: a typed nil (e.g., nil []string)
	// wrapped in an any interface is NOT == nil. Check concrete types explicitly.
	if result == nil {
		return nil
	}
	switch v := result.(type) {
	case []string:
		if v == nil {
			return nil
		}
	case []any:
		if v == nil {
			return nil
		}
	}

	return result
}

// substituteVars replaces ${var} patterns in commands.
// For string commands (executed via sh -c), values are shell-quoted to prevent injection.
// For []string/[]any commands (executed via exec.Command with separate args), values are
// substituted without quoting since they don't pass through a shell.
func (p *Pipeline) substituteVars(cmd any, svc serviceinfo.ServiceInfo) any {
	// Shell-quoted replacer for string commands (passed to sh -c)
	shellReplacer := strings.NewReplacer(
		"${name}", shellutil.Quote(svc.Name),
		"${service.name}", shellutil.Quote(svc.Name),
		"${provider}", shellutil.Quote(svc.Provider),
		"${region}", shellutil.Quote(svc.Region),
		"${project}", shellutil.Quote(svc.Project),
		"${build.version}", shellutil.Quote(p.options.Tag),
		"${tag}", shellutil.Quote(p.options.Tag),
	)

	// Raw replacer for []string/[]any commands (exec.Command with separate args)
	rawReplacer := strings.NewReplacer(
		"${name}", svc.Name,
		"${service.name}", svc.Name,
		"${provider}", svc.Provider,
		"${region}", svc.Region,
		"${project}", svc.Project,
		"${build.version}", p.options.Tag,
		"${tag}", p.options.Tag,
	)

	switch v := cmd.(type) {
	case string:
		return shellReplacer.Replace(v)
	case []string:
		result := make([]string, len(v))
		for i, s := range v {
			result[i] = rawReplacer.Replace(s)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			if s, ok := item.(string); ok {
				result[i] = rawReplacer.Replace(s)
			} else {
				result[i] = item
			}
		}
		return result
	}
	return cmd
}

// getRecipeForService finds the recipe for a service using composite key with provider-only fallback.
func (p *Pipeline) getRecipeForService(svc serviceinfo.ServiceInfo) (recipe.Recipe, bool) {
	if recipe, ok := p.recipes[svc.RecipeKey()]; ok {
		return recipe, true
	}
	if recipe, ok := p.recipes[svc.Provider]; ok {
		return recipe, true
	}
	return recipe.Recipe{}, false
}

// findMaxSteps returns the max number of steps across all recipes.
func (p *Pipeline) findMaxSteps() int {
	maxSteps := 0
	for _, svc := range p.services {
		recipe, exists := p.getRecipeForService(svc)
		if !exists {
			continue
		}
		if len(recipe.Steps) > maxSteps {
			maxSteps = len(recipe.Steps)
		}
	}
	// Limit to MaxSteps if set
	if p.options.MaxSteps > 0 && p.options.MaxSteps < maxSteps {
		maxSteps = p.options.MaxSteps
	}
	return maxSteps
}

// countRunnableSteps returns how many step indices (up to maxSteps) have at least
// one service with a non-skipped step. This gives an accurate "X/Y" for headers
// when tag filtering is active (e.g., --only-tags build).
func (p *Pipeline) countRunnableSteps(maxSteps int) int {
	count := 0
	for stepIdx := 0; stepIdx < maxSteps; stepIdx++ {
		for _, svc := range p.services {
			recipe, exists := p.getRecipeForService(svc)
			if !exists || stepIdx >= len(recipe.Steps) {
				continue
			}
			if !p.shouldSkipStep(&recipe.Steps[stepIdx]) {
				count++
				break
			}
		}
	}
	return count
}

// stepHasAlwaysRun returns true if any recipe in this pipeline marks the step
// at stepIdx with always_run: true. Used by the pipeline loop to decide whether
// to continue execution past a failure. Returns false if no recipe declares it.
func (p *Pipeline) stepHasAlwaysRun(stepIdx int) bool {
	for _, svc := range p.services {
		recipe, exists := p.getRecipeForService(svc)
		if !exists || stepIdx >= len(recipe.Steps) {
			continue
		}
		if recipe.Steps[stepIdx].AlwaysRun {
			return true
		}
	}
	return false
}
