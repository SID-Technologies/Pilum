package homebrew

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// Platforms defines the default build targets.
var Platforms = []string{
	"darwin/amd64",
	"darwin/arm64",
	"linux/amd64",
	"linux/arm64",
}

// GenerateBuildCommand creates a multi-platform Go build script.
func GenerateBuildCommand(svc serviceinfo.ServiceInfo, tag string, outputDir string) string {
	ldflags := "-s -w"

	var lines []string
	lines = append(lines, fmt.Sprintf("mkdir -p %s", outputDir))

	for _, platform := range Platforms {
		parts := strings.Split(platform, "/")
		if len(parts) != 2 {
			continue
		}
		goos, goarch := parts[0], parts[1]
		output := fmt.Sprintf("%s/%s_%s_%s_%s", outputDir, svc.Name, tag, goos, goarch)
		lines = append(lines, fmt.Sprintf(
			`GOOS=%s GOARCH=%s CGO_ENABLED=0 go build -ldflags="%s" -o "%s" .`,
			goos, goarch, ldflags, output,
		))
	}

	return strings.Join(lines, " && ")
}

// GenerateArchiveCommand creates archives for all built binaries.
func GenerateArchiveCommand(svc serviceinfo.ServiceInfo, tag string, outputDir string) string {
	pattern := fmt.Sprintf("%s_%s_*", svc.Name, tag)
	return fmt.Sprintf(`cd %s && for f in %s; do [ -f "$f" ] && tar -czf "${f}.tar.gz" "$f" && rm "$f"; done`,
		outputDir, pattern)
}

// GenerateChecksumCommand creates SHA256 checksums for all archives.
func GenerateChecksumCommand(outputDir string) string {
	return fmt.Sprintf("cd %s && shasum -a 256 *.tar.gz > checksums.txt", outputDir)
}

// GenerateFormulaCommand creates a command that generates the Homebrew formula.
func GenerateFormulaCommand(svc serviceinfo.ServiceInfo, tag string, outputDir string, formulaPath string) string {
	name := svc.Name
	project := svc.Project

	script := fmt.Sprintf(`
DARWIN_ARM64_SHA=$(grep "%s_%s_darwin_arm64" %s/checksums.txt | awk '{print $1}')
DARWIN_AMD64_SHA=$(grep "%s_%s_darwin_amd64" %s/checksums.txt | awk '{print $1}')
LINUX_ARM64_SHA=$(grep "%s_%s_linux_arm64" %s/checksums.txt | awk '{print $1}')
LINUX_AMD64_SHA=$(grep "%s_%s_linux_amd64" %s/checksums.txt | awk '{print $1}')

cat > %s << FORMULA
class %s < Formula
  desc "Multi-service deployment orchestrator"
  homepage "https://github.com/%s/%s"
  version "%s"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/%s/%s/releases/download/v%s/%s_%s_darwin_arm64.tar.gz"
      sha256 "$DARWIN_ARM64_SHA"
    else
      url "https://github.com/%s/%s/releases/download/v%s/%s_%s_darwin_amd64.tar.gz"
      sha256 "$DARWIN_AMD64_SHA"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/%s/%s/releases/download/v%s/%s_%s_linux_arm64.tar.gz"
      sha256 "$LINUX_ARM64_SHA"
    else
      url "https://github.com/%s/%s/releases/download/v%s/%s_%s_linux_amd64.tar.gz"
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
		// Homepage
		project, name, tag,
		// Darwin ARM64
		project, name, tag, name, tag,
		// Darwin AMD64
		project, name, tag, name, tag,
		// Linux ARM64
		project, name, tag, name, tag,
		// Linux AMD64
		project, name, tag, name, tag,
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
