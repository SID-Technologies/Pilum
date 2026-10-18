package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/schemas"

	"github.com/stretchr/testify/require"
)

func TestSchemaFindings(t *testing.T) {
	t.Parallel()

	svc := &serviceinfo.ServiceInfo{
		Name: "api",
		Config: map[string]any{
			"name":      "api",
			"cloud_run": map[string]any{},
			"fly":       map[string]any{"app": "a"}, // read by the custom recipe
			"secret":    map[string]any{"A": "b"},   // a typo for secrets
		},
		SchemaProblems: []schemas.Problem{
			{Path: "cloud_run.cpu_throtling", Message: "unknown key"},
			{Path: "build.version", Message: "got number, want string", Warning: true},
		},
	}
	rcp := recipe.Recipe{RequiredFields: []recipe.RequiredField{{Name: "fly.app"}}}

	schemaErrors, warnings := schemaFindings(svc, rcp, []string{"cloud_run", "name", "secrets"})

	require.Equal(t, []string{"cloud_run.cpu_throtling: unknown key"}, schemaErrors)
	require.Equal(t, []string{
		"build.version: got number, want string",
		"secret: unknown key - did you mean 'secrets'?",
	}, warnings)
}

func TestCheckFailsOnSchemaErrors(t *testing.T) {
	dir := t.TempDir()
	svc := "name: svc-a\nprovider: gcp\ntype: gcp-cloud-run\nproject: p\nregion: us-central1\nregistry_name: r\ntemplate: t\ncloud_run:\n  cpu_throtling: false\n"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "svc-a"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "svc-a", "pilum.yaml"), []byte(svc), 0o600))
	t.Chdir(dir)

	cmd := CheckCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "cloud_run.cpu_throtling: unknown key - did you mean 'cpu_throttling'?")
}
