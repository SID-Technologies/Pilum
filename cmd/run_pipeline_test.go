package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- runPipeline --only-changed (changes the working directory, not parallel) ---

func TestRunPipelineOnlyChangedWithNothingChanged(t *testing.T) {
	dir := t.TempDir()
	svc := "name: svc-a\nprovider: gcp\ntype: gcp-cloud-run\nproject: p\nregion: us-central1\n"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "svc-a"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "svc-a", "pilum.yaml"), []byte(svc), 0o600))
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "-A"},
		{"-c", "user.email=t@example.com", "-c", "user.name=t", "commit", "-qm", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	t.Chdir(dir)

	opts := deploymentOptions{DryRun: true, OnlyChanged: true, Since: "HEAD"}
	// svc-a is named, and nothing changed since HEAD: that's nothing to do, not an error.
	require.NoError(t, runPipeline("dry-run", []string{"svc-a"}, opts, "no services"))
}
