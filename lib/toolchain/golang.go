package toolchain

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// goCompileFlags are the build flags that change compiled packages, and so
// their cache keys. Link-only flags such as ldflags don't, and are left out.
var goCompileFlags = []string{"asmflags", "gcflags", "mod", "race", "tags", "trimpath"}

// golang warms Go's build cache by compiling every service package of a
// module in one go build. Go's caches are safe to share, but parallel builds
// don't dedupe work in flight: each would compile the shared packages itself.
// One build first leaves them cached, and it writes nothing to the project.
type golang struct{}

func (golang) Name() string { return "go" }

func (golang) Languages() []string { return []string{"go"} }

func (golang) Manifests() []string { return []string{"go.mod"} }

// Root is the nearest go.work, which builds its modules as one, else the
// nearest go.mod.
func (golang) Root(svcDir, bound string) (string, bool) {
	dirs := ancestors(svcDir, bound)
	root, ok := nearest(dirs, "go.work")
	if ok {
		return root, true
	}
	return nearest(dirs, "go.mod")
}

func (golang) Variant(svc serviceinfo.ServiceInfo) string {
	return strings.Join(goFlagArgs(svc), " ")
}

func (golang) Plan(g Group) (Plan, bool) {
	var pkgs []string
	for _, dir := range g.Dirs {
		rel, err := filepath.Rel(g.Root, dir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return Plan{}, false
		}
		pkg := "./" + filepath.ToSlash(rel)
		if rel == "." {
			pkg = "."
		}
		if !slices.Contains(pkgs, pkg) {
			pkgs = append(pkgs, pkg)
		}
	}
	if len(pkgs) == 0 {
		return Plan{}, false
	}

	args := append([]string{"go", "build"}, goFlagArgs(g.Service)...)
	// go build discards what it builds for several packages, but writes a
	// binary for a single main package.
	if len(pkgs) == 1 {
		args = append(args, "-o", os.DevNull)
	}
	for _, pkg := range pkgs {
		args = append(args, shellArg(pkg))
	}
	plan := Plan{Cmd: strings.Join(args, " "), Dir: g.Root}

	// With one package there's no per-service dependency list to compare.
	if len(pkgs) > 1 {
		check := append([]string{"go", "list", "-deps", "-f", shellArg(goListFormat)}, goFlagArgs(g.Service)...)
		for _, pkg := range pkgs {
			check = append(check, shellArg(pkg))
		}
		plan.Check = strings.Join(check, " ")
		plan.Needed = goNeedsWarm
	}
	return plan, true
}

// goListFormat prints one line per package: S or F for stale or fresh, R or
// D for a listed package or a dependency, the import path, and for listed
// packages their dependencies.
const goListFormat = `{{if .Stale}}S{{else}}F{{end}}{{if .DepOnly}}D{{else}}R{{end}} {{.ImportPath}}{{if not .DepOnly}} {{join .Deps " "}}{{end}}`

// goNeedsWarm reports whether any package that two or more of the listed
// service packages depend on still has to be compiled. Those are the only
// packages parallel builds would compile more than once; a service's own
// stale packages are its build's work either way. Main packages always
// read as stale, so they're not counted.
func goNeedsWarm(stdout string) (bool, string) {
	stale := make(map[string]bool)
	var roots [][]string
	for line := range strings.Lines(stdout) {
		fields := strings.Fields(line)
		if len(fields) < 2 || len(fields[0]) != 2 {
			continue
		}
		if fields[0][1] == 'R' {
			roots = append(roots, fields[2:])
			continue
		}
		if fields[0][0] == 'S' {
			stale[fields[1]] = true
		}
	}

	// No listed packages means the output isn't what we asked for; warm.
	if len(roots) == 0 {
		return true, "no package list"
	}

	users := make(map[string]int)
	for _, deps := range roots {
		for _, dep := range deps {
			if stale[dep] {
				users[dep]++
			}
		}
	}
	shared := 0
	for _, n := range users {
		if n >= 2 {
			shared++
		}
	}
	if shared == 0 {
		return false, "shared packages already cached"
	}
	return true, fmt.Sprintf("%d shared packages to compile", shared)
}

// goFlagArgs renders the service's compile-affecting flags the way its build
// command does.
func goFlagArgs(svc serviceinfo.ServiceInfo) []string {
	var args []string
	for _, flag := range svc.BuildConfig.Flags {
		if len(flag.Values) == 0 || !slices.Contains(goCompileFlags, flag.Name) {
			continue
		}
		args = append(args, flag.Arg())
	}
	return args
}

// GoBuildFlags returns the build flags that change which packages and files
// a Go service compiles, such as -tags.
func GoBuildFlags(svc serviceinfo.ServiceInfo) []string {
	return goFlagArgs(svc)
}
