package serviceinfo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		m        map[string]any
		key      string
		def      string
		expected string
	}{
		{
			name:     "existing string key",
			m:        map[string]any{"name": "test-value"},
			key:      "name",
			def:      "default",
			expected: "test-value",
		},
		{
			name:     "missing key returns default",
			m:        map[string]any{"other": "value"},
			key:      "name",
			def:      "default",
			expected: "default",
		},
		{
			name:     "non-string value returns default",
			m:        map[string]any{"count": 42},
			key:      "count",
			def:      "default",
			expected: "default",
		},
		{
			name:     "nil map returns default",
			m:        nil,
			key:      "name",
			def:      "default",
			expected: "default",
		},
		{
			name:     "empty map returns default",
			m:        map[string]any{},
			key:      "name",
			def:      "default",
			expected: "default",
		},
		{
			name:     "empty string value",
			m:        map[string]any{"name": ""},
			key:      "name",
			def:      "default",
			expected: "",
		},
		{
			name:     "empty default",
			m:        map[string]any{},
			key:      "name",
			def:      "",
			expected: "",
		},
		{
			name:     "nil value returns default",
			m:        map[string]any{"name": nil},
			key:      "name",
			def:      "default",
			expected: "default",
		},
		{
			name:     "bool value returns default",
			m:        map[string]any{"enabled": true},
			key:      "enabled",
			def:      "false",
			expected: "false",
		},
		{
			name:     "float value returns default",
			m:        map[string]any{"rate": 3.14},
			key:      "rate",
			def:      "0.0",
			expected: "0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getString(tt.m, tt.key, tt.def)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMapFromAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected map[string]any
	}{
		{
			name:     "map[string]any input",
			input:    map[string]any{"key": "value", "count": 42},
			expected: map[string]any{"key": "value", "count": 42},
		},
		{
			name:     "map[any]any input (yaml.v2 style)",
			input:    map[any]any{"key": "value", "count": 42},
			expected: map[string]any{"key": "value", "count": 42},
		},
		{
			name:     "empty map[string]any",
			input:    map[string]any{},
			expected: map[string]any{},
		},
		{
			name:     "empty map[any]any",
			input:    map[any]any{},
			expected: map[string]any{},
		},
		{
			name:     "nil input returns empty map",
			input:    nil,
			expected: map[string]any{},
		},
		{
			name:     "string input returns empty map",
			input:    "not a map",
			expected: map[string]any{},
		},
		{
			name:     "int input returns empty map",
			input:    42,
			expected: map[string]any{},
		},
		{
			name:     "slice input returns empty map",
			input:    []string{"a", "b"},
			expected: map[string]any{},
		},
		{
			name:     "map[any]any with non-string keys are skipped",
			input:    map[any]any{123: "value", "key": "other"},
			expected: map[string]any{"key": "other"},
		},
		{
			name:     "nested map values preserved",
			input:    map[string]any{"nested": map[string]any{"inner": "value"}},
			expected: map[string]any{"nested": map[string]any{"inner": "value"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := mapFromAny(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
