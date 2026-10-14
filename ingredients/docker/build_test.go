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
			expectedLen:  11, // includes --platform linux/amd64
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
			expectedLen:  13, // 2 more for --build-arg with env vars
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
			{Name: "ALLOWED_ORIGINS", Value: "https://a.dev,https://b.dev"},
		},
	}

	cmd := docker.GenerateDockerBuildCommand(service, "image:tag", "Dockerfile")

	// One --build-arg per variable, sorted, with commas kept inside a value.
	var buildArgs []string
	for i, arg := range cmd {
		if arg == "--build-arg" && i+1 < len(cmd) {
			buildArgs = append(buildArgs, cmd[i+1])
		}
	}
	require.Equal(t, []string{
		"SERVICE_NAME=services/myservice",
		"ALLOWED_ORIGINS=https://a.dev,https://b.dev",
		"API_KEY=secret123",
		"DATABASE_URL=postgres://localhost/db",
	}, buildArgs)
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

// collectTags returns every image ref passed via -t, in order.
func collectTags(cmd []string) []string {
	var tags []string
	for i, arg := range cmd {
		if arg == "-t" && i+1 < len(cmd) {
			tags = append(tags, cmd[i+1])
		}
	}
	return tags
}

func TestGenerateDockerBuildCommandAddsLatestAlias(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{Name: "myservice"}

	cmd := docker.GenerateDockerBuildCommand(service,
		"us-central1-docker.pkg.dev/proj/repo/mcp-controller:v1.0.0", "Dockerfile")

	require.Equal(t, []string{
		"us-central1-docker.pkg.dev/proj/repo/mcp-controller:v1.0.0",
		"us-central1-docker.pkg.dev/proj/repo/mcp-controller:latest",
	}, collectTags(cmd))
}

func TestGenerateDockerBuildCommandNoAliasWhenAlreadyLatest(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{Name: "myservice"}

	cmd := docker.GenerateDockerBuildCommand(service,
		"gcr.io/project/myservice:latest", "Dockerfile")

	require.Equal(t, []string{"gcr.io/project/myservice:latest"}, collectTags(cmd))
}

func TestGenerateDockerBuildCommandNoAliasWhenUntagged(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{Name: "myservice"}

	// The only ":" is the registry port — not a tag separator.
	cmd := docker.GenerateDockerBuildCommand(service,
		"localhost:5000/myservice", "Dockerfile")

	require.Equal(t, []string{"localhost:5000/myservice"}, collectTags(cmd))
}

func TestGenerateDockerBuildCommandAliasWithRegistryPort(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{Name: "myservice"}

	cmd := docker.GenerateDockerBuildCommand(service,
		"localhost:5000/myservice:v2.1.0", "Dockerfile")

	require.Equal(t, []string{
		"localhost:5000/myservice:v2.1.0",
		"localhost:5000/myservice:latest",
	}, collectTags(cmd))
}

func TestGenerateDockerBuildCommandPlatform(t *testing.T) {
	t.Parallel()

	cmd := docker.GenerateDockerBuildCommand(serviceinfo.ServiceInfo{Name: "svc"}, "img:v1", "Dockerfile")
	require.Equal(t, []string{"docker", "build", "--platform", "linux/amd64", "-t", "img:v1"}, cmd[:6])

	arm := serviceinfo.ServiceInfo{Name: "svc", BuildConfig: serviceinfo.BuildConfig{Platform: "linux/arm64"}}
	cmd = docker.GenerateDockerBuildCommand(arm, "img:v1", "Dockerfile")
	require.Equal(t, []string{"docker", "build", "--platform", "linux/arm64"}, cmd[:4])
}

func TestGenerateDockerBuildCommandRegistryCache(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{Name: "svc", BuildConfig: serviceinfo.BuildConfig{DockerCache: true}}
	cmd := docker.GenerateDockerBuildCommand(svc, "us-docker.pkg.dev/p/repo/svc:v1", "Dockerfile")

	require.Equal(t, []string{
		"docker", "buildx", "build",
		"--cache-from", "type=registry,ref=us-docker.pkg.dev/p/repo/svc:buildcache",
		"--cache-to", "type=registry,ref=us-docker.pkg.dev/p/repo/svc:buildcache,mode=max,ignore-error=true",
		"--load",
		"--platform", "linux/amd64",
		"-t", "us-docker.pkg.dev/p/repo/svc:v1",
	}, cmd[:12])

	// A registry port is not a tag.
	cmd = docker.GenerateDockerBuildCommand(svc, "localhost:5000/svc", "Dockerfile")
	require.Contains(t, cmd, "type=registry,ref=localhost:5000/svc:buildcache")
}
