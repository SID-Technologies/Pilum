package sysinfo

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTotalMemoryMB(t *testing.T) {
	t.Parallel()

	mem := TotalMemoryMB()

	switch runtime.GOOS {
	case "linux", "darwin":
		// Should detect memory on supported platforms
		require.Greater(t, mem, 0, "expected non-zero memory on %s", runtime.GOOS)
		// Sanity check: at least 256MB, less than 16TB
		require.Greater(t, mem, 256)
		require.Less(t, mem, 16*1024*1024)
	default:
		// Unsupported platforms return 0
		require.Equal(t, 0, mem)
	}
}
