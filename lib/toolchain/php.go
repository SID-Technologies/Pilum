package toolchain

import (
	"os/exec"
	"regexp"
	"strconv"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// composerVersion is a variable so tests don't need Composer installed.
var composerVersion = func(root string) (major, minor int, ok bool) {
	cmd := exec.Command("composer", "--version", "--no-ansi")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, false
	}
	return parseComposerVersion(string(out))
}

var composerVersionRe = regexp.MustCompile(`Composer (?:version )?(\d+)\.(\d+)`)

func parseComposerVersion(out string) (major, minor int, ok bool) {
	m := composerVersionRe.FindStringSubmatch(out)
	if m == nil {
		return 0, 0, false
	}
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, false
	}
	minor, err = strconv.Atoi(m[2])
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}

// php warms Composer's cache with `composer install --download-only`, which
// only primes the cache and installs nothing (Composer 2.5+).
type php struct{}

func (php) Name() string { return "composer" }

func (php) Languages() []string { return []string{"php"} }

func (php) Manifests() []string { return []string{"composer.json"} }

// Root is the nearest composer.lock: without one there's nothing pinned to
// download.
func (php) Root(svcDir, bound string) (string, bool) {
	return nearest(ancestors(svcDir, bound), "composer.lock")
}

func (php) Variant(serviceinfo.ServiceInfo) string { return "" }

func (php) Plan(g Group) (Plan, bool) {
	_, err := lookPath("composer")
	if err != nil {
		return Plan{}, false
	}
	major, minor, ok := composerVersion(g.Root)
	if !ok || major < 2 || (major == 2 && minor < 5) {
		return Plan{}, false
	}
	return Plan{
		Cmd:  "composer install --download-only --no-scripts --no-plugins --no-interaction --quiet",
		Dir:  g.Root,
		Tool: "composer",
	}, true
}
