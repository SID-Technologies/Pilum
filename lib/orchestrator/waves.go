package orchestrator

import (
	"fmt"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"
)

// stepRequiresWaves returns true if this step should use wave-based execution.
// Waves are only used when: NoDeps is false, services have dependencies, AND
// at least one task's step has a "build", "deploy", or "execute" tag.
func (p *Pipeline) stepRequiresWaves(tasks []stepTask) bool {
	if p.options.NoDeps {
		return false
	}

	// Check if any service has dependencies
	hasDeps := false
	for _, t := range tasks {
		if len(t.service.DependsOn) > 0 {
			hasDeps = true
			break
		}
	}
	if !hasDeps {
		return false
	}

	// Check if any task's step has a build, deploy, or execute tag
	waveTags := []string{"build", "deploy", "execute"}
	for _, t := range tasks {
		if p.stepHasAnyTag(t.step, waveTags) {
			return true
		}
	}

	return false
}

// executeStepWithWaves executes tasks grouped by dependency waves.
// Each wave runs in parallel, but waves are executed sequentially.
func (p *Pipeline) executeStepWithWaves(tasks []stepTask) error {
	// Extract services for wave calculation
	services := make([]serviceinfo.ServiceInfo, len(tasks))
	for i, t := range tasks {
		services[i] = t.service
	}

	waves, err := serviceinfo.CalculateWaves(services)
	if err != nil {
		return err
	}

	// If no waves (no deps), fall back to flat parallel
	if waves == nil {
		return p.executeTasksParallel(tasks)
	}

	// Build lookup from service display name to task
	taskByDisplay := make(map[string]stepTask)
	for _, t := range tasks {
		taskByDisplay[t.service.DisplayName()] = t
	}

	failedServices := make(map[string]bool)

	for waveIdx, wave := range waves {
		p.output.PrintWaveHeader(waveIdx+1, len(waves))

		// Filter out services whose dependencies failed
		var waveTasks []stepTask
		for _, svc := range wave {
			// Check if any dependency failed
			failedDep := ""
			for _, dep := range svc.DependsOn {
				if failedServices[dep] {
					failedDep = dep
					break
				}
			}

			if failedDep != "" {
				p.output.PrintSkippedDependency(svc.DisplayName(), failedDep)
				failedServices[svc.Name] = true

				// Record as failed result
				p.resultsMu.Lock()
				p.results = append(p.results, types.TaskResult{
					ServiceName: svc.DisplayName(),
					StepName:    taskByDisplay[svc.DisplayName()].step.Name,
					Success:     false,
					Error:       fmt.Errorf("skipped: dependency %s failed", failedDep),
				})
				p.resultsMu.Unlock()
				continue
			}

			if t, ok := taskByDisplay[svc.DisplayName()]; ok {
				waveTasks = append(waveTasks, t)
			}
		}

		if len(waveTasks) == 0 {
			continue
		}

		// Execute this wave's tasks in parallel
		waveErr := p.executeTasksParallel(waveTasks)
		if waveErr != nil {
			// Record which services failed in this wave
			for _, t := range waveTasks {
				for _, result := range p.results {
					if result.ServiceName == t.service.DisplayName() && !result.Success {
						failedServices[t.service.Name] = true
					}
				}
			}
		}
	}

	// Build aggregate error
	var failedNames []string
	for name := range failedServices {
		failedNames = append(failedNames, name)
	}
	if len(failedNames) > 0 {
		return errors.New("step failed for: %s", strings.Join(failedNames, ", "))
	}

	return nil
}

// dryRunWithWaves shows dry-run output grouped by waves.
func (p *Pipeline) dryRunWithWaves(tasks []stepTask) error {
	services := make([]serviceinfo.ServiceInfo, len(tasks))
	for i, t := range tasks {
		services[i] = t.service
	}

	waves, err := serviceinfo.CalculateWaves(services)
	if err != nil {
		return err
	}

	if waves == nil {
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

	taskByDisplay := make(map[string]stepTask)
	for _, t := range tasks {
		taskByDisplay[t.service.DisplayName()] = t
	}

	for waveIdx, wave := range waves {
		p.output.PrintWaveHeader(waveIdx+1, len(waves))
		for _, svc := range wave {
			if t, ok := taskByDisplay[svc.DisplayName()]; ok {
				cmd := p.generateCommand(t.service, t.step)
				p.output.PrintDryRun(t.service.DisplayName(), t.step.Name, cmd)
				p.dryRunResults = append(p.dryRunResults, types.DryRunEntry{
					Service: t.service.DisplayName(),
					Step:    t.step.Name,
					Command: output.FormatCommand(cmd),
					Wave:    waveIdx + 1,
				})
			}
		}
	}

	return nil
}
