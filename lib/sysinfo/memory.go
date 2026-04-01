package sysinfo

import "sync"

var (
	totalMemoryOnce sync.Once
	totalMemoryVal  int
)

// TotalMemoryMB returns the total system memory in megabytes.
// Returns 0 if detection fails (caller should fall back to CPU-only scheduling).
func TotalMemoryMB() int {
	totalMemoryOnce.Do(func() {
		totalMemoryVal = detectMemory()
	})
	return totalMemoryVal
}
