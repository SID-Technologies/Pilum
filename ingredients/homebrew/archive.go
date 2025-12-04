package homebrew

import (
	"fmt"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

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
