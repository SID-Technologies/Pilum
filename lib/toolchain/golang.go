package toolchain

import (
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
	return Plan{Cmd: strings.Join(args, " "), Dir: g.Root}, true
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
