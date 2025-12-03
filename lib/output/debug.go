package output

import (
	"fmt"
	"os"
)

// Debug controls whether debug messages are printed.
var Debug bool

// SetDebug enables or disables debug output.
func SetDebug(enabled bool) {
	Debug = enabled
}

// Debugf prints a debug message if debug mode is enabled.
func Debugf(msg string, args ...any) {
	if !Debug {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintf(os.Stderr, "%s[debug] %s%s\n", Muted, formatted, Reset)
}
