package output

const (
	// ANSI escape codes for colors.
	Reset = "\033[0m"
	Bold  = "\033[1m"

	// Basic Colors.
	Red    = "\033[31m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Purple = "\033[35m"

	// Bright/Light Colors.
	LightBlue   = "\033[94m"
	LightCyan   = "\033[96m"
	LightPurple = "\033[95m"

	Gold     = "\033[33m"
	LightRed = "\033[91m"
	DarkRed  = "\033[31m"
	Yellow   = "\033[93m"

	Brown      = "\033[38;5;94m"  // Dark brown (arena sand)
	Orange     = "\033[38;5;208m" // Burnt orange (dust clouds)
	DarkYellow = "\033[38;5;136m" // Dusty gold (weathered bronze)

	Maroon    = "\033[38;5;52m"  // Deep maroon (darkest)
	Crimson   = "\033[38;5;124m" // Rich crimson
	BrightRed = "\033[38;5;196m" // Vivid red
)
