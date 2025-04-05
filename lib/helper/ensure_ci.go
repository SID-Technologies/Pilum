package helper

import (
	"os"
)

func EnsureCIEnvironment() bool {
	// """Ensure this script is running inside a CI/CD environment."""
	if os.Getenv("CI") != "" {
		print("❌ Error: Deployment should only be executed in a CI/CD pipeline!")
		return false
	}
	return true
}
