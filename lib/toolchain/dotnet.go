package toolchain

import (
	"path/filepath"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

var (
	dotnetSolutions = []string{"*.sln", "*.slnx"}
	dotnetProjects  = []string{"*.csproj", "*.fsproj", "*.vbproj"}
)

// dotnet restores NuGet packages before the builds. Restore writes obj/ in
// each project, but that's exactly what every build does first, and running
// it once avoids parallel restores racing over projects they share.
type dotnet struct{}

func (dotnet) Name() string { return "dotnet" }

func (dotnet) Languages() []string { return []string{"dotnet"} }

func (dotnet) Manifests() []string { return nil }

// Root is the nearest directory with a solution, else with a project.
func (dotnet) Root(svcDir, bound string) (string, bool) {
	dirs := ancestors(svcDir, bound)
	root, ok := nearestGlob(dirs, dotnetSolutions...)
	if ok {
		return root, true
	}
	return nearestGlob(dirs, dotnetProjects...)
}

// Owns reports a project file in the service directory, for services that
// declare no language.
func (dotnet) Owns(svcDir string) bool {
	_, ok := nearestGlob([]string{svcDir}, dotnetProjects...)
	return ok
}

func (dotnet) Variant(serviceinfo.ServiceInfo) string { return "" }

// Plan restores the one solution at the root, or else each service's project
// in turn: dotnet restore refuses to guess between several solutions.
func (dotnet) Plan(g Group) (Plan, bool) {
	_, err := lookPath("dotnet")
	if err != nil {
		return Plan{}, false
	}
	solutions := globAll(g.Root, dotnetSolutions...)
	if len(solutions) == 1 {
		return Plan{Cmd: "dotnet restore " + shellArg(filepath.Base(solutions[0])), Dir: g.Root, Tool: "dotnet"}, true
	}
	var cmds []string
	for _, dir := range g.Dirs {
		rel, err := filepath.Rel(g.Root, dir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return Plan{}, false
		}
		cmds = append(cmds, "dotnet restore "+shellArg(rel))
	}
	return Plan{Cmd: strings.Join(cmds, " && "), Dir: g.Root, Tool: "dotnet"}, true
}

// nearestGlob returns the first of dirs with a file matching any pattern.
func nearestGlob(dirs []string, patterns ...string) (string, bool) {
	for _, dir := range dirs {
		if len(globAll(dir, patterns...)) > 0 {
			return dir, true
		}
	}
	return "", false
}

func globAll(dir string, patterns ...string) []string {
	var matches []string
	for _, pattern := range patterns {
		found, _ := filepath.Glob(filepath.Join(dir, pattern))
		matches = append(matches, found...)
	}
	return matches
}
