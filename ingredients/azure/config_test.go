package azure_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/azure"

	"github.com/stretchr/testify/require"
)

func TestParseContainerAppConfigEmpty(t *testing.T) {
	t.Parallel()

	cfg := azure.ParseContainerAppConfig(map[string]any{})

	require.Empty(t, cfg.Environment)
	require.Equal(t, -1, cfg.MinReplicas)
	require.Equal(t, -1, cfg.MaxReplicas)
	require.Empty(t, cfg.CPU)
	require.Empty(t, cfg.Memory)
	require.Equal(t, -1, cfg.IngressPort)
	require.False(t, cfg.IngressExternal) // default false when no config
}

func TestParseContainerAppConfigFull(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"container_app": map[string]any{
			"environment":      "my-env",
			"min_replicas":     0,
			"max_replicas":     10,
			"cpu":              "0.5",
			"memory":           "1Gi",
			"ingress_port":     8080,
			"ingress_external": true,
		},
	}

	cfg := azure.ParseContainerAppConfig(config)

	require.Equal(t, "my-env", cfg.Environment)
	require.Equal(t, 0, cfg.MinReplicas)
	require.Equal(t, 10, cfg.MaxReplicas)
	require.Equal(t, "0.5", cfg.CPU)
	require.Equal(t, "1Gi", cfg.Memory)
	require.Equal(t, 8080, cfg.IngressPort)
	require.True(t, cfg.IngressExternal)
}

func TestParseContainerAppConfigPartial(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"container_app": map[string]any{
			"environment":  "staging",
			"min_replicas": 1,
		},
	}

	cfg := azure.ParseContainerAppConfig(config)

	require.Equal(t, "staging", cfg.Environment)
	require.Equal(t, 1, cfg.MinReplicas)
	require.Equal(t, -1, cfg.MaxReplicas)
	require.Empty(t, cfg.CPU)
	require.Empty(t, cfg.Memory)
	require.Equal(t, -1, cfg.IngressPort)
	require.True(t, cfg.IngressExternal) // default true when config section exists
}

func TestParseContainerAppConfigNil(t *testing.T) {
	t.Parallel()

	cfg := azure.ParseContainerAppConfig(nil)

	require.Equal(t, -1, cfg.MinReplicas)
	require.Equal(t, -1, cfg.MaxReplicas)
	require.Equal(t, -1, cfg.IngressPort)
}
