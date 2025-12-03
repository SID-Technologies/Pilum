package helper

import (
	"os"
)

// EnsureCIEnvironment returns true if running in a CI/CD environment.
func EnsureCIEnvironment() bool {
	return os.Getenv("CI") != ""
}
