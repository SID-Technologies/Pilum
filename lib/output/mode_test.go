package output

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutputModeDefaults(t *testing.T) {
	// Reset to default
	SetMode(ModeNormal)

	require.Equal(t, ModeNormal, GetMode())
	require.False(t, IsVerbose())
	require.False(t, IsQuiet())
	require.False(t, IsJSON())
}

func TestSetModeVerbose(t *testing.T) {
	SetMode(ModeVerbose)
	defer SetMode(ModeNormal)

	require.Equal(t, ModeVerbose, GetMode())
	require.True(t, IsVerbose())
	require.False(t, IsQuiet())
	require.False(t, IsJSON())
}

func TestSetModeQuiet(t *testing.T) {
	SetMode(ModeQuiet)
	defer SetMode(ModeNormal)

	require.Equal(t, ModeQuiet, GetMode())
	require.False(t, IsVerbose())
	require.True(t, IsQuiet())
	require.False(t, IsJSON())
}

func TestSetModeJSON(t *testing.T) {
	SetMode(ModeJSON)
	defer SetMode(ModeNormal)

	require.Equal(t, ModeJSON, GetMode())
	require.False(t, IsVerbose())
	require.False(t, IsQuiet())
	require.True(t, IsJSON())
}

func TestModeConstants(t *testing.T) {
	require.Equal(t, OutputMode(0), ModeNormal)
	require.Equal(t, OutputMode(1), ModeVerbose)
	require.Equal(t, OutputMode(2), ModeQuiet)
	require.Equal(t, OutputMode(3), ModeJSON)
}
