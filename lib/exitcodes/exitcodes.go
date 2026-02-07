// Package exitcodes defines distinct exit codes for the Pilum CLI.
// Scripts and AI agents can branch on these codes without parsing stderr.
package exitcodes

// Exit codes for pilum CLI.
const (
	Success     = 0 // Successful execution
	General     = 1 // Unclassified errors
	Config      = 2 // Configuration/validation errors (bad YAML, missing fields)
	NoServices  = 3 // Discovery errors (no services found, no recipes)
	Deploy      = 4 // Execution errors (command failures, timeouts, step failures)
	IO          = 5 // File I/O errors (can't read/write files)
	InvalidArgs = 6 // Invalid CLI arguments (unknown provider, bad flags)
)

// Error is an error that carries a specific exit code.
type Error struct {
	Code    int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

func (e *Error) Unwrap() error {
	return e.Err
}

// WithCode wraps an existing error with an exit code.
func WithCode(code int, err error) *Error {
	return &Error{Code: code, Err: err}
}

// NewError creates a new exit code error with a message.
func NewError(code int, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

// CodeFrom extracts the exit code from an error.
// If the error contains an exitcodes.Error, its Code is returned.
// Otherwise, General (1) is returned.
func CodeFrom(err error) int {
	if err == nil {
		return Success
	}

	// Check if the error itself is an *Error
	if e, ok := err.(*Error); ok { //nolint:errorlint // intentional direct type check first
		return e.Code
	}

	// Check wrapped errors
	type unwrapper interface{ Unwrap() error }
	for {
		u, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = u.Unwrap()
		if e, ok := err.(*Error); ok { //nolint:errorlint // walking the chain manually
			return e.Code
		}
	}

	return General
}
