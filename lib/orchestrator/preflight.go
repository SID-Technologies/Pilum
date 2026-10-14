package orchestrator

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/lib/output"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/sysinfo"
)

const dockerCheckTimeout = 10 * time.Second

// checkDocker and checkBuildxCache are swapped out in tests so they don't
// depend on a local daemon.
var (
	checkDocker      = sysinfo.CheckDocker
	checkBuildxCache = sysinfo.CheckBuildxCache
)

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
	p.checkDockerCache()
	return nil
}

// checkDockerCache turns docker_cache off for this run when the local builder
// can't export a registry cache, so the build still runs, uncached, instead
// of failing.
func (p *Pipeline) checkDockerCache() {
	usesCache := slices.ContainsFunc(p.services, func(svc serviceinfo.ServiceInfo) bool {
		return svc.BuildConfig.DockerCache
	})
	if !usesCache {
		return
	}

	supported, reason := checkBuildxCache(dockerCheckTimeout)
	if supported {
		return
	}

	output.Warning("docker_cache is off for this run: %s. To use it, switch to a docker-container builder "+
		"(`docker buildx create --use --driver docker-container`, or docker/setup-buildx-action in GitHub Actions)", reason)
	for i := range p.services {
		p.services[i].BuildConfig.DockerCache = false
	}
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
