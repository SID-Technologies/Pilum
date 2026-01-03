package homebrew

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// GenerateFormulaCommand creates a command that generates the Homebrew formula.
func GenerateFormulaCommand(svc serviceinfo.ServiceInfo, tag string, outputDir string, formulaPath string) string {
	cfg := ParseHomebrewConfig(svc.Config)

	name := svc.Name
	projectURL := cfg.ProjectURL
	description := svc.Description
	license := svc.License

	// Strip leading 'v' from tag for version field (URLs already use the full tag)
	version := strings.TrimPrefix(tag, "v")

	// projectURL is like "https://github.com/org/project" - we use it directly for URLs
	script := fmt.Sprintf(`
DARWIN_ARM64_SHA=$(grep "%s_%s_darwin_arm64" %s/checksums.txt | awk '{print $1}')
DARWIN_AMD64_SHA=$(grep "%s_%s_darwin_amd64" %s/checksums.txt | awk '{print $1}')
LINUX_ARM64_SHA=$(grep "%s_%s_linux_arm64" %s/checksums.txt | awk '{print $1}')
LINUX_AMD64_SHA=$(grep "%s_%s_linux_amd64" %s/checksums.txt | awk '{print $1}')

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
		// Checksum grep patterns
		name, tag, outputDir,
		name, tag, outputDir,
		name, tag, outputDir,
		name, tag, outputDir,
		// Output path and class name
		formulaPath, capitalize(name),
		// Description, homepage, version, license
		description, projectURL, version, license,
		// Darwin ARM64
		projectURL, tag, name, tag,
		// Darwin AMD64
		projectURL, tag, name, tag,
		// Linux ARM64
		projectURL, tag, name, tag,
		// Linux AMD64
		projectURL, tag, name, tag,
		// Install
		name, name,
		// Test
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
