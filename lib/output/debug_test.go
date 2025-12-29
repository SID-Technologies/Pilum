package output_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/output"

	"github.com/stretchr/testify/require"
)

func TestSetDebug(t *testing.T) {
	// Not parallel - modifies global state

	// Save original state
	originalDebug := output.Debug
	defer func() { output.Debug = originalDebug }()

	// Test enabling debug
	output.SetDebug(true)
	require.True(t, output.Debug)

	// Test disabling debug
	output.SetDebug(false)
	require.False(t, output.Debug)
}

func TestDebugVariable(t *testing.T) {
	// Not parallel - modifies global state

	// Save original state
	originalDebug := output.Debug
	defer func() { output.Debug = originalDebug }()

	// Test direct assignment
	output.Debug = true
	require.True(t, output.Debug)

	output.Debug = false
	require.False(t, output.Debug)
}

func TestDebugfWhenDisabled(t *testing.T) {
	// Not parallel - modifies global state

	// Save original state
	originalDebug := output.Debug
	defer func() { output.Debug = originalDebug }()

	// Ensure debug is disabled
	output.SetDebug(false)

	// This should not panic or produce output
	output.Debugf("test message %s %d", "arg", 123)
}

func TestDebugfWhenEnabled(t *testing.T) {
	// Not parallel - modifies global state

	// Save original state
	originalDebug := output.Debug
	defer func() { output.Debug = originalDebug }()

	// Enable debug
	output.SetDebug(true)

	// This should not panic - output goes to stderr
	output.Debugf("test message %s %d", "arg", 123)
}
