package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteComposeFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "docker-compose.yml")

	file := &File{
		Services: map[string]*Service{
			"api": {
				Image:         "api:latest",
				ContainerName: "api",
				Restart:       "unless-stopped",
				Ports:         []string{"8080:8080"},
			},
		},
		Networks: map[string]any{
			"app-network": map[string]string{"driver": "bridge"},
		},
	}

	err := WriteComposeFile(file, outputPath)
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "api:latest")
	require.Contains(t, string(content), "8080:8080")
	require.Contains(t, string(content), "app-network")
}

func TestWriteComposeFileCreatesSubdirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "sub", "dir", "docker-compose.yml")

	file := &File{
		Services: map[string]*Service{},
		Networks: map[string]any{},
	}

	err := WriteComposeFile(file, outputPath)
	require.NoError(t, err)

	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}
