package homebrew_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/homebrew"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateTapPushCommand(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "myapp",
		Config: map[string]any{
			"homebrew": map[string]any{
				"tap_url":   "https://github.com/org/homebrew-tap",
				"token_env": "GITHUB_TOKEN",
			},
		},
	}

	result := homebrew.GenerateTapPushCommand(service, "v1.0.0", "dist/myapp.rb")

	require.NotEmpty(t, result)

	// Should check for token
	require.Contains(t, result, `if [ -z "$GITHUB_TOKEN" ]`)
	require.Contains(t, result, "Error: GITHUB_TOKEN environment variable is not set")

	// Should create temp directory
	require.Contains(t, result, "TAP_DIR=$(mktemp -d)")

	// Should clone with authentication
	require.Contains(t, result, "git clone")
	require.Contains(t, result, "$GITHUB_TOKEN@github.com")

	// Should create Formula directory
	require.Contains(t, result, "mkdir -p \"$TAP_DIR/Formula\"")

	// Should copy formula
	require.Contains(t, result, "cp dist/myapp.rb \"$TAP_DIR/Formula/myapp.rb\"")

	// Should configure git
	require.Contains(t, result, "git config user.name")
	require.Contains(t, result, "git config user.email")

	// Should add and commit
	require.Contains(t, result, "git add Formula/myapp.rb")
	require.Contains(t, result, "git commit -m")
	require.Contains(t, result, "Update myapp to v1.0.0")

	// Should push
	require.Contains(t, result, "git push")

	// Should cleanup
	require.Contains(t, result, "rm -rf \"$TAP_DIR\"")
}

func TestGenerateTapPushCommandDifferentToken(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "cli",
		Config: map[string]any{
			"homebrew": map[string]any{
				"tap_url":   "https://github.com/org/homebrew-cli",
				"token_env": "GH_PAT",
			},
		},
	}

	result := homebrew.GenerateTapPushCommand(service, "v2.0.0", "build/cli.rb")

	// Should use correct token env var
	require.Contains(t, result, `if [ -z "$GH_PAT" ]`)
	require.Contains(t, result, "$GH_PAT@github.com")
}

func TestGenerateTapPushCommandDifferentTapURL(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "tool",
		Config: map[string]any{
			"homebrew": map[string]any{
				"tap_url":   "https://github.com/myorg/homebrew-tools",
				"token_env": "TOKEN",
			},
		},
	}

	result := homebrew.GenerateTapPushCommand(service, "v1.0.0", "dist/tool.rb")

	// Should use correct tap URL
	require.Contains(t, result, "$TOKEN@github.com/myorg/homebrew-tools")
}

func TestBuildAuthenticatedURLHTTPS(t *testing.T) {
	t.Parallel()

	// Test with standard HTTPS URL
	service := serviceinfo.ServiceInfo{
		Name: "app",
		Config: map[string]any{
			"homebrew": map[string]any{
				"tap_url":   "https://github.com/org/tap",
				"token_env": "MY_TOKEN",
			},
		},
	}

	result := homebrew.GenerateTapPushCommand(service, "v1.0.0", "dist/app.rb")

	// Should have token inserted after https://
	require.Contains(t, result, "https://$MY_TOKEN@github.com")
}

func TestGenerateTapPushCommandCommitMessage(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "myapp",
		Config: map[string]any{
			"homebrew": map[string]any{
				"tap_url":   "https://github.com/org/tap",
				"token_env": "TOKEN",
			},
		},
	}

	result := homebrew.GenerateTapPushCommand(service, "v3.2.1", "dist/myapp.rb")

	// Should have correct commit message with tag
	require.Contains(t, result, "Update myapp to v3.2.1")
}

func TestGenerateTapPushCommandFormulaPath(t *testing.T) {
	t.Parallel()

	service := serviceinfo.ServiceInfo{
		Name: "cli",
		Config: map[string]any{
			"homebrew": map[string]any{
				"tap_url":   "https://github.com/org/tap",
				"token_env": "TOKEN",
			},
		},
	}

	result := homebrew.GenerateTapPushCommand(service, "v1.0.0", "/custom/path/cli.rb")

	// Should copy from custom path
	require.Contains(t, result, "cp /custom/path/cli.rb")
}
