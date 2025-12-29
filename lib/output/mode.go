package output

// Mode defines the output verbosity level.
type Mode int

const (
	// ModeNormal is the default output with spinners and formatted text.
	ModeNormal Mode = iota
	// ModeVerbose streams command stdout/stderr in real-time.
	ModeVerbose
	// ModeQuiet shows minimal output (CI-friendly).
	ModeQuiet
	// ModeJSON outputs structured JSON for scripting.
	ModeJSON
)

// currentMode holds the global output mode.
var currentMode = ModeNormal

// SetMode sets the global output mode.
func SetMode(mode Mode) {
	currentMode = mode
}

// GetMode returns the current output mode.
func GetMode() Mode {
	return currentMode
}

// IsVerbose returns true if verbose mode is enabled.
func IsVerbose() bool {
	return currentMode == ModeVerbose
}

// IsQuiet returns true if quiet mode is enabled.
func IsQuiet() bool {
	return currentMode == ModeQuiet
}

// IsJSON returns true if JSON mode is enabled.
func IsJSON() bool {
	return currentMode == ModeJSON
}
