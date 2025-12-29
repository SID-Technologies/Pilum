package docker_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/docker"

	"github.com/stretchr/testify/require"
)

func TestGenerateDockerPushCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		imageName string
		expected  []string
	}{
		{
			name:      "basic push command",
			imageName: "myimage:latest",
			expected:  []string{"docker", "push", "myimage:latest"},
		},
		{
			name:      "gcr image",
			imageName: "gcr.io/my-project/myservice:v1.0.0",
			expected:  []string{"docker", "push", "gcr.io/my-project/myservice:v1.0.0"},
		},
		{
			name:      "artifact registry image",
			imageName: "us-central1-docker.pkg.dev/project/repo/image:tag",
			expected:  []string{"docker", "push", "us-central1-docker.pkg.dev/project/repo/image:tag"},
		},
		{
			name:      "docker hub image",
			imageName: "docker.io/username/image:latest",
			expected:  []string{"docker", "push", "docker.io/username/image:latest"},
		},
		{
			name:      "github container registry",
			imageName: "ghcr.io/org/image:v2.0.0",
			expected:  []string{"docker", "push", "ghcr.io/org/image:v2.0.0"},
		},
		{
			name:      "ecr image",
			imageName: "123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:latest",
			expected:  []string{"docker", "push", "123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:latest"},
		},
		{
			name:      "azure container registry",
			imageName: "myregistry.azurecr.io/myimage:v1",
			expected:  []string{"docker", "push", "myregistry.azurecr.io/myimage:v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := docker.GenerateDockerPushCommand(tt.imageName)

			require.Equal(t, tt.expected, cmd)
			require.Len(t, cmd, 3)
			require.Equal(t, "docker", cmd[0])
			require.Equal(t, "push", cmd[1])
			require.Equal(t, tt.imageName, cmd[2])
		})
	}
}

func TestGenerateDockerPushCommandFormat(t *testing.T) {
	t.Parallel()

	cmd := docker.GenerateDockerPushCommand("image:tag")

	require.Len(t, cmd, 3)
	require.Equal(t, "docker", cmd[0])
	require.Equal(t, "push", cmd[1])
}
