package output

import (
	"fmt"
	"os"
	"sync/atomic"
)

// debug controls whether debug messages are printed.
var debug atomic.Bool

// SetDebug enables or disables debug output.
func SetDebug(enabled bool) {
	debug.Store(enabled)
}

// IsDebug returns true if debug mode is enabled.
func IsDebug() bool {
	return debug.Load()
}

// Debugf prints a debug message if debug mode is enabled.
func Debugf(msg string, args ...any) {
	if !debug.Load() {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintf(os.Stderr, "%s[debug] %s%s\n", Muted, formatted, Reset)
}
