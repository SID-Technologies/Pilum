package compose

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPresetKnownTypes(t *testing.T) {
	t.Parallel()

	for _, typeName := range []string{"postgres", "mysql", "mariadb", "redis", "mongo"} {
		p, ok := GetPreset(typeName)
		require.True(t, ok, "preset %q should exist", typeName)
		require.NotEmpty(t, p.Image, "preset %q should have an image", typeName)
		require.Greater(t, p.DefaultPort, 0, "preset %q should have a port", typeName)
		require.NotNil(t, p.HealthCmd, "preset %q should have a health check", typeName)
		require.NotNil(t, p.ConnStringTmpl, "preset %q should have a conn string template", typeName)
		require.NotNil(t, p.Env, "preset %q should have an env func", typeName)
		require.NotNil(t, p.Volumes, "preset %q should have a volumes func", typeName)
	}
}

func TestGetPresetUnknown(t *testing.T) {
	t.Parallel()

	_, ok := GetPreset("cockroachdb")
	require.False(t, ok)
}

func TestSupportedTypes(t *testing.T) {
	t.Parallel()

	types := SupportedTypes()
	require.Len(t, types, 5)
	// Should be sorted
	require.Equal(t, "mariadb", types[0])
	require.Equal(t, "mongo", types[1])
	require.Equal(t, "mysql", types[2])
	require.Equal(t, "postgres", types[3])
	require.Equal(t, "redis", types[4])
}

func TestPresetHealthCmds(t *testing.T) {
	t.Parallel()

	dep := Dependency{Name: "mydb", User: "testuser", Database: "testdb", Port: 5432}

	p, _ := GetPreset("postgres")
	require.Contains(t, p.HealthCmd(dep), "pg_isready")
	require.Contains(t, p.HealthCmd(dep), "testuser")
	require.Contains(t, p.HealthCmd(dep), "testdb")

	p, _ = GetPreset("mysql")
	require.Contains(t, p.HealthCmd(dep), "mysqladmin ping")

	p, _ = GetPreset("redis")
	require.Contains(t, p.HealthCmd(dep), "redis-cli ping")

	p, _ = GetPreset("mongo")
	require.Contains(t, p.HealthCmd(dep), "mongosh")
}

func TestPresetConnStringTemplates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typeName string
		dep      Dependency
		contains string
	}{
		{
			typeName: "postgres",
			dep:      Dependency{Name: "pg", User: "u", Password: "p", Database: "db", Port: 5432},
			contains: "postgresql://u:p@pg:5432/db",
		},
		{
			typeName: "mysql",
			dep:      Dependency{Name: "my", User: "u", Password: "p", Database: "db", Port: 3306},
			contains: "mysql://u:p@my:3306/db",
		},
		{
			typeName: "redis",
			dep:      Dependency{Name: "rd", Port: 6379},
			contains: "redis://rd:6379",
		},
		{
			typeName: "mongo",
			dep:      Dependency{Name: "mg", User: "u", Password: "p", Database: "db", Port: 27017},
			contains: "mongodb://u:p@mg:27017/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			t.Parallel()
			p, _ := GetPreset(tt.typeName)
			result := p.ConnStringTmpl(tt.dep)
			require.Contains(t, result, tt.contains)
		})
	}
}

func TestPresetEnvVars(t *testing.T) {
	t.Parallel()

	dep := Dependency{Name: "pg", User: "testuser", Password: "testpass", Database: "testdb"}

	p, _ := GetPreset("postgres")
	env := p.Env(dep)
	require.Contains(t, env, "POSTGRES_DB=testdb")
	require.Contains(t, env, "POSTGRES_USER=testuser")
	require.Contains(t, env, "POSTGRES_PASSWORD=testpass")

	p, _ = GetPreset("redis")
	env = p.Env(dep)
	require.Nil(t, env)
}
