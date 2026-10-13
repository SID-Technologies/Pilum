// Package changes picks the services a set of git changes affects, for
// --only-changed. A service is affected by a change to its own directory,
// to code it imports (Go), to its workspace's lockfile or manifest, to its
// Dockerfile template, or to a path it lists under watch:. Services that
// depend on an affected service (depends_on) are affected too.
//
// When git can't compare against the reference (a shallow clone, an unknown
// ref), every service is affected: deploying too much is recoverable,
// silently skipping a changed service is not.
package changes

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/git"
	"github.com/sid-technologies/pilum/lib/graph"
	"github.com/sid-technologies/pilum/lib/output"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/toolchain"
)

// DefaultTemplatePath is where Dockerfile templates live unless a command
// says otherwise.
const DefaultTemplatePath = "_templates"

// Options configures Filter.
type Options struct {
	Since        string // git ref to compare against; empty means the default branch
	TemplatePath string // Dockerfile template directory, relative to the working directory
}

// workspaceFiles are the manifests and lockfiles that change what every
// service in a toolchain workspace builds when they change at its root.
var workspaceFiles = []string{
	"go.mod", "go.sum", "go.work", "go.work.sum",
	"package.json", "package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "pnpm-workspace.yaml",
	"yarn.lock", ".yarnrc.yml", "bun.lock", "bun.lockb", "bunfig.toml", ".npmrc",
	"Cargo.toml", "Cargo.lock",
	"pyproject.toml", "uv.lock", "poetry.lock", "requirements.txt",
	"Gemfile", "Gemfile.lock", "composer.json", "composer.lock",
	"settings.gradle", "settings.gradle.kts", "build.gradle", "build.gradle.kts", "gradle.properties",
	"pom.xml", "Directory.Build.props", "Directory.Packages.props", "global.json",
}

// Filter returns the services affected by changes since opts.Since, in their
// original order, and prints why each was picked.
func Filter(services []serviceinfo.ServiceInfo, opts Options) ([]serviceinfo.ServiceInfo, error) {
	if !git.IsGitRepository() {
		output.Warning("Not a git repository, --only-changed has no effect")
		return services, nil
	}

	repoRoot, base, err := roots()
	if err != nil {
		output.Warning("Can't find the git repository root (%v); deploying all services", err)
		return services, nil
	}

	changed, ref, err := git.ChangedFilesSince(opts.Since)
	if err != nil {
		output.Warning("Can't compare against %s (%v); deploying all services. In CI, check out with fetch-depth: 0.", refLabel(ref, opts.Since), err)
		return services, nil
	}

	uncommitted, err := git.ChangedFilesUncommitted()
	if err != nil {
		output.Debugf("Could not get uncommitted changes: %v", err)
	}
	changed = append(changed, uncommitted...)

	if len(changed) == 0 {
		output.Info("No changes since %s", ref)
		return nil, nil
	}

	abs := make([]string, 0, len(changed))
	for _, f := range changed {
		abs = append(abs, filepath.Join(repoRoot, filepath.FromSlash(f)))
	}
	output.Debugf("Changed files since %s: %v", ref, changed)

	templatePath := opts.TemplatePath
	if templatePath == "" {
		templatePath = DefaultTemplatePath
	}

	reasons := reasonsFor(services, abs, repoRoot, base, templatePath)

	return selectAffected(services, reasons, ref), nil
}

func refLabel(resolved, requested string) string {
	if resolved != "" {
		return resolved
	}
	if requested != "" {
		return requested
	}
	return "the default branch"
}

// reasonsFor maps each directly affected service (by DisplayName) to the
// first changed path that affects it.
func reasonsFor(services []serviceinfo.ServiceInfo, changed []string, repoRoot, base, templatePath string) map[string]string {
	reasons := make(map[string]string)
	rel := func(p string) string {
		r, err := filepath.Rel(repoRoot, p)
		if err != nil {
			return p
		}
		return filepath.ToSlash(r)
	}
	note := func(svc serviceinfo.ServiceInfo, file string) {
		if _, done := reasons[svc.DisplayName()]; !done {
			reasons[svc.DisplayName()] = rel(file) + " changed"
		}
	}

	goSets := goDependencyDirs(services, repoRoot, base)

	for _, svc := range services {
		svcDir := absPath(base, svc.Path)
		watched := []string{svcDir}
		if svc.Template != "" {
			watched = append(watched, absPath(base, filepath.Join(templatePath, svc.Template)))
		}
		for _, w := range svc.Watch {
			watched = append(watched, absPath(base, w))
		}

		tc, root, ok := toolchain.Resolve(svcDir, repoRoot, svc.BuildConfig.Language)

		for _, file := range changed {
			if within(file, watched...) {
				note(svc, file)
				break
			}
			if ok && filepath.Dir(file) == root && isWorkspaceFile(file) {
				note(svc, file)
				break
			}
			deps, hasGo := goSets[svc.DisplayName()]
			if ok && tc.Name() == "go" && hasGo {
				if deps.unknown {
					reasons[svc.DisplayName()] = "Go dependencies unknown, assuming changed"
					break
				}
				if _, imported := deps.dirs[filepath.Dir(file)]; imported {
					note(svc, file)
					break
				}
			}
		}
	}

	return reasons
}

// selectAffected adds the dependents of each affected service and prints
// the selection.
func selectAffected(services []serviceinfo.ServiceInfo, reasons map[string]string, ref string) []serviceinfo.ServiceInfo {
	g := graph.New()
	direct := make(map[string]bool)
	for _, svc := range services {
		g.AddNode(svc.Name, svc.DependsOn)
		if _, ok := reasons[svc.DisplayName()]; ok {
			direct[svc.Name] = true
		}
	}
	affected := g.PropagateChanges(direct)

	var result []serviceinfo.ServiceInfo
	var lines []string
	for _, svc := range services {
		if !affected[svc.Name] {
			continue
		}
		result = append(result, svc)

		reason, ok := reasons[svc.DisplayName()]
		if !ok {
			reason = "depends on a changed service"
		}
		lines = append(lines, "  "+svc.DisplayName()+": "+reason)
	}

	if len(result) == 0 {
		output.Info("No services affected by changes since %s", ref)
		return nil
	}

	output.Info("%d of %d service(s) affected by changes since %s:", len(result), len(services), ref)
	slices.Sort(lines)
	for _, line := range lines {
		output.Dimmed("%s", line)
	}

	return result
}

// roots returns the repository root and the working directory with
// symlinks resolved, so paths from git, go list and pilum.yaml compare
// equal (on macOS, /var is a link to /private/var).
func roots() (repoRoot, base string, err error) {
	top, err := git.TopLevel()
	if err != nil {
		return "", "", err //nolint:wrapcheck // already wrapped by git
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get the working directory")
	}

	repoRoot, err = filepath.EvalSymlinks(top)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to resolve %s", top)
	}
	base, err = filepath.EvalSymlinks(wd)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to resolve %s", wd)
	}

	return repoRoot, base, nil
}

// absPath resolves a path from pilum.yaml against the working directory.
func absPath(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}

// within reports whether file is one of paths or inside one of them.
func within(file string, paths ...string) bool {
	for _, p := range paths {
		if file == p || strings.HasPrefix(file, p+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func isWorkspaceFile(file string) bool {
	base := filepath.Base(file)
	for _, name := range workspaceFiles {
		if base == name {
			return true
		}
	}
	return false
}
