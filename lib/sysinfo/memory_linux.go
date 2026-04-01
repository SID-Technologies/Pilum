package sysinfo

import (
	"os"
	"strconv"
	"strings"
)

func detectMemory() int {
	return readLinuxMemory()
}

// readLinuxMemory reads total memory from /proc/meminfo.
func readLinuxMemory() int {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0
			}
			kb, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0
			}
			return int(kb / 1024) // KB -> MB
		}
	}
	return 0
}
