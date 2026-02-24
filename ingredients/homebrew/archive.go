package homebrew

import (
	"fmt"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/shellutil"
)

// GenerateArchiveCommand creates archives for all built binaries.
func GenerateArchiveCommand(svc serviceinfo.ServiceInfo, tag string, outputDir string) string {
	// Name and tag are validated by ValidateServiceName, but quote the pattern
	// to prevent glob injection if values ever contain metacharacters.
	pattern := fmt.Sprintf("%s_%s_*", shellutil.Quote(svc.Name), shellutil.Quote(tag))
	return fmt.Sprintf(`cd %s && for f in %s; do [ -f "$f" ] && tar -czf "${f}.tar.gz" "$f" && rm "$f"; done`,
		outputDir, pattern)
}

// GenerateChecksumCommand creates SHA256 checksums for all archives.
func GenerateChecksumCommand(outputDir string) string {
	return fmt.Sprintf("cd %s && shasum -a 256 *.tar.gz > checksums.txt", outputDir)
}
