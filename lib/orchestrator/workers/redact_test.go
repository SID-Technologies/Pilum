package workers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldRedact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key      string
		expected bool
	}{
		{"GITHUB_TOKEN", true},
		{"AWS_SECRET_ACCESS_KEY", true},
		{"NPM_TOKEN", true},
		{"DATABASE_PASSWORD", true},
		{"API_KEY", true},
		{"MY_SERVICE_CREDENTIAL", true},
		{"CLOUDFLARE_API_KEY", true},
		{"MY_SECRET", true},
		{"MY_SERVICE_NAME", false},
		{"HOME", false},
		{"PATH", false},
		{"GOPATH", false},
		{"NODE_ENV", false},
		{"PORT", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, shouldRedact(tt.key))
		})
	}
}

func TestRedactEnvVars(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"GITHUB_TOKEN": "ghp_abc123",
		"MY_API_KEY":   "sk-secret",
		"HOME":         "/home/user",
		"SERVICE_NAME": "my-app",
		"AWS_SECRET":   "wJalrXUtnFEMI/K7MDENG",
		"DB_PASSWORD":  "hunter2",
	}

	redacted := redactEnvVars(envVars)

	require.Equal(t, "[REDACTED]", redacted["GITHUB_TOKEN"])
	require.Equal(t, "[REDACTED]", redacted["MY_API_KEY"])
	require.Equal(t, "/home/user", redacted["HOME"])
	require.Equal(t, "my-app", redacted["SERVICE_NAME"])
	require.Equal(t, "[REDACTED]", redacted["AWS_SECRET"])
	require.Equal(t, "[REDACTED]", redacted["DB_PASSWORD"])
}
