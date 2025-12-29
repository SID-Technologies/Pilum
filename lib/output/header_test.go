package output_test

import (
	"os"
	"testing"

	"github.com/sid-technologies/pilum/lib/output"
	"github.com/stretchr/testify/require"
)

func TestPrintBanner(t *testing.T) {
	t.Parallel()

	result := output.PrintBanner("1.0.0")

	require.NotEmpty(t, result)
	require.Contains(t, result, "1.0.0")
}

func TestPrintBannerContainsPilum(t *testing.T) {
	t.Parallel()

	result := output.PrintBanner("v2.0.0")

	// Banner should contain some recognizable pattern
	require.NotEmpty(t, result)
	require.Contains(t, result, "Version")
}

func TestPrintBannerWithDifferentVersions(t *testing.T) {
	t.Parallel()

	versions := []string{
		"1.0.0",
		"v1.0.0",
		"2.3.4-beta",
		"0.0.1",
		"latest",
	}

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			result := output.PrintBanner(version)
			require.Contains(t, result, version)
		})
	}
}

func TestPrintBannerNoColor(t *testing.T) {
	// Save current state
	originalNoColor := os.Getenv("NO_COLOR")
	originalTerm := os.Getenv("TERM")
	defer func() {
		if originalNoColor != "" {
			os.Setenv("NO_COLOR", originalNoColor)
		} else {
			os.Unsetenv("NO_COLOR")
		}
		if originalTerm != "" {
			os.Setenv("TERM", originalTerm)
		} else {
			os.Unsetenv("TERM")
		}
	}()

	// Set NO_COLOR to disable colors
	os.Setenv("NO_COLOR", "1")

	result := output.PrintBanner("1.0.0")
	require.NotEmpty(t, result)
	require.Contains(t, result, "1.0.0")
}

func TestPrintBannerDumbTerminal(t *testing.T) {
	// Save current state
	originalTerm := os.Getenv("TERM")
	defer func() {
		if originalTerm != "" {
			os.Setenv("TERM", originalTerm)
		} else {
			os.Unsetenv("TERM")
		}
	}()

	// Set TERM to dumb
	os.Setenv("TERM", "dumb")

	result := output.PrintBanner("1.0.0")
	require.NotEmpty(t, result)
	require.Contains(t, result, "1.0.0")
}
