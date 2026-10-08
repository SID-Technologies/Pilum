package toolchain

import (
	"github.com/sid-technologies/pilum/lib/output"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// ruby recognizes Bundler projects but doesn't warm them: Bundler has no
// download-only install (bundle cache writes vendor/cache into the project),
// so build.warm is the way to opt in.
type ruby struct{}

func (ruby) Name() string { return "bundler" }

func (ruby) Languages() []string { return []string{"ruby"} }

func (ruby) Manifests() []string { return []string{"Gemfile"} }

func (ruby) Root(svcDir, bound string) (string, bool) {
	return nearest(ancestors(svcDir, bound), "Gemfile.lock")
}

func (ruby) Variant(serviceinfo.ServiceInfo) string { return "" }

func (ruby) Plan(g Group) (Plan, bool) {
	output.Debugf("Not warming %s: Bundler has no download-only install; set build.warm to warm it", g.Root)
	return Plan{}, false
}
