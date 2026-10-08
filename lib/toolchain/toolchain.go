// Package toolchain detects the build tool a service uses and plans the one
// command that warms its shared cache before parallel builds start.
//
// A toolchain warms a group automatically only when its command can be
// derived without guessing, and running it either leaves the project tree
// untouched or is an idempotent first step of what the build does anyway.
// Anything else stays opt-in through build.warm.
package toolchain

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/shellutil"
	"github.com/sid-technologies/pilum/lib/templates"
)

// Plan is a warm command for one group of services.
type Plan struct {
	// Cmd runs through sh -c, like build.cmd, so flag quoting matches the build.
	Cmd string
	// Dir is the absolute directory Cmd runs in.
	Dir string
	// Env is added to the services' shared build env.
	Env map[string]string
	// Isolate lists files, relative to Dir, that are copied into a temporary
	// directory for Cmd to run in instead, for tools whose only way to fill
	// their cache also writes into the project.
	Isolate []string
	// Tool names the group in output when it's more specific than the
	// toolchain, e.g. "pnpm" for node.
	Tool string
}

// Group is the services that share a workspace and warm together.
type Group struct {
	Root string   // absolute workspace root
	Dirs []string // absolute service directories
	// Service represents the group: members share its build env and Variant.
	Service serviceinfo.ServiceInfo
}

// Toolchain is one build tool (go, pnpm, cargo, ...).
type Toolchain interface {
	Name() string
	// Languages are the canonical build.language values the tool serves.
	Languages() []string
	// Manifests identify the tool in a service directory that declares no
	// language, e.g. go.mod.
	Manifests() []string
	// Root returns the workspace root for a service directory, searching no
	// higher than bound.
	Root(svcDir, bound string) (string, bool)
	// Variant is the part of a service's build config, beyond its env, that
	// changes what lands in the cache; services warm together only when it
	// matches.
	Variant(svc serviceinfo.ServiceInfo) string
	// Plan returns the warm command for a group, or false when there is
	// nothing safe to run.
	Plan(g Group) (Plan, bool)
}

var registry = []Toolchain{golang{}, node{}}

// Canonical normalizes a build.language value, e.g. "nodejs" to "node".
func Canonical(language string) string {
	return templates.Canonical(language)
}

// Resolve finds the toolchain and workspace root for a service directory. A
// declared language picks the candidates. Without one, a tool counts only if
// its manifest is in the service directory itself: a Node app inside a Go
// monorepo has a go.mod above it, which says nothing about how it builds. An
// unknown language resolves to nothing, so nothing warms by guesswork.
func Resolve(svcDir, bound, language string) (Toolchain, string, bool) {
	lang := Canonical(language)
	for _, tc := range registry {
		if lang != "" && !slices.Contains(tc.Languages(), lang) {
			continue
		}
		_, owns := nearest([]string{svcDir}, tc.Manifests()...)
		if lang == "" && !owns {
			continue
		}
		root, ok := tc.Root(svcDir, bound)
		if ok {
			return tc, root, true
		}
	}
	return nil, "", false
}

// ancestors lists dir and its parents up to and including bound. When dir is
// outside bound, the walk ends at the filesystem root.
func ancestors(dir, bound string) []string {
	var dirs []string
	for {
		dirs = append(dirs, dir)
		parent := filepath.Dir(dir)
		if dir == bound || parent == dir {
			return dirs
		}
		dir = parent
	}
}

// nearest returns the first of dirs that contains any of names.
func nearest(dirs []string, names ...string) (string, bool) {
	for _, dir := range dirs {
		for _, name := range names {
			info, err := os.Stat(filepath.Join(dir, name))
			if err == nil && !info.IsDir() {
				return dir, true
			}
		}
	}
	return "", false
}

var plainArg = regexp.MustCompile(`^[A-Za-z0-9_./=:@%+-]+$`)

// shellArg quotes s for sh only when it needs it, so commands stay readable
// in dry runs.
func shellArg(s string) string {
	if plainArg.MatchString(s) {
		return s
	}
	return shellutil.Quote(s)
}
