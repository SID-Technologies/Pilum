package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCommaSeparated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single value",
			input:    "build",
			expected: []string{"build"},
		},
		{
			name:     "multiple values",
			input:    "build,test,deploy",
			expected: []string{"build", "test", "deploy"},
		},
		{
			name:     "values with whitespace",
			input:    "build , test , deploy",
			expected: []string{"build", "test", "deploy"},
		},
		{
			name:     "leading and trailing whitespace",
			input:    "  build  ,  test  ",
			expected: []string{"build", "test"},
		},
		{
			name:     "trailing comma",
			input:    "build,test,",
			expected: []string{"build", "test"},
		},
		{
			name:     "leading comma",
			input:    ",build,test",
			expected: []string{"build", "test"},
		},
		{
			name:     "multiple commas",
			input:    "build,,test",
			expected: []string{"build", "test"},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: []string{},
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: []string{},
		},
		{
			name:     "whitespace with commas",
			input:    "  ,  ,  ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseCommaSeparated(tt.input)

			if tt.expected == nil {
				require.Nil(t, result)
			} else if len(tt.expected) == 0 {
				require.Empty(t, result)
			} else {
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDeploymentOptionsToRunnerOptions(t *testing.T) {
	t.Parallel()

	opts := deploymentOptions{
		Tag:         "v1.0.0",
		Debug:       true,
		Timeout:     120,
		Retries:     5,
		DryRun:      true,
		RecipePath:  "./custom-recipes",
		MaxWorkers:  4,
		OnlyTags:    []string{"build", "test"},
		ExcludeTags: []string{"deploy"},
	}

	runnerOpts := opts.toRunnerOptions()

	require.Equal(t, opts.Tag, runnerOpts.Tag)
	require.Equal(t, opts.Debug, runnerOpts.Debug)
	require.Equal(t, opts.Timeout, runnerOpts.Timeout)
	require.Equal(t, opts.Retries, runnerOpts.Retries)
	require.Equal(t, opts.DryRun, runnerOpts.DryRun)
	require.Equal(t, opts.MaxWorkers, runnerOpts.MaxWorkers)
	require.Equal(t, opts.OnlyTags, runnerOpts.OnlyTags)
	require.Equal(t, opts.ExcludeTags, runnerOpts.ExcludeTags)
}

func TestDeploymentOptionsToRunnerOptionsDefaults(t *testing.T) {
	t.Parallel()

	opts := deploymentOptions{}
	runnerOpts := opts.toRunnerOptions()

	require.Equal(t, "", runnerOpts.Tag)
	require.False(t, runnerOpts.Debug)
	require.Equal(t, 0, runnerOpts.Timeout)
	require.Equal(t, 0, runnerOpts.Retries)
	require.False(t, runnerOpts.DryRun)
	require.Equal(t, 0, runnerOpts.MaxWorkers)
	require.Nil(t, runnerOpts.OnlyTags)
	require.Nil(t, runnerOpts.ExcludeTags)
}
