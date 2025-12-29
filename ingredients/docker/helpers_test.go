package docker_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/docker"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/stretchr/testify/require"
)

func TestGenerateImageName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		service  serviceinfo.ServiceInfo
		registry string
		tag      string
		expected string
	}{
		{
			name: "AWS ECR image",
			service: serviceinfo.ServiceInfo{
				Name:         "myservice",
				Provider:     "aws",
				Region:       "us-east-1",
				RegistryName: "123456789012",
			},
			registry: "",
			tag:      "v1.0.0",
			expected: "123456789012.dkr.ecr.us-east-1.amazonaws.com/myservice::v1.0.0",
		},
		{
			name: "GCP Artifact Registry image",
			service: serviceinfo.ServiceInfo{
				Name:         "myservice",
				Provider:     "gcp",
				Region:       "us-central1",
				Project:      "my-project",
				RegistryName: "my-repo",
			},
			registry: "",
			tag:      "v1.0.0",
			expected: "us-central1-docker.pkg.dev/my-project/my-repo/myservice::v1.0.0",
		},
		{
			name: "Azure Container Registry image",
			service: serviceinfo.ServiceInfo{
				Name:         "myservice",
				Provider:     "azure",
				RegistryName: "myregistry",
			},
			registry: "",
			tag:      "v1.0.0",
			expected: "myregistry.azurecr.io/myservice::v1.0.0",
		},
		{
			name: "Docker Hub image",
			service: serviceinfo.ServiceInfo{
				Name:     "myservice",
				Provider: "dockerhub",
			},
			registry: "",
			tag:      "v1.0.0",
			expected: "docker.io/myservice::v1.0.0",
		},
		{
			name: "GitLab Registry image",
			service: serviceinfo.ServiceInfo{
				Name:         "myservice",
				Provider:     "gitlab",
				RegistryName: "mygroup",
			},
			registry: "",
			tag:      "v1.0.0",
			expected: "mygroup.gitlab.io/myservice::v1.0.0",
		},
		{
			name: "GitHub Container Registry image",
			service: serviceinfo.ServiceInfo{
				Name:         "myservice",
				Provider:     "github",
				RegistryName: "myorg",
			},
			registry: "",
			tag:      "v1.0.0",
			expected: "ghcr.io/myorg/myservice::v1.0.0",
		},
		{
			name: "unknown provider defaults to service name",
			service: serviceinfo.ServiceInfo{
				Name:     "myservice",
				Provider: "unknown",
			},
			registry: "",
			tag:      "v1.0.0",
			expected: "myservice::v1.0.0",
		},
		{
			name: "explicit registry overrides provider",
			service: serviceinfo.ServiceInfo{
				Name:     "myservice",
				Provider: "gcp",
				Region:   "us-central1",
				Project:  "my-project",
			},
			registry: "custom.registry.io/org",
			tag:      "v1.0.0",
			expected: "custom.registry.io/org/myservice::v1.0.0",
		},
		{
			name: "empty tag defaults to latest",
			service: serviceinfo.ServiceInfo{
				Name:     "myservice",
				Provider: "dockerhub",
			},
			registry: "",
			tag:      "",
			expected: "docker.io/myservice::latest",
		},
		{
			name: "empty provider with no registry",
			service: serviceinfo.ServiceInfo{
				Name:     "myservice",
				Provider: "",
			},
			registry: "",
			tag:      "v1.0.0",
			expected: "myservice::v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := docker.GenerateImageName(tt.service, tt.registry, tt.tag)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateImageNameWithLatestDefault(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:     "testservice",
		Provider: "dockerhub",
	}

	// Test that empty tag defaults to "latest"
	result := docker.GenerateImageName(service, "", "")
	require.Contains(t, result, ":latest")
}
