package toolchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// gradleResolveScript adds a task to every project that resolves each
// resolvable configuration. Gradle has no built-in command that downloads a
// build's dependencies; this is the usual workaround, injected as an init
// script so the project's build files aren't touched. Configurations Gradle
// only creates while running tasks are missed, so it's best effort.
const gradleResolveScript = `allprojects {
    tasks.register("pilumResolveAll") {
        doLast {
            configurations.findAll { it.canBeResolved }.each { c ->
                try {
                    c.resolve()
                } catch (Exception e) {
                    logger.info("pilum: skipped ${c.name}: ${e.message}")
                }
            }
        }
    }
}
`

// gradle warms Gradle's dependency cache. It runs in the project: Gradle's
// own .gradle/ and build/ directories are what the build writes first anyway.
type gradle struct{}

func (gradle) Name() string { return "gradle" }

func (gradle) Languages() []string { return []string{"java", "kotlin"} }

func (gradle) Manifests() []string {
	return []string{"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"}
}

// Root is the nearest settings file, which defines a multi-project build,
// else the nearest build file.
func (gradle) Root(svcDir, bound string) (string, bool) {
	dirs := ancestors(svcDir, bound)
	root, ok := nearest(dirs, "settings.gradle", "settings.gradle.kts")
	if ok {
		return root, true
	}
	return nearest(dirs, "build.gradle", "build.gradle.kts")
}

func (gradle) Variant(serviceinfo.ServiceInfo) string { return "" }

func (gradle) Plan(g Group) (Plan, bool) {
	launcher, ok := jvmLauncher(g.Root, "gradlew", "gradle")
	if !ok {
		return Plan{}, false
	}
	script, err := gradleInitScript()
	if err != nil {
		return Plan{}, false
	}
	// The task reads configurations as it runs, which the configuration
	// cache forbids.
	cmd := launcher + " -q --no-configuration-cache --init-script " + shellArg(script) + " pilumResolveAll"
	return Plan{Cmd: cmd, Dir: g.Root, Tool: "gradle"}, true
}

// gradleInitScript writes the init script to the temp directory once, named
// by its content, and returns its path.
func gradleInitScript() (string, error) {
	sum := sha256.Sum256([]byte(gradleResolveScript))
	path := filepath.Join(os.TempDir(), "pilum-resolve-"+hex.EncodeToString(sum[:6])+".gradle")
	_, err := os.Stat(path)
	if err == nil {
		return path, nil
	}
	return path, os.WriteFile(path, []byte(gradleResolveScript), 0o600)
}

// maven warms the local Maven repository with dependency:go-offline. The
// plugin writes only marker files under target/. It's known to miss some
// plugin dependencies, so it's best effort.
type maven struct{}

func (maven) Name() string { return "maven" }

func (maven) Languages() []string { return []string{"java", "kotlin"} }

func (maven) Manifests() []string { return []string{"pom.xml"} }

// Root is the outermost reactor that builds the service: starting from the
// nearest pom.xml, each pom above it that lists the current root among its
// <modules> (at any depth, e.g. services/api) becomes the new root.
func (maven) Root(svcDir, bound string) (string, bool) {
	root, ok := nearest(ancestors(svcDir, bound), "pom.xml")
	if !ok {
		return "", false
	}
	if root == bound {
		return root, true
	}
	for _, dir := range ancestors(filepath.Dir(root), bound) {
		if !exists(dir, "pom.xml") {
			continue
		}
		if !listsModule(dir, root) {
			break
		}
		root = dir
	}
	return root, true
}

// listsModule reports whether the pom.xml in dir declares module as one of
// its <modules>.
func listsModule(dir, module string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "pom.xml"))
	if err != nil {
		return false
	}
	var pom struct {
		Modules []string `xml:"modules>module"`
	}
	err = xml.Unmarshal(data, &pom)
	if err != nil {
		return false
	}
	for _, m := range pom.Modules {
		if filepath.Clean(filepath.Join(dir, strings.TrimSpace(m))) == module {
			return true
		}
	}
	return false
}

func (maven) Variant(serviceinfo.ServiceInfo) string { return "" }

func (maven) Plan(g Group) (Plan, bool) {
	launcher, ok := jvmLauncher(g.Root, "mvnw", "mvn")
	if !ok {
		return Plan{}, false
	}
	return Plan{Cmd: launcher + " -B -q dependency:go-offline", Dir: g.Root, Tool: "maven"}, true
}

// jvmLauncher prefers the project's wrapper, which pins the tool's version,
// over an installed tool.
func jvmLauncher(root, wrapper, tool string) (string, bool) {
	if exists(root, wrapper) {
		return "./" + wrapper, true
	}
	_, err := lookPath(tool)
	return tool, err == nil
}
