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
	// Base ldflags for smaller binaries
	ldflags := "-s -w"

	// Inject version at build time if version_var is configured
	if svc.BuildConfig.VersionVar != "" {
		ldflags = fmt.Sprintf("%s -X %s=%s", ldflags, svc.BuildConfig.VersionVar, tag)
	}

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
