//go:build darwin || linux

package output

import (
	"os"

	"golang.org/x/sys/unix"
)

// terminalWidth returns stdout's width in columns, or 0 if it isn't a terminal.
func terminalWidth() int {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0
	}
	return int(ws.Col)
}
