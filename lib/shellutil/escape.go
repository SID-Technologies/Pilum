// Package shellutil provides shell-safe string quoting and input validation
// to prevent command injection when constructing shell commands.
package shellutil

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Quote wraps a string in POSIX-compliant single quotes, escaping any
// embedded single quotes using the '\” idiom. This is equivalent to
// Python's shlex.quote() and is safe for use in sh -c command strings.
//
// Examples:
//
//	Quote("hello")       → 'hello'
//	Quote("it's")        → 'it'\''s'
//	Quote("$(whoami)")   → '$(whoami)'
//	Quote("")            → ''
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// serviceNameRe matches valid service names: alphanumeric, dots, hyphens, underscores.
var serviceNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ValidateServiceName returns an error if the name contains characters outside
// [a-zA-Z0-9._-]. This is a defense-in-depth check to reject obviously
// malicious names before they reach any command construction.
func ValidateServiceName(name string) error {
	if name == "" {
		return errors.New("service name must not be empty")
	}
	if !serviceNameRe.MatchString(name) {
		return fmt.Errorf("service name %q contains invalid characters (allowed: a-z, A-Z, 0-9, '.', '-', '_')", name)
	}
	return nil
}

// SanitizeHeredocValue escapes characters that are dangerous inside an
// unquoted shell heredoc (where $ and ` are still expanded). This replaces
// $, `, \, and double-quotes with escaped variants so values can be safely
// interpolated into heredoc content without shell expansion or breakout.
func SanitizeHeredocValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "$", `\$`)
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
