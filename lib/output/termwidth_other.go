//go:build !linux && !darwin

package output

// terminalWidth is unknown on this platform; callers fall back to $COLUMNS or 80.
func terminalWidth() int {
	return 0
}
