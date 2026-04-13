package compose

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeDefaultsFillsEmptyFields(t *testing.T) {
	t.Parallel()

	dep := Dependency{Type: "postgres"}
	MergeDefaults(&dep)

	require.Equal(t, "postgres:15-alpine", dep.Image)
	require.Equal(t, 5432, dep.Port)
	require.Equal(t, "appuser", dep.User)
	require.Equal(t, "apppassword", dep.Password)
	require.Equal(t, "appdb", dep.Database)
}

func TestMergeDefaultsPreservesExplicit(t *testing.T) {
	t.Parallel()

	dep := Dependency{
		Type:     "postgres",
		Image:    "postgres:16",
		Port:     5433,
		User:     "custom",
		Password: "secret",
		Database: "mydb",
	}
	MergeDefaults(&dep)

	require.Equal(t, "postgres:16", dep.Image)
	require.Equal(t, 5433, dep.Port)
	require.Equal(t, "custom", dep.User)
	require.Equal(t, "secret", dep.Password)
	require.Equal(t, "mydb", dep.Database)
}

func TestMergeDefaultsCustomTypeNoOp(t *testing.T) {
	t.Parallel()

	dep := Dependency{Type: "custom", Image: "vault:1.15", Port: 8200}
	MergeDefaults(&dep)

	// Should not change anything — custom has no preset
	require.Equal(t, "vault:1.15", dep.Image)
	require.Equal(t, 8200, dep.Port)
	require.Equal(t, "", dep.User)
}

func TestBuildDependencyServicePostgres(t *testing.T) {
	t.Parallel()

	dep := Dependency{
		Type:     "postgres",
		Name:     "postgres",
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
	}

	svc, volumes := BuildDependencyService(dep)

	require.Equal(t, "postgres:15-alpine", svc.Image)
	require.Equal(t, "postgres", svc.ContainerName)
	require.Equal(t, "unless-stopped", svc.Restart)
	require.Contains(t, svc.Ports, "5432:5432")
	require.NotNil(t, svc.Healthcheck)
	require.Contains(t, svc.Healthcheck.Test[1], "pg_isready")
	require.NotEmpty(t, svc.Environment)
	require.NotEmpty(t, svc.Volumes)
	require.Contains(t, volumes, "postgres_data")
}

func TestBuildDependencyServiceRedis(t *testing.T) {
	t.Parallel()

	dep := Dependency{Type: "redis", Name: "redis"}

	svc, volumes := BuildDependencyService(dep)

	require.Equal(t, "redis:7-alpine", svc.Image)
	require.Contains(t, svc.Ports, "6379:6379")
	require.NotNil(t, svc.Healthcheck)
	require.Contains(t, svc.Healthcheck.Test[1], "redis-cli ping")
	require.Empty(t, svc.Environment) // redis has no env vars
	require.Contains(t, volumes, "redis_data")
}

func TestBuildDependencyServiceCustom(t *testing.T) {
	t.Parallel()

	dep := Dependency{
		Type:        "custom",
		Name:        "vault",
		Image:       "hashicorp/vault:1.15",
		Port:        8200,
		Healthcheck: "vault status",
		Environment: map[string]string{"VAULT_DEV_ROOT_TOKEN_ID": "dev-token"},
	}

	svc, volumes := BuildDependencyService(dep)

	require.Equal(t, "hashicorp/vault:1.15", svc.Image)
	require.Equal(t, "vault", svc.ContainerName)
	require.Contains(t, svc.Ports, "8200:8200")
	require.NotNil(t, svc.Healthcheck)
	require.Contains(t, svc.Healthcheck.Test[1], "vault status")
	require.Contains(t, svc.Environment, "VAULT_DEV_ROOT_TOKEN_ID=dev-token")
	require.Empty(t, volumes) // custom has no preset volumes
}

func TestRenderConnectionStringPostgres(t *testing.T) {
	t.Parallel()

	dep := Dependency{
		Type:     "postgres",
		Name:     "pg",
		User:     "u",
		Password: "p",
		Database: "db",
	}

	conn := RenderConnectionString(dep)
	require.Equal(t, "postgresql://u:p@pg:5432/db?sslmode=disable", conn)
}

func TestRenderConnectionStringRedis(t *testing.T) {
	t.Parallel()

	dep := Dependency{Type: "redis", Name: "rd"}

	conn := RenderConnectionString(dep)
	require.Equal(t, "redis://rd:6379", conn)
}

func TestRenderConnectionStringCustom(t *testing.T) {
	t.Parallel()

	dep := Dependency{Type: "custom", Image: "vault:1.15"}

	conn := RenderConnectionString(dep)
	require.Equal(t, "", conn) // custom has no conn string template
}

func TestBuildDependencyServiceVolumeExtraction(t *testing.T) {
	t.Parallel()

	dep := Dependency{Type: "postgres", Name: "pg"}
	_, volumes := BuildDependencyService(dep)

	// Should extract named volume but not bind mount
	require.Contains(t, volumes, "pg_data")
	// "./init-scripts:..." should NOT be in the volume names list
	for _, v := range volumes {
		require.NotContains(t, v, "init-scripts")
	}
}
