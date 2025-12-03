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
func Error(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Fprintf(os.Stderr, "%s%s %s%s\n", ErrorColor, SymbolFailure, formatted, Reset)
}

// ErrorWithDetail prints an error with additional detail on the next line.
func ErrorWithDetail(msg string, detail string) {
	fmt.Fprintf(os.Stderr, "%s%s %s%s\n", ErrorColor, SymbolFailure, msg, Reset)
	if detail != "" {
		fmt.Fprintf(os.Stderr, "  %s%s%s\n", Muted, detail, Reset)
	}
}

// Warning prints a formatted warning message.
func Warning(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s%s %s%s\n", WarningColor, SymbolWarning, formatted, Reset)
}

// Success prints a formatted success message.
func Success(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s%s %s%s\n", SuccessColor, SymbolSuccess, formatted, Reset)
}

// Info prints a formatted info message.
func Info(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s%s %s%s\n", InfoColor, SymbolInfo, formatted, Reset)
}

// Header prints a bold header message.
func Header(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("\n%s%s%s\n\n", Bold, formatted, Reset)
}

// Dimmed prints a muted/gray message.
func Dimmed(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	fmt.Printf("%s%s%s\n", Muted, formatted, Reset)
}
