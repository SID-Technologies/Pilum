package homebrew_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/homebrew"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/stretchr/testify/require"
)

func TestGenerateArchiveCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		service   serviceinfo.ServiceInfo
		tag       string
		outputDir string
	}{
		{
			name:      "basic archive command",
			service:   serviceinfo.ServiceInfo{Name: "myapp"},
			tag:       "v1.0.0",
			outputDir: "dist",
		},
		{
			name:      "different name and tag",
			service:   serviceinfo.ServiceInfo{Name: "cli-tool"},
			tag:       "v2.5.0",
			outputDir: "build",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := homebrew.GenerateArchiveCommand(tt.service, tt.tag, tt.outputDir)

			require.NotEmpty(t, result)

			// Should cd to output directory
			require.Contains(t, result, "cd "+tt.outputDir)

			// Should contain the file pattern
			pattern := tt.service.Name + "_" + tt.tag + "_*"
			require.Contains(t, result, pattern)

			// Should create tar.gz
			require.Contains(t, result, "tar -czf")
			require.Contains(t, result, ".tar.gz")

			// Should remove original binary after archiving
			require.Contains(t, result, "rm \"$f\"")
		})
	}
}

func TestGenerateChecksumCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		outputDir string
	}{
		{
			name:      "dist directory",
			outputDir: "dist",
		},
		{
			name:      "custom directory",
			outputDir: "/tmp/build",
		},
		{
			name:      "relative path",
			outputDir: "./output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := homebrew.GenerateChecksumCommand(tt.outputDir)

			require.NotEmpty(t, result)

			// Should cd to output directory
			require.Contains(t, result, "cd "+tt.outputDir)

			// Should use shasum with SHA256
			require.Contains(t, result, "shasum -a 256")

			// Should match tar.gz files
			require.Contains(t, result, "*.tar.gz")

			// Should output to checksums.txt
			require.Contains(t, result, "checksums.txt")
		})
	}
}
