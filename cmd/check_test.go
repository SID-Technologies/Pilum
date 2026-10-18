package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- check --recipe-path (changes the working directory, not parallel) ---

func TestCheckUsesRecipePath(t *testing.T) {
	dir := t.TempDir()
	recipeYAML := "name: custom\nprovider: custom\nsteps:\n  - name: deploy\n    command: echo deploy\n"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "recipes"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "recipes", "custom-recipe.yaml"), []byte(recipeYAML), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "svc-a"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "svc-a", "pilum.yaml"), []byte("name: svc-a\nprovider: custom\n"), 0o600))
	t.Chdir(dir)

	// The embedded recipes have no "custom" provider.
	cmd := CheckCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{})
	require.Error(t, cmd.Execute())

	cmd = CheckCmd()
	cmd.SetArgs([]string{"--recipe-path", "recipes"})
	require.NoError(t, cmd.Execute())
}
