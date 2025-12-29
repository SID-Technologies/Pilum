package flags_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/flags"
	"github.com/sid-technologies/pilum/lib/types"

	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		flags       []types.FlagArg
		expected    map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name: "parse string flag with space separator",
			args: []string{"--name", "test-value"},
			flags: []types.FlagArg{
				{Name: "name", Flag: "name", Type: "string"},
			},
			expected:    map[string]any{"name": "test-value"},
			expectError: false,
		},
		{
			name: "parse int flag",
			args: []string{"--count", "42"},
			flags: []types.FlagArg{
				{Name: "count", Flag: "count", Type: "int"},
			},
			expected:    map[string]any{"count": 42},
			expectError: false,
		},
		{
			name: "parse float flag",
			args: []string{"--rate", "3.14"},
			flags: []types.FlagArg{
				{Name: "rate", Flag: "rate", Type: "float"},
			},
			expected:    map[string]any{"rate": 3.14},
			expectError: false,
		},
		{
			name: "parse bool flag true",
			args: []string{"--enabled", "true"},
			flags: []types.FlagArg{
				{Name: "enabled", Flag: "enabled", Type: "bool"},
			},
			expected:    map[string]any{"enabled": true},
			expectError: false,
		},
		{
			name: "parse bool flag false",
			args: []string{"--enabled", "false"},
			flags: []types.FlagArg{
				{Name: "enabled", Flag: "enabled", Type: "bool"},
			},
			expected:    map[string]any{"enabled": false},
			expectError: false,
		},
		{
			name: "parse multiple flags",
			args: []string{"--name", "myservice", "--count", "5", "--enabled", "true"},
			flags: []types.FlagArg{
				{Name: "name", Flag: "name", Type: "string"},
				{Name: "count", Flag: "count", Type: "int"},
				{Name: "enabled", Flag: "enabled", Type: "bool"},
			},
			expected: map[string]any{
				"name":    "myservice",
				"count":   5,
				"enabled": true,
			},
			expectError: false,
		},
		{
			name:        "empty args",
			args:        []string{},
			flags:       []types.FlagArg{},
			expected:    map[string]any{},
			expectError: false,
		},
		{
			name: "unexpected argument without --",
			args: []string{"invalid"},
			flags: []types.FlagArg{
				{Name: "name", Flag: "name", Type: "string"},
			},
			expectError: true,
			errorMsg:    "unexpected argument",
		},
		{
			name: "unknown flag",
			args: []string{"--unknown", "value"},
			flags: []types.FlagArg{
				{Name: "name", Flag: "name", Type: "string"},
			},
			expectError: true,
			errorMsg:    "unexpected flag",
		},
		{
			name: "missing value for flag",
			args: []string{"--name"},
			flags: []types.FlagArg{
				{Name: "name", Flag: "name", Type: "string"},
			},
			expectError: true,
			errorMsg:    "missing value",
		},
		{
			name: "invalid int value",
			args: []string{"--count", "not-a-number"},
			flags: []types.FlagArg{
				{Name: "count", Flag: "count", Type: "int"},
			},
			expectError: true,
			errorMsg:    "error parsing flag",
		},
		{
			name: "invalid float value",
			args: []string{"--rate", "not-a-float"},
			flags: []types.FlagArg{
				{Name: "rate", Flag: "rate", Type: "float"},
			},
			expectError: true,
			errorMsg:    "error parsing flag",
		},
		{
			name: "invalid bool value",
			args: []string{"--enabled", "maybe"},
			flags: []types.FlagArg{
				{Name: "enabled", Flag: "enabled", Type: "bool"},
			},
			expectError: true,
			errorMsg:    "error parsing flag",
		},
		{
			name: "unsupported flag type",
			args: []string{"--data", "value"},
			flags: []types.FlagArg{
				{Name: "data", Flag: "data", Type: "array"},
			},
			expectError: true,
			errorMsg:    "unsupported flag type",
		},
		{
			name: "empty string value",
			args: []string{"--name", ""},
			flags: []types.FlagArg{
				{Name: "name", Flag: "name", Type: "string"},
			},
			expected:    map[string]any{"name": ""},
			expectError: false,
		},
		{
			name: "string value with spaces",
			args: []string{"--message", "hello world"},
			flags: []types.FlagArg{
				{Name: "message", Flag: "message", Type: "string"},
			},
			expected:    map[string]any{"message": "hello world"},
			expectError: false,
		},
		{
			name: "negative int value",
			args: []string{"--count", "-5"},
			flags: []types.FlagArg{
				{Name: "count", Flag: "count", Type: "int"},
			},
			expected:    map[string]any{"count": -5},
			expectError: false,
		},
		{
			name: "negative float value",
			args: []string{"--rate", "-3.14"},
			flags: []types.FlagArg{
				{Name: "rate", Flag: "rate", Type: "float"},
			},
			expected:    map[string]any{"rate": -3.14},
			expectError: false,
		},
		{
			name: "zero int value",
			args: []string{"--count", "0"},
			flags: []types.FlagArg{
				{Name: "count", Flag: "count", Type: "int"},
			},
			expected:    map[string]any{"count": 0},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := flags.ParseArgs(tt.args, tt.flags)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateRequiredFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		options       []types.FlagArg
		providedFlags map[string]string
		expectedCount int
	}{
		{
			name: "all required flags present",
			options: []types.FlagArg{
				{Name: "name", Required: true},
				{Name: "region", Required: true},
			},
			providedFlags: map[string]string{
				"name":   "test",
				"region": "us-east-1",
			},
			expectedCount: 0,
		},
		{
			name: "one required flag missing",
			options: []types.FlagArg{
				{Name: "name", Required: true},
				{Name: "region", Required: true},
			},
			providedFlags: map[string]string{
				"name": "test",
			},
			expectedCount: 1,
		},
		{
			name: "all required flags missing",
			options: []types.FlagArg{
				{Name: "name", Required: true},
				{Name: "region", Required: true},
			},
			providedFlags: map[string]string{},
			expectedCount: 2,
		},
		{
			name: "no required flags",
			options: []types.FlagArg{
				{Name: "name", Required: false},
				{Name: "region", Required: false},
			},
			providedFlags: map[string]string{},
			expectedCount: 0,
		},
		{
			name: "mixed required and optional flags",
			options: []types.FlagArg{
				{Name: "name", Required: true},
				{Name: "region", Required: false},
				{Name: "project", Required: true},
			},
			providedFlags: map[string]string{
				"name": "test",
			},
			expectedCount: 1, // project is missing
		},
		{
			name:          "empty options",
			options:       []types.FlagArg{},
			providedFlags: map[string]string{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			missing := flags.ValidateRequiredFlags(tt.options, tt.providedFlags)
			require.Len(t, missing, tt.expectedCount)
		})
	}
}
