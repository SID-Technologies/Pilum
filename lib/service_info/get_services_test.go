package serviceinfo_test

import (
	"os"
	"path/filepath"
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestFindServices(t *testing.T) {
	t.Parallel()

	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create service1/pilum.yaml
	svc1Dir := filepath.Join(tmpDir, "service1")
	require.NoError(t, os.MkdirAll(svc1Dir, 0755))
	svc1Content := `name: myservice
provider: gcp
region: us-central1
project: my-project
`
	require.NoError(t, os.WriteFile(filepath.Join(svc1Dir, "pilum.yaml"), []byte(svc1Content), 0644))

	// Create service2/pilum.yaml
	svc2Dir := filepath.Join(tmpDir, "service2")
	require.NoError(t, os.MkdirAll(svc2Dir, 0755))
	svc2Content := `name: api-service
provider: aws
region: us-east-1
`
	require.NoError(t, os.WriteFile(filepath.Join(svc2Dir, "pilum.yaml"), []byte(svc2Content), 0644))

	services, err := serviceinfo.FindServices(tmpDir)

	require.NoError(t, err)
	require.Len(t, services, 2)

	// Check that both services were found (order may vary)
	names := make(map[string]bool)
	for _, svc := range services {
		names[svc.Name] = true
	}
	require.True(t, names["myservice"])
	require.True(t, names["api-service"])
}

func TestFindServicesEmpty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	services, err := serviceinfo.FindServices(tmpDir)

	require.NoError(t, err)
	require.Empty(t, services)
}

func TestFindServicesNestedDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create nested/deep/service/pilum.yaml
	nestedDir := filepath.Join(tmpDir, "nested", "deep", "service")
	require.NoError(t, os.MkdirAll(nestedDir, 0755))
	content := `name: nested-service
provider: gcp
`
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "pilum.yaml"), []byte(content), 0644))

	services, err := serviceinfo.FindServices(tmpDir)

	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "nested-service", services[0].Name)
}

func TestFindServicesInvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "invalid")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	// Write invalid YAML
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte("invalid: yaml: content:"), 0644))

	_, err := serviceinfo.FindServices(tmpDir)

	require.Error(t, err)
	require.Contains(t, err.Error(), "error parsing")
}

func TestFindServicesMissingName(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "noname")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	// Write YAML without name field
	// Note: There's a bug in get_services.go:54 where it wraps a nil error
	// This test documents the current (buggy) behavior - it panics
	content := `provider: gcp
region: us-central1
`
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))

	require.Panics(t, func() {
		_, _ = serviceinfo.FindServices(tmpDir)
	})
}

func TestFindServicesNonExistentDirectory(t *testing.T) {
	t.Parallel()

	_, err := serviceinfo.FindServices("/nonexistent/path/that/does/not/exist")

	require.Error(t, err)
}

func TestFilterServices(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "service-a", Provider: "gcp"},
		{Name: "service-b", Provider: "aws"},
		{Name: "service-c", Provider: "azure"},
	}

	tests := []struct {
		name     string
		filter   []string
		expected []string
	}{
		{
			name:     "filter single service",
			filter:   []string{"service-a"},
			expected: []string{"service-a"},
		},
		{
			name:     "filter multiple services",
			filter:   []string{"service-a", "service-c"},
			expected: []string{"service-a", "service-c"},
		},
		{
			name:     "filter all services",
			filter:   []string{"service-a", "service-b", "service-c"},
			expected: []string{"service-a", "service-b", "service-c"},
		},
		{
			name:     "filter with non-existent service",
			filter:   []string{"service-a", "nonexistent"},
			expected: []string{"service-a"},
		},
		{
			name:     "filter with only non-existent",
			filter:   []string{"nonexistent"},
			expected: []string{},
		},
		{
			name:     "empty filter",
			filter:   []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := serviceinfo.FilterServices(tt.filter, services)

			require.Len(t, result, len(tt.expected))
			for i, name := range tt.expected {
				require.Equal(t, name, result[i].Name)
			}
		})
	}
}

func TestFilterServicesPreservesOrder(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "alpha", Provider: "gcp"},
		{Name: "beta", Provider: "aws"},
		{Name: "gamma", Provider: "azure"},
	}

	// Request in different order than found
	filter := []string{"gamma", "alpha", "beta"}
	result := serviceinfo.FilterServices(filter, services)

	require.Len(t, result, 3)
	// Should preserve the order of the filter, not the found order
	require.Equal(t, "gamma", result[0].Name)
	require.Equal(t, "alpha", result[1].Name)
	require.Equal(t, "beta", result[2].Name)
}

func TestFindAndFilterServices(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create multiple services
	for _, name := range []string{"svc-a", "svc-b", "svc-c"} {
		svcDir := filepath.Join(tmpDir, name)
		require.NoError(t, os.MkdirAll(svcDir, 0755))
		content := "name: " + name + "\nprovider: gcp\n"
		require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))
	}

	tests := []struct {
		name     string
		filter   []string
		expected int
	}{
		{
			name:     "no filter returns all",
			filter:   nil,
			expected: 3,
		},
		{
			name:     "empty filter returns all",
			filter:   []string{},
			expected: 3,
		},
		{
			name:     "filter to one",
			filter:   []string{"svc-a"},
			expected: 1,
		},
		{
			name:     "filter to two",
			filter:   []string{"svc-a", "svc-c"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := serviceinfo.FindAndFilterServices(tmpDir, tt.filter)

			require.NoError(t, err)
			require.Len(t, result, tt.expected)
		})
	}
}

func TestFindAndFilterServicesError(t *testing.T) {
	t.Parallel()

	_, err := serviceinfo.FindAndFilterServices("/nonexistent/path", nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "error finding services")
}

func TestListServices(t *testing.T) {
	t.Parallel()

	services := []*serviceinfo.ServiceInfo{
		{
			Name:     "myservice",
			Provider: "gcp",
			Project:  "my-project",
			Region:   "us-central1",
			Runtime:  serviceinfo.RuntimeConfig{Service: "cloud-run"},
		},
		{
			Name:     "api-service",
			Provider: "aws",
			Project:  "aws-account",
			Region:   "us-east-1",
			Runtime:  serviceinfo.RuntimeConfig{Service: "lambda"},
		},
	}

	// This function prints to stdout - just verify it doesn't panic
	require.NotPanics(t, func() {
		serviceinfo.ListServices(services)
	})
}

func TestListServicesEmpty(t *testing.T) {
	t.Parallel()

	services := []*serviceinfo.ServiceInfo{}

	require.NotPanics(t, func() {
		serviceinfo.ListServices(services)
	})
}

func TestListServicesNil(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		serviceinfo.ListServices(nil)
	})
}

func TestCalculateWaves(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		services   []serviceinfo.ServiceInfo
		wantWaves  int // 0 means nil return (flat parallel)
		wantErr    bool
		checkWaves func(t *testing.T, waves [][]serviceinfo.ServiceInfo)
	}{
		{
			name: "no dependencies returns nil",
			services: []serviceinfo.ServiceInfo{
				{Name: "a", Provider: "gcp"},
				{Name: "b", Provider: "gcp"},
			},
			wantWaves: 0,
		},
		{
			name:      "empty services returns nil",
			services:  nil,
			wantWaves: 0,
		},
		{
			name: "single dependency creates two waves",
			services: []serviceinfo.ServiceInfo{
				{Name: "db", Provider: "gcp"},
				{Name: "api", Provider: "gcp", DependsOn: []string{"db"}},
			},
			wantWaves: 2,
			checkWaves: func(t *testing.T, waves [][]serviceinfo.ServiceInfo) {
				require.Len(t, waves[0], 1)
				require.Equal(t, "db", waves[0][0].Name)
				require.Len(t, waves[1], 1)
				require.Equal(t, "api", waves[1][0].Name)
			},
		},
		{
			name: "diamond creates three waves",
			services: []serviceinfo.ServiceInfo{
				{Name: "base", Provider: "gcp"},
				{Name: "auth", Provider: "gcp", DependsOn: []string{"base"}},
				{Name: "users", Provider: "gcp", DependsOn: []string{"base"}},
				{Name: "gateway", Provider: "gcp", DependsOn: []string{"auth", "users"}},
			},
			wantWaves: 3,
			checkWaves: func(t *testing.T, waves [][]serviceinfo.ServiceInfo) {
				require.Len(t, waves[0], 1)
				require.Equal(t, "base", waves[0][0].Name)
				require.Len(t, waves[1], 2) // auth and users
				require.Len(t, waves[2], 1)
				require.Equal(t, "gateway", waves[2][0].Name)
			},
		},
		{
			name: "linear chain creates N waves",
			services: []serviceinfo.ServiceInfo{
				{Name: "a", Provider: "gcp"},
				{Name: "b", Provider: "gcp", DependsOn: []string{"a"}},
				{Name: "c", Provider: "gcp", DependsOn: []string{"b"}},
				{Name: "d", Provider: "gcp", DependsOn: []string{"c"}},
			},
			wantWaves: 4,
			checkWaves: func(t *testing.T, waves [][]serviceinfo.ServiceInfo) {
				require.Len(t, waves[0], 1)
				require.Equal(t, "a", waves[0][0].Name)
				require.Len(t, waves[3], 1)
				require.Equal(t, "d", waves[3][0].Name)
			},
		},
		{
			name: "all depend on one root creates two waves",
			services: []serviceinfo.ServiceInfo{
				{Name: "root", Provider: "gcp"},
				{Name: "svc1", Provider: "gcp", DependsOn: []string{"root"}},
				{Name: "svc2", Provider: "gcp", DependsOn: []string{"root"}},
				{Name: "svc3", Provider: "gcp", DependsOn: []string{"root"}},
			},
			wantWaves: 2,
			checkWaves: func(t *testing.T, waves [][]serviceinfo.ServiceInfo) {
				require.Len(t, waves[0], 1)
				require.Equal(t, "root", waves[0][0].Name)
				require.Len(t, waves[1], 3)
			},
		},
		{
			name: "mixed deps and no-deps",
			services: []serviceinfo.ServiceInfo{
				{Name: "independent", Provider: "gcp"},
				{Name: "base", Provider: "gcp"},
				{Name: "dependent", Provider: "gcp", DependsOn: []string{"base"}},
			},
			wantWaves: 2,
			checkWaves: func(t *testing.T, waves [][]serviceinfo.ServiceInfo) {
				// Wave 0: independent + base (both depth 0)
				require.Len(t, waves[0], 2)
				// Wave 1: dependent
				require.Len(t, waves[1], 1)
				require.Equal(t, "dependent", waves[1][0].Name)
			},
		},
		{
			name: "services with external deps not in set go to wave 0",
			services: []serviceinfo.ServiceInfo{
				{Name: "svc", Provider: "gcp", DependsOn: []string{"external-not-in-set"}},
				{Name: "other", Provider: "gcp", DependsOn: []string{"svc"}},
			},
			wantWaves: 2,
			checkWaves: func(t *testing.T, waves [][]serviceinfo.ServiceInfo) {
				// svc depends only on external (not in graph) → depth 0
				require.Len(t, waves[0], 1)
				require.Equal(t, "svc", waves[0][0].Name)
				require.Len(t, waves[1], 1)
				require.Equal(t, "other", waves[1][0].Name)
			},
		},
		{
			name: "multi-region services in same wave",
			services: []serviceinfo.ServiceInfo{
				{Name: "db", Provider: "gcp", Region: "us-central1"},
				{Name: "api", Provider: "gcp", Region: "us-central1", DependsOn: []string{"db"}, IsMultiRegion: true},
				{Name: "api", Provider: "gcp", Region: "us-east1", DependsOn: []string{"db"}, IsMultiRegion: true},
			},
			wantWaves: 2,
			checkWaves: func(t *testing.T, waves [][]serviceinfo.ServiceInfo) {
				require.Len(t, waves[0], 1) // db
				require.Len(t, waves[1], 2) // both api regions
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			waves, err := serviceinfo.CalculateWaves(tt.services)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantWaves == 0 {
				require.Nil(t, waves)
				return
			}

			require.Len(t, waves, tt.wantWaves)
			if tt.checkWaves != nil {
				tt.checkWaves(t, waves)
			}
		})
	}
}

func TestFindServicesWithBuildConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "go-service")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	content := `name: go-service
provider: gcp
build:
  language: go
  version: "1.23"
  cmd: "go build -o ./dist/app"
`
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))

	services, err := serviceinfo.FindServices(tmpDir)

	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "go-service", services[0].Name)
	require.Equal(t, "go", services[0].BuildConfig.Language)
	require.Equal(t, "1.23", services[0].BuildConfig.Version)
}

func TestFindServicesIgnoresNonServiceFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a pilum.yaml
	svcDir := filepath.Join(tmpDir, "valid")
	require.NoError(t, os.MkdirAll(svcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte("name: valid\nprovider: gcp\n"), 0644))

	// Create other YAML files that should be ignored
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "config.yaml"), []byte("key: value\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "other.yml"), []byte("data: test\n"), 0644))

	// Create a non-YAML file
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "README.md"), []byte("# Readme\n"), 0644))

	services, err := serviceinfo.FindServices(tmpDir)

	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "valid", services[0].Name)
}

func TestDefaultDiscoveryOptions(t *testing.T) {
	t.Parallel()

	opts := serviceinfo.DefaultDiscoveryOptions()

	require.Equal(t, serviceinfo.DefaultMaxDepth, opts.MaxDepth)
	require.False(t, opts.NoGitIgnore)
}

func TestFindServicesWithOptions(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a service
	svcDir := filepath.Join(tmpDir, "myservice")
	require.NoError(t, os.MkdirAll(svcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte("name: myservice\nprovider: gcp\n"), 0644))

	opts := serviceinfo.DiscoveryOptions{
		MaxDepth:    4,
		NoGitIgnore: false,
	}

	services, err := serviceinfo.FindServicesWithOptions(tmpDir, opts)

	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "myservice", services[0].Name)
}

func TestFindServicesWithDepth(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create service at depth 1
	svc1Dir := filepath.Join(tmpDir, "level1")
	require.NoError(t, os.MkdirAll(svc1Dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svc1Dir, "pilum.yaml"), []byte("name: level1-svc\nprovider: gcp\n"), 0644))

	// Create service at depth 3
	svc3Dir := filepath.Join(tmpDir, "a", "b", "level3")
	require.NoError(t, os.MkdirAll(svc3Dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svc3Dir, "pilum.yaml"), []byte("name: level3-svc\nprovider: gcp\n"), 0644))

	// With depth 2, should only find level1
	services, err := serviceinfo.FindServicesWithDepth(tmpDir, 2)
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "level1-svc", services[0].Name)

	// With depth 4, should find both
	services, err = serviceinfo.FindServicesWithDepth(tmpDir, 4)
	require.NoError(t, err)
	require.Len(t, services, 2)
}

func TestFindServicesReadsGitignore(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .gitignore that ignores "ignored-dir"
	gitignore := "ignored-dir\nnode_modules\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignore), 0644))

	// Create service in normal directory
	normalDir := filepath.Join(tmpDir, "normal")
	require.NoError(t, os.MkdirAll(normalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(normalDir, "pilum.yaml"), []byte("name: normal-svc\nprovider: gcp\n"), 0644))

	// Create service in gitignored directory
	ignoredDir := filepath.Join(tmpDir, "ignored-dir")
	require.NoError(t, os.MkdirAll(ignoredDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ignoredDir, "pilum.yaml"), []byte("name: ignored-svc\nprovider: gcp\n"), 0644))

	// With default options (reads .gitignore), should only find normal-svc
	services, err := serviceinfo.FindServices(tmpDir)
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "normal-svc", services[0].Name)
}

func TestFindServicesNoGitIgnoreFlag(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .gitignore that ignores "ignored-dir"
	gitignore := "ignored-dir\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignore), 0644))

	// Create service in normal directory
	normalDir := filepath.Join(tmpDir, "normal")
	require.NoError(t, os.MkdirAll(normalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(normalDir, "pilum.yaml"), []byte("name: normal-svc\nprovider: gcp\n"), 0644))

	// Create service in gitignored directory
	ignoredDir := filepath.Join(tmpDir, "ignored-dir")
	require.NoError(t, os.MkdirAll(ignoredDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ignoredDir, "pilum.yaml"), []byte("name: ignored-svc\nprovider: gcp\n"), 0644))

	// With NoGitIgnore=true, should find BOTH services
	opts := serviceinfo.DiscoveryOptions{
		MaxDepth:    serviceinfo.DefaultMaxDepth,
		NoGitIgnore: true,
	}
	services, err := serviceinfo.FindServicesWithOptions(tmpDir, opts)
	require.NoError(t, err)
	require.Len(t, services, 2)

	names := make(map[string]bool)
	for _, svc := range services {
		names[svc.Name] = true
	}
	require.True(t, names["normal-svc"])
	require.True(t, names["ignored-svc"])
}

func TestFindServicesReadsPilumignore(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pilumignore that ignores "local-only"
	pilumignore := "local-only\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".pilumignore"), []byte(pilumignore), 0644))

	// Create service in normal directory
	normalDir := filepath.Join(tmpDir, "normal")
	require.NoError(t, os.MkdirAll(normalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(normalDir, "pilum.yaml"), []byte("name: normal-svc\nprovider: gcp\n"), 0644))

	// Create service in pilumignored directory
	ignoredDir := filepath.Join(tmpDir, "local-only")
	require.NoError(t, os.MkdirAll(ignoredDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ignoredDir, "pilum.yaml"), []byte("name: local-svc\nprovider: gcp\n"), 0644))

	// Should only find normal-svc
	services, err := serviceinfo.FindServices(tmpDir)
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "normal-svc", services[0].Name)
}

func TestFindServicesCombinesGitignoreAndPilumignore(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .gitignore
	gitignore := "git-ignored\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignore), 0644))

	// Create .pilumignore
	pilumignore := "pilum-ignored\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".pilumignore"), []byte(pilumignore), 0644))

	// Create service in normal directory
	normalDir := filepath.Join(tmpDir, "normal")
	require.NoError(t, os.MkdirAll(normalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(normalDir, "pilum.yaml"), []byte("name: normal-svc\nprovider: gcp\n"), 0644))

	// Create service in gitignored directory
	gitIgnoredDir := filepath.Join(tmpDir, "git-ignored")
	require.NoError(t, os.MkdirAll(gitIgnoredDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitIgnoredDir, "pilum.yaml"), []byte("name: git-ignored-svc\nprovider: gcp\n"), 0644))

	// Create service in pilumignored directory
	pilumIgnoredDir := filepath.Join(tmpDir, "pilum-ignored")
	require.NoError(t, os.MkdirAll(pilumIgnoredDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(pilumIgnoredDir, "pilum.yaml"), []byte("name: pilum-ignored-svc\nprovider: gcp\n"), 0644))

	// Should only find normal-svc (both ignore files are combined)
	services, err := serviceinfo.FindServices(tmpDir)
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "normal-svc", services[0].Name)
}

func TestFindServicesFallbackIgnorePatterns(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// No .gitignore or .pilumignore - should use fallback patterns

	// Create service in normal directory
	normalDir := filepath.Join(tmpDir, "normal")
	require.NoError(t, os.MkdirAll(normalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(normalDir, "pilum.yaml"), []byte("name: normal-svc\nprovider: gcp\n"), 0644))

	// Create service in node_modules (should be ignored by fallback)
	nodeModulesDir := filepath.Join(tmpDir, "node_modules", "some-pkg")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nodeModulesDir, "pilum.yaml"), []byte("name: npm-svc\nprovider: gcp\n"), 0644))

	// Create service in .git (should be ignored by fallback)
	gitDir := filepath.Join(tmpDir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "pilum.yaml"), []byte("name: git-svc\nprovider: gcp\n"), 0644))

	// Should only find normal-svc
	services, err := serviceinfo.FindServices(tmpDir)
	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "normal-svc", services[0].Name)
}

func TestFindServicesWithEnv_ScalarOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "api")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	content := `name: api
provider: gcp
project: acme-dev
region: us-central1
environments:
  prod:
    project: acme-prod
    region: us-east1
`
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))

	opts := serviceinfo.DiscoveryOptions{
		MaxDepth: serviceinfo.DefaultMaxDepth,
		Env:      "prod",
	}
	services, err := serviceinfo.FindServicesWithOptions(tmpDir, opts)

	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "api", services[0].Name)
	require.Equal(t, "acme-prod", services[0].Project)
	require.Equal(t, "us-east1", services[0].Region)
}

func TestFindServicesWithEnv_NestedMapMerge(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "api")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	content := `name: api
provider: gcp
project: acme-dev
cloud_run:
  min_instances: 0
  max_instances: 10
environments:
  prod:
    cloud_run:
      min_instances: 2
`
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))

	opts := serviceinfo.DiscoveryOptions{
		MaxDepth: serviceinfo.DefaultMaxDepth,
		Env:      "prod",
	}
	services, err := serviceinfo.FindServicesWithOptions(tmpDir, opts)

	require.NoError(t, err)
	require.Len(t, services, 1)
	// Verify the nested map was deep-merged: min_instances overridden, max_instances preserved
	cloudRun, ok := services[0].Config["cloud_run"].(map[string]any)
	require.True(t, ok, "cloud_run should be map[string]any")
	require.Equal(t, 2, cloudRun["min_instances"])
	require.Equal(t, 10, cloudRun["max_instances"])
}

func TestFindServicesWithEnv_MissingEnvError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "api")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	content := `name: api
provider: gcp
environments:
  staging:
    project: acme-staging
  prod:
    project: acme-prod
`
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))

	opts := serviceinfo.DiscoveryOptions{
		MaxDepth: serviceinfo.DefaultMaxDepth,
		Env:      "qa",
	}
	_, err := serviceinfo.FindServicesWithOptions(tmpDir, opts)

	require.Error(t, err)
	require.Contains(t, err.Error(), `environment "qa" not found`)
	require.Contains(t, err.Error(), "prod")
	require.Contains(t, err.Error(), "staging")
}

func TestFindServicesWithEnv_NoEnvironmentsBlockSkipped(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "worker")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	// Service with no environments block — env-agnostic
	content := `name: worker
provider: gcp
project: acme-dev
region: us-central1
`
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))

	opts := serviceinfo.DiscoveryOptions{
		MaxDepth: serviceinfo.DefaultMaxDepth,
		Env:      "prod",
	}
	services, err := serviceinfo.FindServicesWithOptions(tmpDir, opts)

	require.NoError(t, err)
	require.Len(t, services, 1)
	require.Equal(t, "worker", services[0].Name)
	require.Equal(t, "acme-dev", services[0].Project)
}

func TestFindServicesWithEnv_EnvironmentsStripped(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "api")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	content := `name: api
provider: gcp
project: acme-dev
environments:
  prod:
    project: acme-prod
`
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))

	// With --env: environments key should be stripped from Config
	opts := serviceinfo.DiscoveryOptions{
		MaxDepth: serviceinfo.DefaultMaxDepth,
		Env:      "prod",
	}
	services, err := serviceinfo.FindServicesWithOptions(tmpDir, opts)
	require.NoError(t, err)
	require.Len(t, services, 1)
	_, hasEnvs := services[0].Config["environments"]
	require.False(t, hasEnvs, "environments key should be stripped when --env is used")

	// Without --env: environments key should also be stripped
	optsNoEnv := serviceinfo.DiscoveryOptions{
		MaxDepth: serviceinfo.DefaultMaxDepth,
	}
	services2, err := serviceinfo.FindServicesWithOptions(tmpDir, optsNoEnv)
	require.NoError(t, err)
	require.Len(t, services2, 1)
	_, hasEnvs2 := services2[0].Config["environments"]
	require.False(t, hasEnvs2, "environments key should be stripped even without --env")
}

func TestFindServicesWithEnv_RegionOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "global-api")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	content := `name: global-api
provider: gcp
project: acme-dev
regions:
  - us-central1
  - us-east1
environments:
  prod:
    project: acme-prod
    regions:
      - eu-west1
      - ap-southeast1
      - us-central1
`
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))

	opts := serviceinfo.DiscoveryOptions{
		MaxDepth: serviceinfo.DefaultMaxDepth,
		Env:      "prod",
	}
	services, err := serviceinfo.FindServicesWithOptions(tmpDir, opts)

	require.NoError(t, err)
	// Multi-region expansion should create 3 instances (prod regions replace base regions)
	require.Len(t, services, 3)
	require.Equal(t, "acme-prod", services[0].Project)
	regions := make(map[string]bool)
	for _, svc := range services {
		regions[svc.Region] = true
	}
	require.True(t, regions["eu-west1"])
	require.True(t, regions["ap-southeast1"])
	require.True(t, regions["us-central1"])
}

func TestFindServicesWithEnv_DependsOnOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "api")
	require.NoError(t, os.MkdirAll(svcDir, 0755))

	content := `name: api
provider: gcp
project: acme-dev
depends_on:
  - db
  - cache
environments:
  prod:
    depends_on:
      - db
      - cache
      - auth
`
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "pilum.yaml"), []byte(content), 0644))

	opts := serviceinfo.DiscoveryOptions{
		MaxDepth: serviceinfo.DefaultMaxDepth,
		Env:      "prod",
	}
	services, err := serviceinfo.FindServicesWithOptions(tmpDir, opts)

	require.NoError(t, err)
	require.Len(t, services, 1)
	// Arrays are replaced entirely, so depends_on should be the prod override
	require.Equal(t, []string{"db", "cache", "auth"}, services[0].DependsOn)
}

func TestFilterOptionsNoGitIgnore(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .gitignore
	gitignore := "ignored-dir\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignore), 0644))

	// Create services
	normalDir := filepath.Join(tmpDir, "normal")
	require.NoError(t, os.MkdirAll(normalDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(normalDir, "pilum.yaml"), []byte("name: normal-svc\nprovider: gcp\n"), 0644))

	ignoredDir := filepath.Join(tmpDir, "ignored-dir")
	require.NoError(t, os.MkdirAll(ignoredDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ignoredDir, "pilum.yaml"), []byte("name: ignored-svc\nprovider: gcp\n"), 0644))

	// Test with NoGitIgnore in FilterOptions
	opts := serviceinfo.FilterOptions{
		NoGitIgnore: true,
	}
	services, err := serviceinfo.FindAndFilterServicesWithOptions(tmpDir, opts)
	require.NoError(t, err)
	require.Len(t, services, 2)
}
