//go:build !linux && !darwin

package sysinfo

func detectMemory() int {
	return 0 // Unsupported platform — fall back to CPU-only scheduling
}
