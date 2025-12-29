package build_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/build"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/stretchr/testify/require"
)

func TestGenerateBuildCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		service          serviceinfo.ServiceInfo
		registry         string
		tag              string
		expectedCmd      []string
		expectedImage    string
		expectedEmptyCmd bool
	}{
		{
			name: "basic build command",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build -o ./dist/myservice",
				},
			},
			registry:      "",
			tag:           "v1.0.0",
			expectedCmd:   []string{"/bin/sh", "-c", "go build -o ./dist/myservice"},
			expectedImage: "myservice:v1.0.0",
		},
		{
			name: "build command with flags",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build -o ./dist/myservice",
					Flags: []serviceinfo.BuildFlag{
						{Name: "ldflags", Values: []string{"-s", "-w"}},
					},
				},
			},
			registry:      "",
			tag:           "v1.0.0",
			expectedCmd:   []string{"/bin/sh", "-c", "go build -o ./dist/myservice -ldflags='-s -w'"},
			expectedImage: "myservice:v1.0.0",
		},
		{
			name: "build command with multiple flags",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build -o ./dist/myservice",
					Flags: []serviceinfo.BuildFlag{
						{Name: "ldflags", Values: []string{"-s", "-w"}},
						{Name: "gcflags", Values: []string{"-N", "-l"}},
					},
				},
			},
			registry:      "",
			tag:           "v1.0.0",
			expectedCmd:   []string{"/bin/sh", "-c", "go build -o ./dist/myservice -ldflags='-s -w' -gcflags='-N -l'"},
			expectedImage: "myservice:v1.0.0",
		},
		{
			name: "build command with empty flag values",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build -o ./dist/myservice",
					Flags: []serviceinfo.BuildFlag{
						{Name: "ldflags", Values: []string{}},
					},
				},
			},
			registry:      "",
			tag:           "v1.0.0",
			expectedCmd:   []string{"/bin/sh", "-c", "go build -o ./dist/myservice"},
			expectedImage: "myservice:v1.0.0",
		},
		{
			name: "empty build command returns nil",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "",
				},
			},
			registry:         "",
			tag:              "v1.0.0",
			expectedEmptyCmd: true,
			expectedImage:    "",
		},
		{
			name: "with explicit registry",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build -o ./dist/myservice",
				},
			},
			registry:      "gcr.io/myproject",
			tag:           "v1.0.0",
			expectedCmd:   []string{"/bin/sh", "-c", "go build -o ./dist/myservice"},
			expectedImage: "gcr.io/myproject/myservice:v1.0.0",
		},
		{
			name: "GCP provider without registry",
			service: serviceinfo.ServiceInfo{
				Name:     "myservice",
				Provider: "gcp",
				Region:   "us-central1",
				Project:  "my-project",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build -o ./dist/myservice",
				},
			},
			registry:      "",
			tag:           "v1.0.0",
			expectedCmd:   []string{"/bin/sh", "-c", "go build -o ./dist/myservice"},
			expectedImage: "us-central1-docker.pkg.dev/my-project/my-project/myservice:v1.0.0",
		},
		{
			name: "empty tag defaults to latest",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build -o ./dist/myservice",
				},
			},
			registry:      "",
			tag:           "",
			expectedCmd:   []string{"/bin/sh", "-c", "go build -o ./dist/myservice"},
			expectedImage: "myservice:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd, imageName := build.GenerateBuildCommand(tt.service, tt.registry, tt.tag)

			if tt.expectedEmptyCmd {
				require.Nil(t, cmd)
				require.Empty(t, imageName)
			} else {
				require.Equal(t, tt.expectedCmd, cmd)
				require.Equal(t, tt.expectedImage, imageName)
			}
		})
	}
}

func TestGenerateBuildCommandString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		service  serviceinfo.ServiceInfo
		expected string
	}{
		{
			name: "basic build command string",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build -o ./dist/myservice",
				},
			},
			expected: "go build -o ./dist/myservice",
		},
		{
			name: "build command string with flags",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build -o ./dist/myservice",
					Flags: []serviceinfo.BuildFlag{
						{Name: "ldflags", Values: []string{"-s", "-w"}},
					},
				},
			},
			expected: "go build -o ./dist/myservice -ldflags='-s -w'",
		},
		{
			name: "empty build command returns empty string",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "",
				},
			},
			expected: "",
		},
		{
			name: "flags with empty values are skipped",
			service: serviceinfo.ServiceInfo{
				Name: "myservice",
				BuildConfig: serviceinfo.BuildConfig{
					Cmd: "go build",
					Flags: []serviceinfo.BuildFlag{
						{Name: "ldflags", Values: []string{}},
						{Name: "gcflags", Values: []string{"-N"}},
					},
				},
			},
			expected: "go build -gcflags='-N'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := build.GenerateBuildCommandString(tt.service)
			require.Equal(t, tt.expected, result)
		})
	}
}
