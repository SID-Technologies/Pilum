package orchestrator

import (
	"fmt"
	"sync"

	"github.com/sid-technologies/pilum/ingredients/build"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recipe"
	"github.com/sid-technologies/pilum/lib/registry"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"
)

// Pipeline executes deployment pipelines for multiple services.
type Pipeline struct {
	services      []serviceinfo.ServiceInfo
	recipes       map[string]recipe.Recipe
	imageNames    map[string]string // service name -> image name
	options       types.PipelineOptions
	output        *output.PipelineOutput
	results       []types.TaskResult
	resultsMu     sync.Mutex
	registry      *registry.CommandRegistry
	dryRunResults []types.DryRunEntry
}

// NewPipeline creates a new deployment pipeline.
func NewPipeline(services []serviceinfo.ServiceInfo, recipes []recipe.Info, opts types.PipelineOptions) *Pipeline {
	// Initialize command registry with default handlers
	cmdRegistry := registry.NewCommandRegistry()
	registry.RegisterDefaultHandlers(cmdRegistry)

	// Sort services by dependencies (topological order)
	sortedServices, err := serviceinfo.SortByDependencies(services)
	if err != nil {
		// Log warning but continue with original order
		output.Warning("Could not sort by dependencies: %v", err)
		sortedServices = services
	} else if hasDependencies(services) {
		output.Debugf("Services sorted by dependencies")
	}

	p := &Pipeline{
		services:   sortedServices,
		recipes:    make(map[string]recipe.Recipe),
		imageNames: make(map[string]string),
		options:    opts,
		output:     output.NewPipelineOutput(),
		registry:   cmdRegistry,
	}

	// Index recipes by composite key (e.g., "gcp-cloud-run") and provider-only fallback
	for _, rec := range recipes {
		p.recipes[rec.RecipeKey()] = rec.Recipe
		if _, exists := p.recipes[rec.Provider]; !exists {
			p.recipes[rec.Provider] = rec.Recipe
		}
	}

	// Calculate max name length for output alignment (use DisplayName for multi-region)
	maxLen := 0
	for _, svc := range services {
		displayName := svc.DisplayName()
		if len(displayName) > maxLen {
			maxLen = len(displayName)
		}
	}
	p.output.SetMaxNameLength(maxLen + 2)

	return p
}

// Run executes the full deployment pipeline.
func (p *Pipeline) Run() error {
	if len(p.services) == 0 {
		fmt.Println("No services to deploy")
		return nil
	}

	// Validate all services before execution
	if err := p.validateServices(); err != nil {
		return err
	}

	// Find max steps
	maxSteps := p.findMaxSteps()
	if maxSteps == 0 {
		fmt.Println("No recipe steps found for services")
		return nil
	}

	// Count how many steps will actually run (after tag filtering)
	runnableSteps := p.countRunnableSteps(maxSteps)

	p.output.PrintHeader(fmt.Sprintf("Deploying %d service(s)", len(p.services)))

	// Pre-calculate image names for all services
	for _, svc := range p.services {
		_, imageName := build.GenerateBuildCommand(svc, svc.RegistryName, p.options.Tag)
		p.imageNames[svc.Name] = imageName
	}

	// Execute step by step, tracking display number for runnable steps only
	displayStep := 0
	for stepIdx := 0; stepIdx < maxSteps; stepIdx++ {
		err := p.executeStep(stepIdx, runnableSteps, &displayStep)
		if err != nil {
			return err
		}
	}

	// Dry-run JSON: emit collected entries instead of the normal summary
	if p.options.DryRun && output.IsJSON() {
		output.JSON(struct {
			DryRun bool                `json:"dry_run"`
			Steps  []types.DryRunEntry `json:"steps"`
		}{true, p.dryRunResults})
		return nil
	}

	p.output.PrintComplete(p.results)
	return nil
}

// validateServices validates all services before execution.
// Returns an error if any service is invalid or has no matching recipe.
func (p *Pipeline) validateServices() error {
	for _, svc := range p.services {
		// Base validation (name, provider required)
		if err := svc.Validate(); err != nil {
			return errors.Wrap(err, "service '%s' validation failed", svc.Name)
		}

		// Check for matching recipe
		if _, exists := p.getRecipeForService(svc); !exists {
			return errors.New("service '%s' (recipe key '%s') has no matching recipe",
				svc.Name, svc.RecipeKey())
		}
	}
	return nil
}

// Results returns the collected task results after a run.
func (p *Pipeline) Results() []types.TaskResult {
	return p.results
}

// hasDependencies returns true if any service has dependencies.
func hasDependencies(services []serviceinfo.ServiceInfo) bool {
	for _, svc := range services {
		if len(svc.DependsOn) > 0 {
			return true
		}
	}
	return false
}
