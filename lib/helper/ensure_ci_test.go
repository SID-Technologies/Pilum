package helper_test

import (
	"os"
	"testing"

	"github.com/sid-technologies/pilum/lib/helper"
	"github.com/stretchr/testify/require"
)

func TestEnsureCIEnvironment(t *testing.T) {
	// Note: Not parallel because we modify environment variables

	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "CI env var set to true",
			envValue: "true",
			expected: true,
		},
		{
			name:     "CI env var set to 1",
			envValue: "1",
			expected: true,
		},
		{
			name:     "CI env var set to any non-empty value",
			envValue: "yes",
			expected: true,
		},
		{
			name:     "CI env var empty",
			envValue: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current value
			originalCI := os.Getenv("CI")
			defer func() {
				if originalCI != "" {
					os.Setenv("CI", originalCI)
				} else {
					os.Unsetenv("CI")
				}
			}()

			if tt.envValue != "" {
				os.Setenv("CI", tt.envValue)
			} else {
				os.Unsetenv("CI")
			}

			result := helper.EnsureCIEnvironment()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureCIEnvironmentUnset(t *testing.T) {
	// Save current value
	originalCI := os.Getenv("CI")
	defer func() {
		if originalCI != "" {
			os.Setenv("CI", originalCI)
		} else {
			os.Unsetenv("CI")
		}
	}()

	// Ensure CI is unset
	os.Unsetenv("CI")

	result := helper.EnsureCIEnvironment()
	require.False(t, result, "should return false when CI env var is not set")
}
