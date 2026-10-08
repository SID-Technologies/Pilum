package toolchain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestRustFetch(t *testing.T) {
	t.Parallel()

	root := tree(t, "Cargo.toml", "Cargo.lock", "crates/api/Cargo.toml")
	tc, got, ok := Resolve(filepath.Join(root, "crates/api"), root, "rust")
	require.True(t, ok)
	require.Equal(t, "cargo", tc.Name())
	require.Equal(t, root, got)

	svc := serviceinfo.ServiceInfo{BuildConfig: serviceinfo.BuildConfig{Language: "rust", Flags: []serviceinfo.BuildFlag{
		{Name: "target", Values: []string{"x86_64-unknown-linux-musl"}},
	}}}
	plan, ok := tc.Plan(Group{Root: got, Service: svc})
	require.True(t, ok)
	require.Equal(t, "cargo fetch --locked --target x86_64-unknown-linux-musl", plan.Cmd)
	require.Empty(t, plan.Isolate, "cargo fetch writes nothing to the project")
	require.Equal(t, "x86_64-unknown-linux-musl", tc.Variant(svc), "crates are fetched per target")
}

func TestGradleRootAndPlan(t *testing.T) {
	t.Parallel()

	root := tree(t, "settings.gradle.kts", "gradlew", "services/api/build.gradle.kts")
	tc, got, ok := Resolve(filepath.Join(root, "services/api"), root, "kotlin")
	require.True(t, ok)
	require.Equal(t, "gradle", tc.Name())
	require.Equal(t, root, got)

	plan, ok := tc.Plan(Group{Root: got})
	require.True(t, ok)
	require.True(t, strings.HasPrefix(plan.Cmd, "./gradlew -q --no-configuration-cache --init-script "), plan.Cmd)
	require.True(t, strings.HasSuffix(plan.Cmd, " pilumResolveAll"), plan.Cmd)

	script := strings.Fields(plan.Cmd)[4]
	data, err := os.ReadFile(script)
	require.NoError(t, err)
	require.Equal(t, gradleResolveScript, string(data))
}

func TestGradleWithoutWrapperUsesInstalledGradle(t *testing.T) {
	t.Parallel()

	root := tree(t, "build.gradle")
	plan, ok := gradle{}.Plan(Group{Root: root})
	require.True(t, ok)
	require.True(t, strings.HasPrefix(plan.Cmd, "gradle "), plan.Cmd)
}

// pomTree writes poms whose <modules> are given, keyed by directory.
func pomTree(t *testing.T, poms map[string][]string, extra ...string) string {
	t.Helper()
	root := tree(t, extra...)
	for dir, modules := range poms {
		var b strings.Builder
		b.WriteString("<project><modules>")
		for _, m := range modules {
			b.WriteString("<module>" + m + "</module>")
		}
		b.WriteString("</modules></project>")
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, dir, "pom.xml"), []byte(b.String()), 0o600))
	}
	return root
}

func TestMavenRootFollowsModules(t *testing.T) {
	t.Parallel()

	// The aggregator lists services/api directly: services/ has no pom.
	root := pomTree(t, map[string][]string{".": {"services/api", "services/worker"}, "services/api": nil}, "mvnw")
	tc, got, ok := Resolve(filepath.Join(root, "services/api"), root, "java")
	require.True(t, ok)
	require.Equal(t, "maven", tc.Name())
	require.Equal(t, root, got)

	plan, ok := tc.Plan(Group{Root: got})
	require.True(t, ok)
	require.Equal(t, "./mvnw -B -q dependency:go-offline", plan.Cmd)
}

func TestMavenRootNested(t *testing.T) {
	t.Parallel()

	root := pomTree(t, map[string][]string{".": {"services"}, "services": {"api"}, "services/api": nil})
	got, ok := maven{}.Root(filepath.Join(root, "services/api"), root)
	require.True(t, ok)
	require.Equal(t, root, got)
}

func TestMavenRootIgnoresAnUnrelatedPom(t *testing.T) {
	t.Parallel()

	// The outer pom doesn't list libs/api, so it doesn't build it.
	root := pomTree(t, map[string][]string{".": {"other"}, "libs/api": nil})
	got, ok := maven{}.Root(filepath.Join(root, "libs/api"), root)
	require.True(t, ok)
	require.Equal(t, filepath.Join(root, "libs/api"), got)
}

func TestGradleWinsOverMaven(t *testing.T) {
	t.Parallel()

	root := tree(t, "settings.gradle", "pom.xml")
	tc, _, ok := Resolve(root, root, "java")
	require.True(t, ok)
	require.Equal(t, "gradle", tc.Name())
}

func TestDotnetSolution(t *testing.T) {
	t.Parallel()

	root := tree(t, "Platform.sln", "src/Api/Api.csproj", "src/Worker/Worker.csproj")
	tc, got, ok := Resolve(filepath.Join(root, "src/Api"), root, "csharp")
	require.True(t, ok)
	require.Equal(t, "dotnet", tc.Name())
	require.Equal(t, root, got)

	plan, ok := tc.Plan(Group{Root: got, Dirs: []string{filepath.Join(root, "src/Api"), filepath.Join(root, "src/Worker")}})
	require.True(t, ok)
	require.Equal(t, "dotnet restore Platform.sln", plan.Cmd)
}

func TestDotnetSeveralSolutionsRestoresEachProject(t *testing.T) {
	t.Parallel()

	root := tree(t, "A.sln", "B.slnx", "src/Api/Api.csproj", "src/Worker/Worker.fsproj")
	plan, ok := dotnet{}.Plan(Group{Root: root, Dirs: []string{filepath.Join(root, "src/Api"), filepath.Join(root, "src/Worker")}})
	require.True(t, ok)
	require.Equal(t, "dotnet restore src/Api && dotnet restore src/Worker", plan.Cmd)
}

func TestDotnetWithoutLanguageNeedsAProjectFile(t *testing.T) {
	t.Parallel()

	root := tree(t, "Platform.sln", "src/Api/Api.csproj", "docs/readme.md")
	tc, _, ok := Resolve(filepath.Join(root, "src/Api"), root, "")
	require.True(t, ok)
	require.Equal(t, "dotnet", tc.Name())

	_, _, ok = Resolve(filepath.Join(root, "docs"), root, "")
	require.False(t, ok)
}
