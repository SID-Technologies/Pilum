// Package errors provides a consistent interface for using errors.
// It also supports slog structured logging attributes; i.e. structured errors.
// It is also a drop-in replacement for the standard library errors package.
package errors

import (
	"fmt"
	"strings"

	"github.com/sid-technologies/pilum/lib/output"

	pkgerrors "github.com/pkg/errors"
)

// New returns an error that formats as the given text and
// contains the structured (slog) attributes.
//
//nolint:wrapcheck,inamedparam // This function does custom wrapping and errors.
func New(msg string, attrs ...any) error {
	formatted := fmt.Sprintf(msg, attrs...)
	output.Error("%s", formatted)
	return structured{
		err:   pkgerrors.New(formatted),
		attrs: attrs,
	}
}

// Wrap returns a new error wrapping the provided with additional
// structured fields. Like New, msg is a format string when it contains verbs
// ("describing %s", name); attrs without verbs are kept as attributes only.
//
//nolint:wrapcheck,inamedparam // This function does custom wrapping and errors.
func Wrap(err error, msg string, attrs ...any) error {
	if err == nil {
		panic("wrap nil error")
	}

	// Support error types that do their own wrapping.
	if wrapper, ok := err.(interface{ Wrap(string, ...any) error }); ok {
		return wrapper.Wrap(msg, attrs...)
	}

	formatted := msg
	if len(attrs) > 0 && strings.Contains(msg, "%") {
		formatted = fmt.Sprintf(msg, attrs...)
	}

	var inner structured
	if As(err, &inner) {
		attrs = append(attrs, inner.attrs...) // Append inner attributes
	}

	return structured{
		err:   pkgerrors.Wrap(err, formatted),
		attrs: attrs,
	}
}
