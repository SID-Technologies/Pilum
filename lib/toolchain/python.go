package toolchain

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sid-technologies/pilum/lib/output"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// pythonLocks identify the tool, in the order they're checked when a
// directory holds more than one.
var pythonLocks = []string{"uv.lock", "poetry.lock", "requirements.txt"}

// pythonSkipDirs are never searched for manifests.
var pythonSkipDirs = []string{".venv", "venv", "__pycache__", "node_modules", ".git", "dist", "build", ".tox", ".mypy_cache"}

// python warms uv's or pip's cache. Python has nothing to compile ahead, but
// every service downloading the same wheels is the same work repeated. Both
// tools write into the project when they install (.venv, or wherever pip
// download points), so they run in a temporary copy of the manifests.
// Poetry has no download-only mode, so it doesn't warm automatically.
type python struct{}

func (python) Name() string { return "python" }

func (python) Languages() []string { return []string{"python"} }

func (python) Manifests() []string { return []string{"pyproject.toml", "requirements.txt", "setup.py"} }

// Root is the nearest directory with a lockfile, or a requirements.txt.
func (python) Root(svcDir, bound string) (string, bool) {
	return nearest(ancestors(svcDir, bound), pythonLocks...)
}

func (python) Variant(serviceinfo.ServiceInfo) string { return "" }

func (python) Plan(g Group) (Plan, bool) {
	switch {
	case exists(g.Root, "uv.lock"):
		return uvPlan(g.Root)
	case exists(g.Root, "poetry.lock"):
		output.Debugf("Not warming %s: Poetry has no download-only install; set build.warm to warm it", g.Root)
		return Plan{}, false
	case exists(g.Root, "requirements.txt"):
		return pipPlan(g.Root)
	default:
		return Plan{}, false
	}
}

// uvPlan syncs the locked dependencies, without the workspace's own packages,
// into a throwaway .venv in the copy: the downloads stay in uv's cache.
func uvPlan(root string) (Plan, bool) {
	_, err := lookPath("uv")
	if err != nil {
		return Plan{}, false
	}
	files, err := findFiles(root, pythonSkipDirs, func(_, name string) bool { return name == "pyproject.toml" })
	if err != nil {
		return Plan{}, false
	}
	return Plan{
		Cmd:     "uv sync --frozen --no-install-workspace --quiet",
		Dir:     root,
		Isolate: append([]string{"uv.lock", "uv.toml", ".python-version"}, files...),
		Tool:    "uv",
	}, true
}

// pipPlan downloads the requirements into the copy, which fills pip's cache.
// Every requirements and constraints file comes along, for -r and -c.
func pipPlan(root string) (Plan, bool) {
	pip, ok := pipCommand()
	if !ok {
		return Plan{}, false
	}
	files, err := findFiles(root, pythonSkipDirs, func(_, name string) bool {
		return strings.HasSuffix(name, ".txt") && (strings.HasPrefix(name, "requirements") || strings.HasPrefix(name, "constraints"))
	})
	if err != nil {
		return Plan{}, false
	}
	return Plan{
		Cmd:     pip + " download --quiet -r requirements.txt -d .pilum-download",
		Dir:     root,
		Isolate: files,
		Tool:    "pip",
	}, true
}

// pipCommand finds pip as installed: pip, pip3, or the interpreter's module.
func pipCommand() (string, bool) {
	for _, name := range []string{"pip", "pip3"} {
		_, err := lookPath(name)
		if err == nil {
			return name, true
		}
	}
	_, err := lookPath("python3")
	if err == nil {
		return "python3 -m pip", true
	}
	return "", false
}

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
