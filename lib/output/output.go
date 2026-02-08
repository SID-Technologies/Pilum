// Package output provides consistent CLI output formatting with colors and symbols.
package output

import (
	"fmt"
	"os"
)

// Status symbols.
const (
	SymbolSuccess = "✓"
	SymbolFailure = "✗"
	SymbolWarning = "⚠"
	SymbolInfo    = "●"
	SymbolSkipped = "○"
	SymbolDryRun  = "◌"
)

// Error prints a formatted error message to stderr.
// In JSON mode, prints plain text without ANSI codes.
func Error(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	if IsJSON() {
		fmt.Fprintf(os.Stderr, "%s %s\n", SymbolFailure, formatted)
		return
	}
	fmt.Fprintf(os.Stderr, "%s%s %s%s\n", ErrorColor, SymbolFailure, formatted, Reset)
}

// ErrorWithDetail prints an error with additional detail on the next line.
// In JSON mode, prints plain text without ANSI codes.
func ErrorWithDetail(msg string, detail string) {
	if IsJSON() {
		fmt.Fprintf(os.Stderr, "%s %s\n", SymbolFailure, msg)
		if detail != "" {
			fmt.Fprintf(os.Stderr, "  %s\n", detail)
		}
		return
	}
	fmt.Fprintf(os.Stderr, "%s%s %s%s\n", ErrorColor, SymbolFailure, msg, Reset)
	if detail != "" {
		fmt.Fprintf(os.Stderr, "  %s%s%s\n", Muted, detail, Reset)
	}
}

// Warning prints a formatted warning message.
// Suppressed in JSON mode.
func Warning(msg string, args ...any) {
	if IsJSON() {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s%s %s%s\n", WarningColor, SymbolWarning, formatted, Reset)
}

// Success prints a formatted success message.
// Suppressed in JSON mode.
func Success(msg string, args ...any) {
	if IsJSON() {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s%s %s%s\n", SuccessColor, SymbolSuccess, formatted, Reset)
}

// Info prints a formatted info message.
// Suppressed in JSON mode.
func Info(msg string, args ...any) {
	if IsJSON() {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s%s %s%s\n", InfoColor, SymbolInfo, formatted, Reset)
}

// Header prints a bold header message.
// Suppressed in JSON mode.
func Header(msg string, args ...any) {
	if IsJSON() {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("\n%s%s%s\n\n", Bold, formatted, Reset)
}

// Dimmed prints a muted/gray message.
// Suppressed in JSON mode.
func Dimmed(msg string, args ...any) {
	if IsJSON() {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s%s%s\n", Muted, formatted, Reset)
}

// VerboseStdout prints a line of stdout with service name prefix.
func VerboseStdout(serviceName, line string) {
	if !IsVerbose() || IsJSON() {
		return
	}
	fmt.Printf("%s[%s]%s %s\n", Muted, serviceName, Reset, line)
}

// VerboseStderr prints a line of stderr with service name prefix in warning color.
func VerboseStderr(serviceName, line string) {
	if !IsVerbose() || IsJSON() {
		return
	}
	fmt.Fprintf(os.Stderr, "%s[%s]%s %s%s%s\n", Muted, serviceName, Reset, WarningColor, line, Reset)
}
