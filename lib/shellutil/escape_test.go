package shellutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty string", input: "", expected: "''"},
		{name: "simple string", input: "hello", expected: "'hello'"},
		{name: "string with spaces", input: "hello world", expected: "'hello world'"},
		{name: "single quote", input: "it's", expected: `'it'\''s'`},
		{name: "double quotes", input: `say "hi"`, expected: `'say "hi"'`},
		{name: "backticks", input: "`whoami`", expected: "'`whoami`'"},
		{name: "command substitution", input: "$(whoami)", expected: "'$(whoami)'"},
		{name: "newline", input: "line1\nline2", expected: "'line1\nline2'"},
		{name: "semicolon", input: "a; rm -rf /", expected: "'a; rm -rf /'"},
		{name: "pipe", input: "a | b", expected: "'a | b'"},
		{name: "ampersand", input: "a && b", expected: "'a && b'"},
		{name: "dollar sign", input: "$HOME", expected: "'$HOME'"},
		{name: "multiple single quotes", input: "a'b'c", expected: `'a'\''b'\''c'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Quote(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateServiceName(t *testing.T) {
	t.Parallel()

	valid := []string{
		"my-service",
		"my_service",
		"my.service",
		"MyService123",
		"api",
		"a",
		"service-v2.0",
		"service_name-with.all-3",
	}
	for _, name := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ValidateServiceName(name))
		})
	}

	invalid := []struct {
		name string
		desc string
	}{
		{name: "", desc: "empty"},
		{name: "my service", desc: "space"},
		{name: "my;service", desc: "semicolon"},
		{name: "$(whoami)", desc: "command substitution"},
		{name: "`whoami`", desc: "backtick"},
		{name: "my/service", desc: "slash"},
		{name: "my\nservice", desc: "newline"},
		{name: "a&b", desc: "ampersand"},
		{name: "a|b", desc: "pipe"},
		{name: "a'b", desc: "single quote"},
	}
	for _, tt := range invalid {
		t.Run("invalid/"+tt.desc, func(t *testing.T) {
			t.Parallel()
			require.Error(t, ValidateServiceName(tt.name))
		})
	}
}

func TestSanitizeHeredocValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "plain text", input: "hello world", expected: "hello world"},
		{name: "dollar sign", input: "$HOME", expected: `\$HOME`},
		{name: "backtick", input: "`whoami`", expected: "\\`whoami\\`"},
		{name: "double quote", input: `say "hi"`, expected: `say \"hi\"`},
		{name: "backslash", input: `a\b`, expected: `a\\b`},
		{name: "command substitution", input: "$(rm -rf /)", expected: `\$(rm -rf /)`},
		{name: "mixed", input: `$HOME/` + "`cmd`", expected: `\$HOME/` + "\\`cmd\\`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := SanitizeHeredocValue(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
