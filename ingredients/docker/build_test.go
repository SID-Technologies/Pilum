package docker_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/docker"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateDockerBuildCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		service      serviceinfo.ServiceInfo
		imageName    string
		templatePath string
		expectedLen  int
	}{
		{
			name: "basic build command",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
			},
			imageName:    "gcr.io/project/myservice:latest",
			templatePath: "./_templates/Dockerfile",
			expectedLen:  9,
		},
		{
			name: "build with env vars",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				EnvVars: []serviceinfo.EnvVars{
					{Name: "API_URL", Value: "https://api.example.com"},
				},
			},
			imageName:    "gcr.io/project/myservice:latest",
			templatePath: "./_templates/Dockerfile",
			expectedLen:  11, // 2 more for --build-arg with env vars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := docker.GenerateDockerBuildCommand(tt.service, tt.imageName, tt.templatePath)

			require.Len(t, cmd, tt.expectedLen)
			require.Equal(t, "docker", cmd[0])
			require.Equal(t, "build", cmd[1])
		})
	}
}

func TestGenerateDockerBuildCommandImageTag(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "myservice",
	}

	imageName := "us-central1-docker.pkg.dev/project/repo/myservice:v1.0.0"
	cmd := docker.GenerateDockerBuildCommand(service, imageName, "Dockerfile")

	// Find -t flag and check image name
	for i, arg := range cmd {
		if arg == "-t" && i+1 < len(cmd) {
			require.Equal(t, imageName, cmd[i+1])
			return
		}
	}
	t.Fatal("-t flag not found")
}

func TestGenerateDockerBuildCommandServiceName(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "my-awesome-service",
		Path: "services/my-awesome-service",
	}

	cmd := docker.GenerateDockerBuildCommand(service, "image:tag", "Dockerfile")

	// Should contain SERVICE_NAME build arg with the service path (not name)
	foundServiceName := false
	for i, arg := range cmd {
		if arg == "--build-arg" && i+1 < len(cmd) {
			if cmd[i+1] == "SERVICE_NAME=services/my-awesome-service" {
				foundServiceName = true
				break
			}
		}
	}
	require.True(t, foundServiceName, "SERVICE_NAME build arg not found")
}

func TestGenerateDockerBuildCommandWithEnvVars(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "myservice",
		Path: "services/myservice",
		EnvVars: []serviceinfo.EnvVars{
			{Name: "DATABASE_URL", Value: "postgres://localhost/db"},
			{Name: "API_KEY", Value: "secret123"},
		},
	}

	cmd := docker.GenerateDockerBuildCommand(service, "image:tag", "Dockerfile")

	// Should contain env vars as build arg
	foundEnvArg := false
	for i, arg := range cmd {
		if arg == "--build-arg" && i+1 < len(cmd) {
			val := cmd[i+1]
			if val != "SERVICE_NAME=services/myservice" {
				require.Contains(t, val, "DATABASE_URL=")
				require.Contains(t, val, "API_KEY=")
				foundEnvArg = true
				break
			}
		}
	}
	require.True(t, foundEnvArg, "env vars build arg not found")
}

func TestGenerateDockerBuildCommandNoEnvVars(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:    "myservice",
		EnvVars: []serviceinfo.EnvVars{},
	}

	cmd := docker.GenerateDockerBuildCommand(service, "image:tag", "Dockerfile")

	// Should only have SERVICE_NAME build arg
	buildArgCount := 0
	for _, arg := range cmd {
		if arg == "--build-arg" {
			buildArgCount++
		}
	}
	require.Equal(t, 1, buildArgCount, "should only have one --build-arg for SERVICE_NAME")
}

func TestGenerateDockerBuildCommandDockerfilePath(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "myservice",
	}

	templatePath := "./_templates/custom/Dockerfile.prod"
	cmd := docker.GenerateDockerBuildCommand(service, "image:tag", templatePath)

	// Should contain -f flag with correct path
	for i, arg := range cmd {
		if arg == "-f" && i+1 < len(cmd) {
			require.Equal(t, templatePath, cmd[i+1])
			return
		}
	}
	t.Fatal("-f flag not found")
}

func TestGenerateDockerBuildCommandContext(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "myservice",
	}

	cmd := docker.GenerateDockerBuildCommand(service, "image:tag", "Dockerfile")

	// Last argument should be build context "."
	require.Equal(t, ".", cmd[len(cmd)-1])
}
