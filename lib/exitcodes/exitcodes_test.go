package exitcodes_test

import (
	"fmt"
	"testing"

	"github.com/sid-technologies/pilum/lib/exitcodes"

	"github.com/stretchr/testify/require"
)

func TestConstants(t *testing.T) {
	t.Parallel()

	// Ensure all codes are distinct
	codes := map[int]string{
		exitcodes.Success:     "Success",
		exitcodes.General:     "General",
		exitcodes.Config:      "Config",
		exitcodes.NoServices:  "NoServices",
		exitcodes.Deploy:      "Deploy",
		exitcodes.IO:          "IO",
		exitcodes.InvalidArgs: "InvalidArgs",
	}
	require.Len(t, codes, 7, "all exit codes should be distinct")
}

func TestErrorMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *exitcodes.Error
		expected string
	}{
		{
			name:     "with message",
			err:      exitcodes.NewError(exitcodes.Config, "bad config"),
			expected: "bad config",
		},
		{
			name:     "with wrapped error",
			err:      exitcodes.WithCode(exitcodes.Deploy, fmt.Errorf("deploy failed")),
			expected: "deploy failed",
		},
		{
			name:     "message takes precedence",
			err:      &exitcodes.Error{Code: 1, Message: "msg", Err: fmt.Errorf("inner")},
			expected: "msg",
		},
		{
			name:     "neither message nor error",
			err:      &exitcodes.Error{Code: 1},
			expected: "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestUnwrap(t *testing.T) {
	t.Parallel()

	inner := fmt.Errorf("root cause")
	err := exitcodes.WithCode(exitcodes.Deploy, inner)

	require.ErrorIs(t, err, inner)
}

func TestCodeFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: exitcodes.Success,
		},
		{
			name:     "direct exit code error",
			err:      exitcodes.NewError(exitcodes.Config, "bad config"),
			expected: exitcodes.Config,
		},
		{
			name:     "wrapped exit code error",
			err:      fmt.Errorf("outer: %w", exitcodes.WithCode(exitcodes.Deploy, fmt.Errorf("inner"))),
			expected: exitcodes.Deploy,
		},
		{
			name:     "plain error returns General",
			err:      fmt.Errorf("some error"),
			expected: exitcodes.General,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, exitcodes.CodeFrom(tt.err))
		})
	}
}

func TestWithCode(t *testing.T) {
	t.Parallel()

	inner := fmt.Errorf("step failed")
	err := exitcodes.WithCode(exitcodes.Deploy, inner)

	require.Equal(t, exitcodes.Deploy, err.Code)
	require.Equal(t, "step failed", err.Error())
	require.Equal(t, inner, err.Unwrap())
}

func TestNewError(t *testing.T) {
	t.Parallel()

	err := exitcodes.NewError(exitcodes.InvalidArgs, "unknown provider")

	require.Equal(t, exitcodes.InvalidArgs, err.Code)
	require.Equal(t, "unknown provider", err.Error())
	require.Nil(t, err.Unwrap())
}
