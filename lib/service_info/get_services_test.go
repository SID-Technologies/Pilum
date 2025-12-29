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

	// Create service1/service.yaml
	svc1Dir := filepath.Join(tmpDir, "service1")
	require.NoError(t, os.MkdirAll(svc1Dir, 0755))
	svc1Content := `name: myservice
provider: gcp
region: us-central1
project: my-project
`
	require.NoError(t, os.WriteFile(filepath.Join(svc1Dir, "service.yaml"), []byte(svc1Content), 0644))

	// Create service2/service.yaml
	svc2Dir := filepath.Join(tmpDir, "service2")
	require.NoError(t, os.MkdirAll(svc2Dir, 0755))
	svc2Content := `name: api-service
provider: aws
region: us-east-1
`
	require.NoError(t, os.WriteFile(filepath.Join(svc2Dir, "service.yaml"), []byte(svc2Content), 0644))

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

	// Create nested/deep/service/service.yaml
	nestedDir := filepath.Join(tmpDir, "nested", "deep", "service")
	require.NoError(t, os.MkdirAll(nestedDir, 0755))
	content := `name: nested-service
provider: gcp
`
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "service.yaml"), []byte(content), 0644))

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
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "service.yaml"), []byte("invalid: yaml: content:"), 0644))

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
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "service.yaml"), []byte(content), 0644))

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
		require.NoError(t, os.WriteFile(filepath.Join(svcDir, "service.yaml"), []byte(content), 0644))
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
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "service.yaml"), []byte(content), 0644))

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

	// Create a service.yaml
	svcDir := filepath.Join(tmpDir, "valid")
	require.NoError(t, os.MkdirAll(svcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "service.yaml"), []byte("name: valid\nprovider: gcp\n"), 0644))

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
