package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sid-technologies/pilum/lib/cache"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestClearSingleService_Exists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	c := make(cache.Cache)
	cache.Update(c, "api", "abc123", "v1.0.0")
	cache.Update(c, "worker", "def456", "v1.0.0")
	require.NoError(t, cache.Save(tmpDir, c))

	result, err := clearSingleService(tmpDir, "api")

	require.NoError(t, err)
	require.NotNil(t, result)

	m, ok := result.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, m["success"])
	require.Equal(t, 1, m["cleared"])
	require.Equal(t, "api", m["service"])

	// Verify api is gone, worker remains
	loaded, err := cache.Load(tmpDir)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	require.NotNil(t, loaded["worker"])
}

func TestClearSingleService_NotExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	c := make(cache.Cache)
	cache.Update(c, "api", "abc123", "v1.0.0")
	require.NoError(t, cache.Save(tmpDir, c))

	result, err := clearSingleService(tmpDir, "nonexistent")

	require.NoError(t, err)
	require.NotNil(t, result)

	m, ok := result.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, m["success"])
	require.Equal(t, 0, m["cleared"])

	// Original entry should still be there
	loaded, err := cache.Load(tmpDir)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
}

func TestClearAllCache_WithEntries(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	c := make(cache.Cache)
	cache.Update(c, "api", "abc123", "v1.0.0")
	cache.Update(c, "worker", "def456", "v1.0.0")
	require.NoError(t, cache.Save(tmpDir, c))

	result, err := clearAllCache(tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)

	m, ok := result.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, m["success"])
	require.Equal(t, 2, m["cleared"])

	// File should be gone
	_, statErr := os.Stat(cache.FilePath(tmpDir))
	require.True(t, os.IsNotExist(statErr))
}

func TestClearAllCache_Empty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	result, err := clearAllCache(tmpDir)

	require.NoError(t, err)
	require.NotNil(t, result)

	m, ok := result.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, m["success"])
	require.Equal(t, 0, m["cleared"])
}

func TestClearAllCache_Idempotent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	c := make(cache.Cache)
	cache.Update(c, "api", "abc123", "v1.0.0")
	require.NoError(t, cache.Save(tmpDir, c))

	// Clear twice — second should not error
	_, err := clearAllCache(tmpDir)
	require.NoError(t, err)

	_, err = clearAllCache(tmpDir)
	require.NoError(t, err)
}

func TestFormatAge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{name: "seconds", duration: 30 * time.Second, expected: "30s ago"},
		{name: "minutes", duration: 5 * time.Minute, expected: "5m ago"},
		{name: "hours", duration: 3 * time.Hour, expected: "3h ago"},
		{name: "days", duration: 48 * time.Hour, expected: "2d ago"},
		{name: "zero", duration: 0, expected: "0s ago"},
		{name: "just under a minute", duration: 59 * time.Second, expected: "59s ago"},
		{name: "exactly one minute", duration: time.Minute, expected: "1m ago"},
		{name: "just under an hour", duration: 59 * time.Minute, expected: "59m ago"},
		{name: "exactly one hour", duration: time.Hour, expected: "1h ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, formatAge(tt.duration))
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{name: "zero", bytes: 0, expected: "0 B"},
		{name: "bytes", bytes: 512, expected: "512 B"},
		{name: "kilobytes", bytes: 2048, expected: "2.0 KB"},
		{name: "megabytes", bytes: 1048576, expected: "1.0 MB"},
		{name: "just under KB", bytes: 1023, expected: "1023 B"},
		{name: "exactly 1KB", bytes: 1024, expected: "1.0 KB"},
		{name: "fractional KB", bytes: 1536, expected: "1.5 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, formatBytes(tt.bytes))
		})
	}
}

func TestCacheCmdStructure(t *testing.T) {
	t.Parallel()

	cmd := CacheCmd()

	require.Equal(t, "cache", cmd.Use)
	require.Len(t, cmd.Commands(), 2)

	// Verify subcommands exist
	names := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		names[sub.Use] = true
	}
	require.True(t, names["status"])
	require.True(t, names["clear"])
}

func TestCacheClearCmdHasServiceFlag(t *testing.T) {
	t.Parallel()

	cmd := CacheCmd()

	var clearCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "clear" {
			clearCmd = sub
			break
		}
	}

	require.NotNil(t, clearCmd)

	flag := clearCmd.Flags().Lookup("service")
	require.NotNil(t, flag)
	require.Equal(t, "s", flag.Shorthand)
}

func TestPrintCacheStatus_DoesNotPanic(t *testing.T) {
	t.Parallel()

	c := make(cache.Cache)
	cache.Update(c, "api", "abcdef123456789", "v1.0.0")

	// Should not panic — just prints to stdout
	require.NotPanics(t, func() {
		printCacheStatus(c, 1024)
	})
}

func TestPrintCacheStatus_EmptyCache(t *testing.T) {
	t.Parallel()

	c := make(cache.Cache)

	require.NotPanics(t, func() {
		printCacheStatus(c, 0)
	})
}

func TestClearSingleService_CorruptedCacheFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a corrupted cache file
	dir := filepath.Join(tmpDir, ".pilum")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(cache.FilePath(tmpDir), []byte("not json"), 0o644))

	// Should not error — corrupted cache loads as empty
	result, err := clearSingleService(tmpDir, "api")
	require.NoError(t, err)

	m, ok := result.(map[string]any)
	require.True(t, ok)
	require.Equal(t, 0, m["cleared"])
}
