package toolchain

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// nodeLockfiles identify the package manager, in the order they're checked
// when a directory holds more than one.
var nodeLockfiles = []string{"pnpm-lock.yaml", "bun.lock", "bun.lockb", "yarn.lock", "package-lock.json"}

// nodeConfigs are package manager settings that change what an install
// resolves or where it fetches from. Missing ones are skipped.
var nodeConfigs = []string{".npmrc", "pnpm-workspace.yaml", ".pnpmfile.cjs", ".yarnrc", ".yarnrc.yml", "bunfig.toml"}

// nodeCopyDirs hold files an install reads beyond manifests: Yarn Berry's
// pinned release and plugins, and dependency patches.
var nodeCopyDirs = []string{".yarn/releases", ".yarn/plugins", "patches", ".yarn/patches"}

// nodeSkipDirs are never searched for workspace manifests.
var nodeSkipDirs = []string{"node_modules", ".git", "dist", "build", ".next", ".turbo", "target", "vendor"}

// lookPath and pnpmStorePath are variables so tests don't depend on which
// package managers are installed.
var (
	lookPath      = exec.LookPath
	pnpmStorePath = func(root string) (string, error) {
		cmd := exec.Command("pnpm", "store", "path")
		cmd.Dir = root
		out, err := cmd.Output()
		return strings.TrimSpace(string(out)), err
	}
)

// node warms a package manager's global cache. None of npm, Yarn or Bun can
// download without also installing into the project, and pnpm fetch writes
// node_modules/.pnpm, so every install runs in a temporary copy of the
// lockfile and manifests: the cache fills and the project isn't touched.
type node struct{}

func (node) Name() string { return "node" }

func (node) Languages() []string { return []string{"node", "bun"} }

func (node) Manifests() []string { return []string{"package.json"} }

// Root is the nearest directory with a lockfile: a workspace has one, at its
// root.
func (node) Root(svcDir, bound string) (string, bool) {
	return nearest(ancestors(svcDir, bound), nodeLockfiles...)
}

func (node) Variant(serviceinfo.ServiceInfo) string { return "" }

func (node) Plan(g Group) (Plan, bool) {
	tool, cmd, ok := nodeInstall(g.Root)
	if !ok {
		return Plan{}, false
	}
	// A package manager that isn't installed has nothing to warm.
	_, err := lookPath(tool)
	if err != nil {
		return Plan{}, false
	}
	// pnpm keeps a store per filesystem, and a temp dir may be on another one
	// (tmpfs on Linux CI): fetch into the store the project's installs use.
	if tool == "pnpm" {
		store, err := pnpmStorePath(g.Root)
		if err != nil || store == "" {
			return Plan{}, false
		}
		cmd += " --store-dir " + shellArg(store)
	}
	files, err := nodeFiles(g.Root)
	if err != nil {
		return Plan{}, false
	}
	return Plan{Cmd: cmd, Dir: g.Root, Isolate: files, Tool: tool}, true
}

// nodeInstall picks the package manager from the lockfile at root and the
// install that fills its cache without running package scripts.
func nodeInstall(root string) (tool, cmd string, ok bool) {
	for _, lock := range nodeLockfiles {
		_, err := os.Stat(filepath.Join(root, lock))
		if err != nil {
			continue
		}
		switch lock {
		case "pnpm-lock.yaml":
			return "pnpm", "pnpm fetch", true
		case "bun.lock", "bun.lockb":
			return "bun", "bun install --frozen-lockfile --ignore-scripts", true
		case "yarn.lock":
			if isYarnBerry(root) {
				return "yarn", "yarn install --immutable --mode=skip-build", true
			}
			return "yarn", "yarn install --frozen-lockfile --ignore-scripts --non-interactive", true
		default:
			return "npm", "npm ci --ignore-scripts --no-audit --no-fund", true
		}
	}
	return "", "", false
}

// isYarnBerry tells Yarn 2+ from Yarn 1, which share yarn.lock but not flags.
func isYarnBerry(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".yarnrc.yml"))
	if err == nil {
		return true
	}
	lock, err := os.ReadFile(filepath.Join(root, "yarn.lock"))
	return err == nil && bytes.Contains(lock, []byte("__metadata:"))
}

// nodeFiles lists what an install reads, relative to root: lockfiles,
// configs, every package.json (workspace members included) and the files in
// nodeCopyDirs.
func nodeFiles(root string) ([]string, error) {
	files := append(slices.Clone(nodeLockfiles), nodeConfigs...)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && slices.Contains(nodeSkipDirs, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "package.json" || inCopyDir(rel) {
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

func inCopyDir(rel string) bool {
	slashed := filepath.ToSlash(rel)
	for _, dir := range nodeCopyDirs {
		if strings.HasPrefix(slashed, dir+"/") {
			return true
		}
	}
	return false
}
