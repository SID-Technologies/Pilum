package homebrew

import (
	"fmt"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/shellutil"
)

// GenerateArchiveCommand creates archives for all built binaries.
// The binary inside each archive is renamed to just the service name (e.g., "pilum")
// since the version and platform are already encoded in the archive filename.
func GenerateArchiveCommand(svc serviceinfo.ServiceInfo, tag string, outputDir string) string {
	// Name and tag are validated by ValidateServiceName, but quote the pattern
	// to prevent glob injection if values ever contain metacharacters.
	pattern := fmt.Sprintf("%s_%s_*", shellutil.Quote(svc.Name), shellutil.Quote(tag))
	name := shellutil.Quote(svc.Name)
	return fmt.Sprintf(`cd %s && for f in %s; do [ -f "$f" ] && mv "$f" %s && tar -czf "${f}.tar.gz" %s && rm %s; done`,
		outputDir, pattern, name, name, name)
}

// GenerateChecksumCommand creates SHA256 checksums for all archives.
func GenerateChecksumCommand(outputDir string) string {
	return fmt.Sprintf("cd %s && shasum -a 256 *.tar.gz > checksums.txt", outputDir)
}
