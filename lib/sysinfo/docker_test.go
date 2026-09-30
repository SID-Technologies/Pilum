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
