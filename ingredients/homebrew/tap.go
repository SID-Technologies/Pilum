package homebrew

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateTapPushCommand creates a command to clone the tap repo, copy the formula, and push.
// It expects the formula to already exist at the formulaPath.
// Uses the tap_url and token_env from the service's homebrew config.
func GenerateTapPushCommand(svc serviceinfo.ServiceInfo, tag string, formulaPath string) string {
	name := svc.Name
	tapURL := svc.HomebrewConfig.TapURL
	tokenEnvVar := svc.HomebrewConfig.TokenEnv

	// Parse tap URL to construct authenticated clone URL
	// e.g., "https://github.com/org/tap" -> insert token before host
	cloneURL := buildAuthenticatedURL(tapURL, tokenEnvVar)

	// The script:
	// 1. Validates token is set
	// 2. Creates a temp directory for the tap clone
	// 3. Clones the tap repo using token for auth
	// 4. Copies the formula file
	// 5. Commits and pushes
	script := fmt.Sprintf(`
if [ -z "$%s" ]; then
  echo "Error: %s environment variable is not set"
  exit 1
fi

TAP_DIR=$(mktemp -d)
echo "Cloning tap repository..."
git clone "%s" "$TAP_DIR" --depth 1
mkdir -p "$TAP_DIR/Formula"
cp %s "$TAP_DIR/Formula/%s.rb"
cd "$TAP_DIR"
git config user.name "pilum[bot]"
git config user.email "pilum[bot]@noreply.local"
git add Formula/%s.rb
git commit -m "Update %s to %s"
git push
rm -rf "$TAP_DIR"
echo "Successfully pushed formula to tap"
`,
		tokenEnvVar, tokenEnvVar,
		cloneURL,
		formulaPath, name,
		name,
		name, tag,
	)

	return script
}

// buildAuthenticatedURL inserts the token env var into the URL for git clone.
// e.g., "https://github.com/org/tap" -> "https://$TOKEN@github.com/org/tap"
func buildAuthenticatedURL(url string, tokenEnvVar string) string {
	if strings.HasPrefix(url, "https://") {
		return strings.Replace(url, "https://", fmt.Sprintf("https://$%s@", tokenEnvVar), 1)
	}
	return url
}
