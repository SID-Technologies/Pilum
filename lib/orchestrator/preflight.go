package orchestrator

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/lib/sysinfo"
)

const dockerCheckTimeout = 10 * time.Second

// checkDocker is swapped out in tests so they don't depend on a local daemon.
var checkDocker = sysinfo.CheckDocker

// preflight fails fast on missing prerequisites before any step starts, so a
// stopped Docker daemon surfaces as one clear error instead of N parallel
// build failures after the binary builds have already burned CPU. It also
// records the daemon's resources so docker steps are sized against them.
func (p *Pipeline) preflight(maxSteps int) error {
	if p.options.DryRun || !p.requiresDocker(maxSteps) {
		return nil
	}

	res, err := checkDocker(dockerCheckTimeout)
	if err != nil {
		return err
	}
	p.dockerResources = res
	return nil
}

// requiresDocker reports whether any step that will run invokes the docker CLI.
func (p *Pipeline) requiresDocker(maxSteps int) bool {
	for _, svc := range p.services {
		rec, exists := p.getRecipeForService(svc)
		if !exists {
			continue
		}
		for stepIdx := 0; stepIdx < maxSteps && stepIdx < len(rec.Steps); stepIdx++ {
			step := &rec.Steps[stepIdx]
			if p.shouldSkipStep(step) {
				continue
			}
			if invokesDocker(p.generateCommand(svc, step)) {
				return true
			}
		}
	}
	return false
}

// invokesDocker reports whether a generated command's executable is docker.
func invokesDocker(cmd any) bool {
	var first string
	switch v := cmd.(type) {
	case string:
		if fields := strings.Fields(v); len(fields) > 0 {
			first = fields[0]
		}
	case []string:
		if len(v) > 0 {
			first = v[0]
		}
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				first = s
			}
		}
	}
	return filepath.Base(first) == "docker"
}
