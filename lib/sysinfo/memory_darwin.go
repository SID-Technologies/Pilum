package sysinfo

import "syscall"

func detectMemory() int {
	val, err := syscall.Sysctl("hw.memsize")
	if err != nil || len(val) == 0 {
		return 0
	}

	// syscall.Sysctl returns raw bytes for numeric values.
	// hw.memsize is a 64-bit little-endian integer.
	raw := []byte(val)
	var mem uint64
	for i := range raw {
		if i >= 8 {
			break
		}
		mem |= uint64(raw[i]) << (uint(i) * 8)
	}

	return int(mem / (1024 * 1024)) // bytes -> MB
}
