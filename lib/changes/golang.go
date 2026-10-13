package changes

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sid-technologies/pilum/lib/output"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/toolchain"
)

// goDeps is the set of directories inside the repository whose packages a
// Go service compiles.
type goDeps struct {
	dirs    map[string]struct{}
	unknown bool // go list failed, so any change may affect the service
}

// goEnvKeys are the build env vars that change which files Go compiles.
var goEnvKeys = []string{"GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS", "GOEXPERIMENT"}

// goListFormat prints one package per line: import path, directory, whether
// it is only a dependency, and its transitive imports.
const goListFormat = "{{.ImportPath}}\t{{.Dir}}\t{{.DepOnly}}\t{{join .Deps \" \"}}"

// goDependencyDirs runs `go list -deps` once per module root and build
// configuration, and returns each Go service's dependency directories, keyed
// by DisplayName.
func goDependencyDirs(services []serviceinfo.ServiceInfo, repoRoot, base string) map[string]goDeps {
	type group struct {
		root     string
		env      []string
		flags    []string
		services []serviceinfo.ServiceInfo
	}
	groups := make(map[string]*group)
	var order []string

	for _, svc := range services {
		svcDir := absPath(base, svc.Path)
		tc, root, ok := toolchain.Resolve(svcDir, repoRoot, svc.BuildConfig.Language)
		if !ok || tc.Name() != "go" {
			continue
		}

		env := goEnv(svc)
		flags := toolchain.GoBuildFlags(svc)
		key := root + "\x00" + strings.Join(env, " ") + "\x00" + strings.Join(flags, " ")
		g, exists := groups[key]
		if !exists {
			g = &group{root: root, env: env, flags: flags}
			groups[key] = g
			order = append(order, key)
		}
		g.services = append(g.services, svc)
	}

	result := make(map[string]goDeps)
	for _, key := range order {
		g := groups[key]
		deps, err := listGoDeps(g.root, g.env, g.flags, g.services, repoRoot, base)
		if err != nil {
			output.Warning("go list failed in %s (%v); treating its services as changed", g.root, err)
			for _, svc := range g.services {
				result[svc.DisplayName()] = goDeps{unknown: true}
			}
			continue
		}
		for name, d := range deps {
			result[name] = d
		}
	}

	return result
}

func goEnv(svc serviceinfo.ServiceInfo) []string {
	var env []string
	for _, ev := range svc.BuildConfig.EnvVars {
		if slices.Contains(goEnvKeys, ev.Name) {
			env = append(env, ev.Name+"="+ev.Value)
		}
	}
	slices.Sort(env)
	return env
}

func listGoDeps(root string, env, flags []string, services []serviceinfo.ServiceInfo, repoRoot, base string) (map[string]goDeps, error) {
	args := []string{"list", "-e", "-deps", "-f", goListFormat}
	args = append(args, flags...)

	svcDirs := make(map[string]string, len(services)) // abs dir -> DisplayName
	for _, svc := range services {
		dir := absPath(base, svc.Path)
		rel, err := filepath.Rel(root, dir)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		svcDirs[dir] = svc.DisplayName()
		args = append(args, "./"+filepath.ToSlash(filepath.Join(rel, "...")))
	}

	// #nosec G204 -- fixed binary; arguments are package patterns and build flags from pilum.yaml.
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	// Environ sets PWD to Dir; an inherited PWD would make go list report
	// directories through whatever symlinked path pilum was started from.
	cmd.Env = append(cmd.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err //nolint:wrapcheck // reported with the root by the caller
	}

	dirOf := make(map[string]string)
	type pkg struct {
		dir  string
		deps []string
	}
	var top []pkg

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "\t", 4)
		if len(fields) < 4 {
			continue
		}
		importPath, dir, depOnly, deps := fields[0], fields[1], fields[2], fields[3]
		dirOf[importPath] = dir
		if depOnly != "true" {
			top = append(top, pkg{dir: dir, deps: strings.Fields(deps)})
		}
	}

	result := make(map[string]goDeps, len(svcDirs))
	add := func(name, dir string) {
		if dir == "" || !within(dir, repoRoot) {
			return
		}
		d, ok := result[name]
		if !ok {
			d = goDeps{dirs: make(map[string]struct{})}
			result[name] = d
		}
		d.dirs[dir] = struct{}{}
	}

	for _, p := range top {
		for svcDir, name := range svcDirs {
			if !within(p.dir, svcDir) {
				continue
			}
			add(name, p.dir)
			for _, dep := range p.deps {
				add(name, dirOf[dep])
			}
		}
	}

	return result, nil
}
