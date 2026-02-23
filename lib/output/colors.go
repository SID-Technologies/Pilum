package output

// ANSI escape codes - base formatting.
const (
	Reset    = "\033[0m"
	Bold     = "\033[1m"
	DimStyle = "\033[2m"
)

// Raw color codes - use semantic colors below instead.
// Roman-themed palette matching the website terminal.
const (
	rawRed     = "\033[31m"
	rawGreen   = "\033[32m"
	rawYellow  = "\033[33m"
	rawBlue    = "\033[34m"
	rawMagenta = "\033[35m"
	rawCyan    = "\033[36m"
	rawGray    = "\033[90m"
	rawOrange  = "\033[38;5;208m"
	rawGold    = "\033[38;5;178m" // Imperial gold (#d4af37)
	rawPurple  = "\033[38;5;141m" // Tyrian purple (#9b6dff)
	rawBronze  = "\033[38;5;172m" // Roman bronze (#cd7f32)
	rawLaurel  = "\033[38;5;108m" // Laurel green (#7a9a6d)
)

// Semantic colors - use these throughout the application
// Roman-themed: gold for success, purple for headers, bronze for progress, laurel for muted.
const (
	// Primary - main brand/action color (step headers, borders).
	Primary = rawPurple

	// Secondary - supporting color for less prominent elements.
	Secondary = rawBronze

	// Accent - highlights and emphasis.
	Accent = rawGold

	// SuccessColor - positive outcomes, completions.
	SuccessColor = rawGold

	// WarningColor - caution, attention needed, in-progress.
	WarningColor = rawBronze

	// ErrorColor - failures, problems.
	ErrorColor = rawRed

	// InfoColor - informational messages.
	InfoColor = rawPurple

	// Muted - de-emphasized text, timestamps, metadata.
	Muted = rawLaurel
)

// Legacy aliases - for backward compatibility
// Prefer semantic colors above for new code.
const (
	Red    = rawRed
	Green  = rawGreen
	Blue   = rawBlue
	Cyan   = rawCyan
	Purple = rawMagenta
	Gray   = rawGray
	Gold   = rawGold
	Yellow = rawYellow
	Orange = rawOrange
	Bronze = rawBronze
	Laurel = rawLaurel
)

const (
	LightBlue   = "\033[94m"
	LightCyan   = "\033[96m"
	LightPurple = "\033[95m"
	LightRed    = "\033[91m"
	DarkRed     = "\033[31m"
	Brown       = "\033[38;5;94m" // Dark brown (arena sand)
	// Orange     = "\033[38;5;208m" // Burnt orange (dust clouds).
	DarkYellow = "\033[38;5;136m" // Dusty gold (weathered bronze)
	Maroon     = "\033[38;5;52m"  // Deep maroon (darkest)
	Crimson    = "\033[38;5;124m" // Rich crimson
	BrightRed  = "\033[38;5;196m" // Vivid red
)
