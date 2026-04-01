package sysinfo

import (
	"encoding/binary"
	"syscall"
)

func detectMemory() int {
	val, err := syscall.Sysctl("hw.memsize")
	if err != nil || len(val) == 0 {
		return 0
	}

	// hw.memsize is a 64-bit little-endian integer returned as raw bytes.
	// syscall.Sysctl may strip the trailing null, so pad to 8 bytes.
	var buf [8]byte
	copy(buf[:], []byte(val))
	mem := binary.LittleEndian.Uint64(buf[:])
	return int(mem / (1024 * 1024)) // bytes -> MB
}
