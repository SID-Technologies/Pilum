package telemetry

import "strings"

// errorClasses map words in a step's error to a category, checked in order.
// The error text is read here on the user's machine and never leaves it;
// only the category is sent.
var errorClasses = []struct {
	class    string
	patterns []string
}{
	{"auth", []string{
		"unauthenticated", "not authenticated", "authentication", "401", "invalid credentials",
		"expired token", "token expired", "token has expired", "reauthenticate", "gcloud auth login",
		"no credentials", "could not find default credentials", "login required",
	}},
	{"permission", []string{
		"permission denied", "permission_denied", "403", "forbidden", "access denied", "accessdenied",
		"not authorized", "unauthorized", "insufficient permission", "does not have permission",
	}},
	{"quota", []string{
		"quota", "rate limit", "ratelimit", "429", "too many requests", "resource_exhausted",
		"resource exhausted", "limit exceeded",
	}},
	{"timeout", []string{"timed out", "timeout", "deadline exceeded", "deadline_exceeded"}},
	{"not_found", []string{"not found", "not_found", "404", "no such file", "does not exist"}},
	{"network", []string{
		"connection refused", "no such host", "dial tcp", "network is unreachable", "tls handshake",
		"connection reset", "i/o timeout", "temporary failure in name resolution",
	}},
	{"build_failed", []string{
		"build failed", "failed to solve", "docker build", "compilation", "npm err", "go build",
		"exit status", "exited with",
	}},
}

// Classify turns an error into one of the schema's error_class values.
func Classify(err error) string {
	if err == nil {
		return "unknown"
	}
	text := strings.ToLower(err.Error())
	for _, c := range errorClasses {
		for _, p := range c.patterns {
			if strings.Contains(text, p) {
				return c.class
			}
		}
	}
	return "unknown"
}
