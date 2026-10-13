package changes_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sid-technologies/pilum/lib/changes"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

// A Go monorepo: api imports pkg/auth, worker imports pkg/queue, web is a
// pnpm app with its own lockfile. The base commit is tagged v1.
func newRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	files := map[string]string{
		"go.mod":                      "module example.com/mono\n\ngo 1.22\n",
		"pkg/auth/auth.go":            "package auth\n\nfunc Check() bool { return true }\n",
		"pkg/queue/queue.go":          "package queue\n\nfunc Push() {}\n",
		"services/api/main.go":        "package main\n\nimport \"example.com/mono/pkg/auth\"\n\nfunc main() { _ = auth.Check() }\n",
		"services/worker/main.go":     "package main\n\nimport \"example.com/mono/pkg/queue\"\n\nfunc main() { queue.Push() }\n",
		"services/web/package.json":   "{\"name\": \"web\"}\n",
		"services/web/pnpm-lock.yaml": "lockfileVersion: '9.0'\n",
		"_templates/golang-api.v1":    "FROM scratch\n",
		"_templates/react-web-app.v1": "FROM scratch\n",
		"README.md":                   "# mono\n",
	}
	for name, content := range files {
		write(t, dir, name, content)
	}

	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "add", ".")
	git(t, dir, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "base")
	git(t, dir, "tag", "v1")

	return dir
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func commit(t *testing.T, dir string) {
	t.Helper()
	git(t, dir, "add", "-A")
	git(t, dir, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "change")
}

func services() []serviceinfo.ServiceInfo {
	goBuild := serviceinfo.BuildConfig{Language: "go"}
	return []serviceinfo.ServiceInfo{
		{Name: "api", Path: "services/api", Template: "golang-api.v1", BuildConfig: goBuild},
		{Name: "worker", Path: "services/worker", Template: "golang-api.v1", BuildConfig: goBuild},
		{Name: "web", Path: "services/web", Template: "react-web-app.v1", BuildConfig: serviceinfo.BuildConfig{Language: "node"}},
	}
}

func names(svcs []serviceinfo.ServiceInfo) []string {
	out := make([]string, 0, len(svcs))
	for _, s := range svcs {
		out = append(out, s.Name)
	}
	return out
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name   string
		change func(t *testing.T, dir string)
		svcs   func() []serviceinfo.ServiceInfo
		want   []string
	}{
		{
			name:   "service's own directory",
			change: func(t *testing.T, dir string) { write(t, dir, "services/worker/extra.go", "package main\n") },
			want:   []string{"worker"},
		},
		{
			name: "shared Go package selects only its importers",
			change: func(t *testing.T, dir string) {
				write(t, dir, "pkg/auth/auth.go", "package auth\n\nfunc Check() bool { return false }\n")
			},
			want: []string{"api"},
		},
		{
			name:   "go.sum at the module root selects every Go service",
			change: func(t *testing.T, dir string) { write(t, dir, "go.sum", "") },
			want:   []string{"api", "worker"},
		},
		{
			name:   "Dockerfile template",
			change: func(t *testing.T, dir string) { write(t, dir, "_templates/react-web-app.v1", "FROM nginx\n") },
			want:   []string{"web"},
		},
		{
			name: "lockfile at a node workspace root",
			change: func(t *testing.T, dir string) {
				write(t, dir, "services/web/pnpm-lock.yaml", "lockfileVersion: '9.1'\n")
			},
			want: []string{"web"},
		},
		{
			name:   "unrelated file selects nothing",
			change: func(t *testing.T, dir string) { write(t, dir, "README.md", "# changed\n") },
			want:   nil,
		},
		{
			name:   "watch paths",
			change: func(t *testing.T, dir string) { write(t, dir, "README.md", "# changed\n") },
			svcs: func() []serviceinfo.ServiceInfo {
				s := services()
				s[2].Watch = []string{"README.md"}
				return s
			},
			want: []string{"web"},
		},
		{
			name:   "dependents of a changed service",
			change: func(t *testing.T, dir string) { write(t, dir, "services/worker/extra.go", "package main\n") },
			svcs: func() []serviceinfo.ServiceInfo {
				s := services()
				s[2].DependsOn = []string{"worker"}
				return s
			},
			want: []string{"worker", "web"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := newRepo(t)
			tt.change(t, dir)
			commit(t, dir)
			t.Chdir(dir)

			svcs := services()
			if tt.svcs != nil {
				svcs = tt.svcs()
			}

			got, err := changes.Filter(svcs, changes.Options{Since: "v1"})
			require.NoError(t, err)
			if tt.want == nil {
				require.Empty(t, got)
				return
			}
			require.Equal(t, tt.want, names(got))
		})
	}
}

// A ref the clone doesn't have (a shallow CI checkout) deploys everything
// instead of failing or deploying nothing.
func TestFilterUnknownRefSelectsAll(t *testing.T) {
	dir := newRepo(t)
	t.Chdir(dir)

	got, err := changes.Filter(services(), changes.Options{Since: "v0-not-fetched"})
	require.NoError(t, err)
	require.Equal(t, []string{"api", "worker", "web"}, names(got))
}

// Without --since, the default branch is used; on a feature branch only
// its own changes count.
func TestFilterDefaultBranch(t *testing.T) {
	dir := newRepo(t)
	git(t, dir, "checkout", "-q", "-b", "feature")
	write(t, dir, "pkg/queue/queue.go", "package queue\n\nfunc Push() { _ = 1 }\n")
	commit(t, dir)
	t.Chdir(dir)

	got, err := changes.Filter(services(), changes.Options{})
	require.NoError(t, err)
	require.Equal(t, []string{"worker"}, names(got))
}
