package toolchain

import (
	"slices"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// rust fetches a workspace's crates. Only the download warms safely: which
// features a dependency compiles with depends on which packages are in the
// build, so a warm compile of several packages can produce artifacts a
// single package's build won't reuse. Parallel cargo builds sharing a target
// directory already queue on its lock rather than duplicating work.
type rust struct{}

func (rust) Name() string { return "cargo" }

func (rust) Languages() []string { return []string{"rust"} }

func (rust) Manifests() []string { return []string{"Cargo.toml"} }

// Root is the nearest Cargo.lock, which a workspace keeps at its root.
func (rust) Root(svcDir, bound string) (string, bool) {
	return nearest(ancestors(svcDir, bound), "Cargo.lock")
}

// Variant is the build's target triple: crates are fetched per target.
func (rust) Variant(svc serviceinfo.ServiceInfo) string {
	return strings.Join(rustTargets(svc), ",")
}

// Plan runs cargo fetch --locked, which fails rather than write Cargo.lock.
func (rust) Plan(g Group) (Plan, bool) {
	_, err := lookPath("cargo")
	if err != nil {
		return Plan{}, false
	}
	args := []string{"cargo", "fetch", "--locked"}
	for _, target := range rustTargets(g.Service) {
		args = append(args, "--target", shellArg(target))
	}
	return Plan{Cmd: strings.Join(args, " "), Dir: g.Root, Tool: "cargo"}, true
}

func rustTargets(svc serviceinfo.ServiceInfo) []string {
	for _, flag := range svc.BuildConfig.Flags {
		if flag.Name == "target" {
			return slices.Clone(flag.Values)
		}
	}
	return nil
}
