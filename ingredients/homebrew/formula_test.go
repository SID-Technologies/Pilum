package homebrew_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/homebrew"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/stretchr/testify/require"
)

func TestGenerateFormulaCommand(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:        "myapp",
		Description: "A test application",
		License:     "MIT",
		HomebrewConfig: serviceinfo.HomebrewConfig{
			ProjectURL: "https://github.com/org/myapp",
		},
	}

	result := homebrew.GenerateFormulaCommand(service, "v1.0.0", "dist", "dist/myapp.rb")

	require.NotEmpty(t, result)

	// Should contain class definition with capitalized name
	require.Contains(t, result, "class Myapp < Formula")

	// Should contain description
	require.Contains(t, result, `desc "A test application"`)

	// Should contain homepage
	require.Contains(t, result, `homepage "https://github.com/org/myapp"`)

	// Should contain version (without 'v' prefix)
	require.Contains(t, result, `version "1.0.0"`)

	// Should contain license
	require.Contains(t, result, `license "MIT"`)

	// Should contain checksum lookups
	require.Contains(t, result, "DARWIN_ARM64_SHA")
	require.Contains(t, result, "DARWIN_AMD64_SHA")
	require.Contains(t, result, "LINUX_ARM64_SHA")
	require.Contains(t, result, "LINUX_AMD64_SHA")

	// Should contain platform-specific blocks
	require.Contains(t, result, "on_macos do")
	require.Contains(t, result, "on_linux do")

	// Should contain download URLs
	require.Contains(t, result, "https://github.com/org/myapp/releases/download/v1.0.0")

	// Should contain install block
	require.Contains(t, result, "def install")

	// Should contain test block
	require.Contains(t, result, "test do")
}

func TestGenerateFormulaCommandVersionStripping(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:        "myapp",
		Description: "Test",
		License:     "Apache-2.0",
		HomebrewConfig: serviceinfo.HomebrewConfig{
			ProjectURL: "https://github.com/org/myapp",
		},
	}

	// Tag with 'v' prefix
	result := homebrew.GenerateFormulaCommand(service, "v2.3.4", "dist", "dist/myapp.rb")

	// Version field should NOT have 'v' prefix
	require.Contains(t, result, `version "2.3.4"`)

	// But URLs should still use full tag
	require.Contains(t, result, "releases/download/v2.3.4")
}

func TestGenerateFormulaCommandOutputPath(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:        "cli",
		Description: "CLI tool",
		License:     "MIT",
		HomebrewConfig: serviceinfo.HomebrewConfig{
			ProjectURL: "https://github.com/org/cli",
		},
	}

	result := homebrew.GenerateFormulaCommand(service, "v1.0.0", "build", "build/cli.rb")

	// Should output to the specified formula path
	require.Contains(t, result, "cat > build/cli.rb")

	// Should reference correct output directory for checksums
	require.Contains(t, result, "build/checksums.txt")
}

func TestGenerateFormulaCommandArchitectureBlocks(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:        "myapp",
		Description: "Test",
		License:     "MIT",
		HomebrewConfig: serviceinfo.HomebrewConfig{
			ProjectURL: "https://github.com/org/myapp",
		},
	}

	result := homebrew.GenerateFormulaCommand(service, "v1.0.0", "dist", "dist/myapp.rb")

	// Should contain ARM detection
	require.Contains(t, result, "Hardware::CPU.arm?")

	// Should contain all platform variants
	require.Contains(t, result, "darwin_arm64.tar.gz")
	require.Contains(t, result, "darwin_amd64.tar.gz")
	require.Contains(t, result, "linux_arm64.tar.gz")
	require.Contains(t, result, "linux_amd64.tar.gz")
}

func TestGenerateFormulaCommandInstallBlock(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:        "mytool",
		Description: "Test",
		License:     "MIT",
		HomebrewConfig: serviceinfo.HomebrewConfig{
			ProjectURL: "https://github.com/org/mytool",
		},
	}

	result := homebrew.GenerateFormulaCommand(service, "v1.0.0", "dist", "dist/mytool.rb")

	// Should install binary with correct name
	require.Contains(t, result, `bin.install Dir["mytool_*"].first => "mytool"`)
}

func TestGenerateFormulaCommandTestBlock(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name:        "myapp",
		Description: "Test",
		License:     "MIT",
		HomebrewConfig: serviceinfo.HomebrewConfig{
			ProjectURL: "https://github.com/org/myapp",
		},
	}

	result := homebrew.GenerateFormulaCommand(service, "v1.0.0", "dist", "dist/myapp.rb")

	// Should test with --version flag
	require.Contains(t, result, `system "#{bin}/myapp", "--version"`)
}
