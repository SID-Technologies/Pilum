package cmd

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/webhook"

	"github.com/spf13/viper"
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

func TestDeploymentOptionsToPipelineOptions(t *testing.T) {
	t.Parallel()

	opts := deploymentOptions{
		Tag:         "v1.0.0",
		Debug:       true,
		Timeout:     120,
		Retries:     5,
		DryRun:      true,
		MaxWorkers:  4,
		OnlyTags:    []string{"build", "test"},
		ExcludeTags: []string{"deploy"},
	}

	pipelineOpts := opts.toPipelineOptions()

	require.Equal(t, opts.Tag, pipelineOpts.Tag)
	require.Equal(t, opts.Debug, pipelineOpts.Debug)
	require.Equal(t, opts.Timeout, pipelineOpts.Timeout)
	require.Equal(t, opts.Retries, pipelineOpts.Retries)
	require.Equal(t, opts.DryRun, pipelineOpts.DryRun)
	require.Equal(t, opts.MaxWorkers, pipelineOpts.MaxWorkers)
	require.Equal(t, opts.OnlyTags, pipelineOpts.OnlyTags)
	require.Equal(t, opts.ExcludeTags, pipelineOpts.ExcludeTags)
}

func TestDeploymentOptionsToPipelineOptionsDefaults(t *testing.T) {
	t.Parallel()

	opts := deploymentOptions{}
	pipelineOpts := opts.toPipelineOptions()

	require.Equal(t, "", pipelineOpts.Tag)
	require.False(t, pipelineOpts.Debug)
	require.Equal(t, 0, pipelineOpts.Timeout)
	require.Equal(t, 0, pipelineOpts.Retries)
	require.False(t, pipelineOpts.DryRun)
	require.Equal(t, 0, pipelineOpts.MaxWorkers)
	require.Nil(t, pipelineOpts.OnlyTags)
	require.Nil(t, pipelineOpts.ExcludeTags)
}

// --- loadWebhookConfigs tests ---
// These use Viper global state, so they must NOT run in parallel.

func TestLoadWebhookConfigs_ValidConfig(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("webhooks", []any{
		map[string]any{
			"url":    "https://hooks.example.com/deploy",
			"events": []any{"start", "success", "failure"},
		},
	})

	configs := loadWebhookConfigs()

	require.Len(t, configs, 1)
	require.Equal(t, "https://hooks.example.com/deploy", configs[0].URL)
	require.Equal(t, []webhook.Event{webhook.EventStart, webhook.EventSuccess, webhook.EventFailure}, configs[0].Events)
}

func TestLoadWebhookConfigs_MultipleWebhooks(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("webhooks", []any{
		map[string]any{
			"url":    "https://slack.example.com/webhook",
			"events": []any{"failure"},
		},
		map[string]any{
			"url":    "https://pagerduty.example.com/alert",
			"events": []any{"start", "failure"},
		},
	})

	configs := loadWebhookConfigs()

	require.Len(t, configs, 2)
	require.Equal(t, "https://slack.example.com/webhook", configs[0].URL)
	require.Equal(t, []webhook.Event{webhook.EventFailure}, configs[0].Events)
	require.Equal(t, "https://pagerduty.example.com/alert", configs[1].URL)
	require.Len(t, configs[1].Events, 2)
}

func TestLoadWebhookConfigs_NoEvents(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("webhooks", []any{
		map[string]any{
			"url": "https://hooks.example.com/all",
		},
	})

	configs := loadWebhookConfigs()

	require.Len(t, configs, 1)
	require.Equal(t, "https://hooks.example.com/all", configs[0].URL)
	require.Nil(t, configs[0].Events) // No events = match all (tested in webhook package)
}

func TestLoadWebhookConfigs_NoWebhooksKey(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	// Don't set any webhooks key
	configs := loadWebhookConfigs()

	require.Nil(t, configs)
}

func TestLoadWebhookConfigs_EmptyArray(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("webhooks", []any{})

	configs := loadWebhookConfigs()

	require.Nil(t, configs)
}

func TestLoadWebhookConfigs_SkipsMissingURL(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("webhooks", []any{
		map[string]any{
			"events": []any{"failure"},
		},
		map[string]any{
			"url":    "",
			"events": []any{"failure"},
		},
		map[string]any{
			"url":    "https://valid.example.com/hook",
			"events": []any{"success"},
		},
	})

	configs := loadWebhookConfigs()

	require.Len(t, configs, 1)
	require.Equal(t, "https://valid.example.com/hook", configs[0].URL)
}

func TestLoadWebhookConfigs_InvalidType(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	// Set webhooks to a non-array value
	viper.Set("webhooks", "not an array")

	configs := loadWebhookConfigs()

	require.Nil(t, configs)
}

func TestLoadWebhookConfigs_SkipsInvalidEntries(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("webhooks", []any{
		"not a map",
		42,
		map[string]any{
			"url":    "https://valid.example.com/hook",
			"events": []any{"failure"},
		},
	})

	configs := loadWebhookConfigs()

	require.Len(t, configs, 1)
	require.Equal(t, "https://valid.example.com/hook", configs[0].URL)
}
