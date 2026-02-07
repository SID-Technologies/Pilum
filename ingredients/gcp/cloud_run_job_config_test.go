package gcp_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"

	"github.com/stretchr/testify/require"
)

func TestParseCloudRunJobConfigEmpty(t *testing.T) {
	t.Parallel()

	cfg := gcp.ParseCloudRunJobConfig(map[string]any{})

	require.Equal(t, "600s", cfg.Timeout)
	require.Equal(t, -1, cfg.MaxRetries)
	require.Equal(t, -1, cfg.Parallelism)
	require.Equal(t, -1, cfg.TaskCount)
	require.Empty(t, cfg.CPU)
	require.Empty(t, cfg.Memory)
	require.Empty(t, cfg.ServiceAccount)
	require.False(t, cfg.ExecuteOnDeploy)
}

func TestParseCloudRunJobConfigFull(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"job": map[string]any{
			"timeout":           "300s",
			"max_retries":       3,
			"parallelism":       2,
			"task_count":        5,
			"cpu":               "2",
			"memory":            "2Gi",
			"service_account":   "sa@project.iam.gserviceaccount.com",
			"execute_on_deploy": true,
		},
	}

	cfg := gcp.ParseCloudRunJobConfig(config)

	require.Equal(t, "300s", cfg.Timeout)
	require.Equal(t, 3, cfg.MaxRetries)
	require.Equal(t, 2, cfg.Parallelism)
	require.Equal(t, 5, cfg.TaskCount)
	require.Equal(t, "2", cfg.CPU)
	require.Equal(t, "2Gi", cfg.Memory)
	require.Equal(t, "sa@project.iam.gserviceaccount.com", cfg.ServiceAccount)
	require.True(t, cfg.ExecuteOnDeploy)
}

func TestParseCloudRunJobConfigPartial(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"job": map[string]any{
			"timeout":     "120s",
			"max_retries": 1,
		},
	}

	cfg := gcp.ParseCloudRunJobConfig(config)

	require.Equal(t, "120s", cfg.Timeout)
	require.Equal(t, 1, cfg.MaxRetries)
	require.Equal(t, -1, cfg.Parallelism)
	require.Equal(t, -1, cfg.TaskCount)
	require.Empty(t, cfg.CPU)
	require.Empty(t, cfg.Memory)
	require.Empty(t, cfg.ServiceAccount)
	require.False(t, cfg.ExecuteOnDeploy)
}

func TestParseCloudRunJobConfigNilConfig(t *testing.T) {
	t.Parallel()

	cfg := gcp.ParseCloudRunJobConfig(nil)

	require.Equal(t, "600s", cfg.Timeout)
	require.Equal(t, -1, cfg.MaxRetries)
	require.False(t, cfg.ExecuteOnDeploy)
}
