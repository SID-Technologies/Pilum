package compose

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestParseDependencyPostgres(t *testing.T) {
	t.Parallel()

	dep, err := parseDependency(map[string]any{
		"type":     "postgres",
		"database": "mydb",
		"user":     "myuser",
		"password": "mypass",
		"secrets":  []any{"DB_CONNECTION", "DATABASE_URL"},
	})

	require.NoError(t, err)
	require.Equal(t, "postgres", dep.Type)
	require.Equal(t, "postgres", dep.Name) // defaults to type
	require.Equal(t, "mydb", dep.Database)
	require.Equal(t, "myuser", dep.User)
	require.Equal(t, "mypass", dep.Password)
	require.Equal(t, []string{"DB_CONNECTION", "DATABASE_URL"}, dep.Secrets)
}

func TestParseDependencyCustom(t *testing.T) {
	t.Parallel()

	dep, err := parseDependency(map[string]any{
		"type":        "custom",
		"name":        "vault",
		"image":       "hashicorp/vault:1.15",
		"port":        8200,
		"healthcheck": "vault status",
		"environment": map[string]any{
			"VAULT_DEV_ROOT_TOKEN_ID": "dev-token",
		},
		"secrets": []any{"VAULT_ADDR"},
	})

	require.NoError(t, err)
	require.Equal(t, "custom", dep.Type)
	require.Equal(t, "vault", dep.Name)
	require.Equal(t, "hashicorp/vault:1.15", dep.Image)
	require.Equal(t, 8200, dep.Port)
	require.Equal(t, "vault status", dep.Healthcheck)
	require.Equal(t, map[string]string{"VAULT_DEV_ROOT_TOKEN_ID": "dev-token"}, dep.Environment)
	require.Equal(t, []string{"VAULT_ADDR"}, dep.Secrets)
}

func TestParseDependencyMissingType(t *testing.T) {
	t.Parallel()

	_, err := parseDependency(map[string]any{
		"database": "mydb",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required field: type")
}

func TestParseDependencyCustomWithoutImage(t *testing.T) {
	t.Parallel()

	_, err := parseDependency(map[string]any{
		"type": "custom",
		"name": "vault",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "custom dependency requires an image")
}

func TestParseDependencyUnknownType(t *testing.T) {
	t.Parallel()

	_, err := parseDependency(map[string]any{
		"type": "cockroachdb",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown dependency type")
}

func TestParseDependencyWithNameOverride(t *testing.T) {
	t.Parallel()

	dep, err := parseDependency(map[string]any{
		"type": "postgres",
		"name": "primary-db",
	})

	require.NoError(t, err)
	require.Equal(t, "primary-db", dep.Name)
}

// Viper tests are not parallel (global state).

func TestLoadDependenciesFromViper(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("compose", map[string]any{
		"dependencies": []any{
			map[string]any{
				"type":    "postgres",
				"secrets": []any{"DB_CONNECTION"},
			},
			map[string]any{
				"type":    "redis",
				"secrets": []any{"REDIS_URL"},
			},
		},
	})

	deps, err := LoadDependencies()
	require.NoError(t, err)
	require.Len(t, deps, 2)
	require.Equal(t, "postgres", deps[0].Type)
	require.Equal(t, "redis", deps[1].Type)
}

func TestLoadDependenciesNoComposeSection(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	deps, err := LoadDependencies()
	require.NoError(t, err)
	require.Nil(t, deps)
}

func TestLoadDependenciesEmptyDependencies(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("compose", map[string]any{
		"dependencies": []any{},
	})

	deps, err := LoadDependencies()
	require.NoError(t, err)
	require.Nil(t, deps)
}

func TestLoadDependenciesInvalidEntry(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("compose", map[string]any{
		"dependencies": []any{
			map[string]any{
				"type": "custom",
				// Missing image for custom type
			},
		},
	})

	_, err := LoadDependencies()
	require.Error(t, err)
	require.Contains(t, err.Error(), "compose.dependencies[0]")
}
