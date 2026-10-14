package sysinfo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDockerResources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want DockerResources
	}{
		{"docker desktop", "10 8323563520\n", DockerResources{CPUs: 10, MemoryMB: 7937}},
		{"empty", "", DockerResources{}},
		{"garbage", "abc def", DockerResources{}},
		{"extra fields", "4 1024 extra", DockerResources{}},
		{"zero cpus", "0 1073741824", DockerResources{MemoryMB: 1024}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, parseDockerResources(tt.in))
		})
	}
}

func TestBuildxCacheSupport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inspect      string
		driverStatus string
		want         bool
	}{
		{"docker-container", "Name: ci\nDriver:        docker-container\n", "", true},
		{"remote", "Driver: remote\n", "", true},
		{"docker driver, classic store", "Name: desktop-linux\nDriver: docker\n", `[["Backing Filesystem","extfs"]]`, false},
		{"docker driver, containerd store", "Driver: docker\n", `[["driver-type","io.containerd.snapshotter.v1"]]`, true},
		{"unknown output", "something else", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, reason := buildxCacheSupport(tt.inspect, tt.driverStatus)
			require.Equal(t, tt.want, got)
			if !tt.want {
				require.NotEmpty(t, reason)
			}
		})
	}
}
