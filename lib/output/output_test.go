package output_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/output"
	"github.com/stretchr/testify/require"
)

func TestSymbolConstants(t *testing.T) {
	t.Parallel()

	// Verify symbols are defined and non-empty
	require.NotEmpty(t, output.SymbolSuccess)
	require.NotEmpty(t, output.SymbolFailure)
	require.NotEmpty(t, output.SymbolWarning)
	require.NotEmpty(t, output.SymbolInfo)
	require.NotEmpty(t, output.SymbolSkipped)
	require.NotEmpty(t, output.SymbolDryRun)

	// Verify expected values
	require.Equal(t, "✓", output.SymbolSuccess)
	require.Equal(t, "✗", output.SymbolFailure)
	require.Equal(t, "⚠", output.SymbolWarning)
	require.Equal(t, "●", output.SymbolInfo)
	require.Equal(t, "○", output.SymbolSkipped)
	require.Equal(t, "◌", output.SymbolDryRun)
}

func TestColorConstants(t *testing.T) {
	t.Parallel()

	// Verify colors are ANSI escape codes
	require.Contains(t, output.Reset, "\033[")
	require.Contains(t, output.Bold, "\033[")
	require.Contains(t, output.Primary, "\033[")
	require.Contains(t, output.Secondary, "\033[")
	require.Contains(t, output.SuccessColor, "\033[")
	require.Contains(t, output.WarningColor, "\033[")
	require.Contains(t, output.ErrorColor, "\033[")
	require.Contains(t, output.InfoColor, "\033[")
	require.Contains(t, output.Muted, "\033[")
}

func TestLegacyColorAliases(t *testing.T) {
	t.Parallel()

	// Verify legacy aliases are defined
	require.NotEmpty(t, output.Red)
	require.NotEmpty(t, output.Green)
	require.NotEmpty(t, output.Blue)
	require.NotEmpty(t, output.Cyan)
	require.NotEmpty(t, output.Purple)
	require.NotEmpty(t, output.Gray)
	require.NotEmpty(t, output.Yellow)
	require.NotEmpty(t, output.Orange)
}

func TestExtendedColors(t *testing.T) {
	t.Parallel()

	// Verify extended color palette
	require.NotEmpty(t, output.LightBlue)
	require.NotEmpty(t, output.LightCyan)
	require.NotEmpty(t, output.LightPurple)
	require.NotEmpty(t, output.LightRed)
	require.NotEmpty(t, output.DarkRed)
	require.NotEmpty(t, output.Brown)
	require.NotEmpty(t, output.DarkYellow)
	require.NotEmpty(t, output.Maroon)
	require.NotEmpty(t, output.Crimson)
	require.NotEmpty(t, output.BrightRed)
}

// Test that print functions don't panic
func TestErrorFunction(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Error("test error message")
	})
}

func TestErrorFunctionWithArgs(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Error("test error %s %d", "message", 42)
	})
}

func TestErrorWithDetail(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.ErrorWithDetail("main error", "additional detail")
	})
}

func TestErrorWithDetailEmpty(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.ErrorWithDetail("main error", "")
	})
}

func TestWarningFunction(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Warning("test warning")
	})
}

func TestWarningFunctionWithArgs(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Warning("warning %s", "message")
	})
}

func TestSuccessFunction(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Success("test success")
	})
}

func TestSuccessFunctionWithArgs(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Success("success %d%%", 100)
	})
}

func TestInfoFunction(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Info("test info")
	})
}

func TestInfoFunctionWithArgs(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Info("info %s", "message")
	})
}

func TestHeaderFunction(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Header("test header")
	})
}

func TestHeaderFunctionWithArgs(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Header("header %d", 1)
	})
}

func TestDimmedFunction(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Dimmed("test dimmed")
	})
}

func TestDimmedFunctionWithArgs(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		output.Dimmed("dimmed %s", "text")
	})
}
