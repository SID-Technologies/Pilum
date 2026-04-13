package compose

import (
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateWithDependencies(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "api", Secrets: []serviceinfo.Secrets{{Name: "DB_CONNECTION"}}},
		{Name: "worker", Secrets: []serviceinfo.Secrets{{Name: "REDIS_URL"}}},
	}

	opts := Options{
		Tag: "latest",
		Dependencies: []Dependency{
			{Type: "postgres", Name: "postgres", Secrets: []string{"DB_CONNECTION"}},
			{Type: "redis", Name: "redis", Secrets: []string{"REDIS_URL"}},
		},
	}

	file := Generate(services, opts)

	// Infrastructure services present
	require.Contains(t, file.Services, "postgres")
	require.Contains(t, file.Services, "redis")

	// App services present
	require.Contains(t, file.Services, "api")
	require.Contains(t, file.Services, "worker")

	// Secret wiring: api → postgres
	apiSvc := file.Services["api"]
	require.Contains(t, apiSvc.DependsOn, "postgres")
	found := false
	for _, env := range apiSvc.Environment {
		if env == "DB_CONNECTION=postgresql://appuser:apppassword@postgres:5432/appdb?sslmode=disable" {
			found = true
			break
		}
	}
	require.True(t, found, "DB_CONNECTION should be wired to postgres conn string")

	// Secret wiring: worker → redis
	workerSvc := file.Services["worker"]
	require.Contains(t, workerSvc.DependsOn, "redis")

	// Volumes registered
	require.Contains(t, file.Volumes, "postgres_data")
	require.Contains(t, file.Volumes, "redis_data")
}

func TestGenerateMigrationOrdering(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "api"},
		{Name: "migrations"},
	}

	file := Generate(services, Options{Tag: "latest"})

	// Migration service has restart: no
	require.Equal(t, "no", file.Services["migrations"].Restart)

	// Non-migration depends on migration
	apiDeps := file.Services["api"].DependsOn
	require.Contains(t, apiDeps, "migrations")
	depCond, ok := apiDeps["migrations"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, "service_completed_successfully", depCond["condition"])

	// Migration does not depend on itself
	require.NotContains(t, file.Services["migrations"].DependsOn, "migrations")
}

func TestGenerateDisableDeps(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "api", Secrets: []serviceinfo.Secrets{{Name: "DB_CONNECTION"}}},
	}

	opts := Options{
		Tag:         "latest",
		DisableDeps: true,
		Dependencies: []Dependency{
			{Type: "postgres", Name: "postgres", Secrets: []string{"DB_CONNECTION"}},
		},
	}

	file := Generate(services, opts)

	require.NotContains(t, file.Services, "postgres")
	require.Contains(t, file.Services, "api")
	require.Nil(t, file.Volumes)
}

func TestGenerateNoDependencies(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "api"},
		{Name: "web"},
	}

	file := Generate(services, Options{Tag: "latest"})

	require.Len(t, file.Services, 2)
	require.Nil(t, file.Volumes)
	require.Contains(t, file.Networks, "app-network")
}

func TestGenerateDeterministicOrder(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "zebra"},
		{Name: "alpha"},
	}

	file := Generate(services, Options{Tag: "latest"})

	// alpha gets port 8080 (index 0), zebra gets 8081 (index 1)
	require.Contains(t, file.Services["alpha"].Ports, "8080:8080")
	require.Contains(t, file.Services["zebra"].Ports, "8081:8080")
}
