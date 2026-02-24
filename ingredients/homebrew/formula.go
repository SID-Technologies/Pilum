package homebrew

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/shellutil"
)

// GenerateFormulaCommand creates a command that generates the Homebrew formula.
func GenerateFormulaCommand(svc serviceinfo.ServiceInfo, tag string, outputDir string, formulaPath string) string {
	cfg := ParseConfig(svc.Config)

	name := svc.Name
	description := shellutil.SanitizeHeredocValue(svc.Description)
	license := shellutil.SanitizeHeredocValue(svc.License)

	// Sanitize values for heredoc/double-quoted shell contexts
	safeProjectURL := shellutil.SanitizeHeredocValue(cfg.ProjectURL)
	safeTag := shellutil.SanitizeHeredocValue(tag)
	safeVersion := shellutil.SanitizeHeredocValue(strings.TrimPrefix(tag, "v"))

	// Shell-quoted paths for commands outside the heredoc
	quotedChecksumPath := shellutil.Quote(outputDir + "/checksums.txt")
	quotedFormulaPath := shellutil.Quote(formulaPath)

	// projectURL is like "https://github.com/org/project" - we use it directly for URLs
	script := fmt.Sprintf(`
DARWIN_ARM64_SHA=$(grep "%s_%s_darwin_arm64" %s | awk '{print $1}')
DARWIN_AMD64_SHA=$(grep "%s_%s_darwin_amd64" %s | awk '{print $1}')
LINUX_ARM64_SHA=$(grep "%s_%s_linux_arm64" %s | awk '{print $1}')
LINUX_AMD64_SHA=$(grep "%s_%s_linux_amd64" %s | awk '{print $1}')

cat > %s << FORMULA
class %s < Formula
  desc "%s"
  homepage "%s"
  version "%s"
  license "%s"

  on_macos do
    if Hardware::CPU.arm?
      url "%s/releases/download/%s/%s_%s_darwin_arm64.tar.gz"
      sha256 "$DARWIN_ARM64_SHA"
    else
      url "%s/releases/download/%s/%s_%s_darwin_amd64.tar.gz"
      sha256 "$DARWIN_AMD64_SHA"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "%s/releases/download/%s/%s_%s_linux_arm64.tar.gz"
      sha256 "$LINUX_ARM64_SHA"
    else
      url "%s/releases/download/%s/%s_%s_linux_amd64.tar.gz"
      sha256 "$LINUX_AMD64_SHA"
    end
  end

  def install
    bin.install Dir["%s_*"].first => "%s"
  end

  test do
    system "#{bin}/%s", "--version"
  end
end
FORMULA
`,
		// Checksum grep patterns (name validated by ValidateServiceName, tag sanitized)
		name, safeTag, quotedChecksumPath,
		name, safeTag, quotedChecksumPath,
		name, safeTag, quotedChecksumPath,
		name, safeTag, quotedChecksumPath,
		// Output path (shell-quoted) and class name (name is validated)
		quotedFormulaPath, capitalize(name),
		// Description, homepage, version, license (all sanitized for heredoc)
		description, safeProjectURL, safeVersion, license,
		// Darwin ARM64 (projectURL and tag sanitized, name validated)
		safeProjectURL, safeTag, name, safeTag,
		// Darwin AMD64
		safeProjectURL, safeTag, name, safeTag,
		// Linux ARM64
		safeProjectURL, safeTag, name, safeTag,
		// Linux AMD64
		safeProjectURL, safeTag, name, safeTag,
		// Install (name is validated)
		name, name,
		// Test (name is validated)
		name,
	)

	return script
}

// capitalize returns the string with the first letter uppercased.
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
