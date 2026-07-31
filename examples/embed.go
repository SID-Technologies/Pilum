// Package examples embeds the reference Dockerfiles under examples/<lang>-service/
// so `pilum init` can scaffold a starter _templates/<name> file for recipes that
// require a Dockerfile template, instead of leaving the user to hand-write one.
package examples

import "embed"

//go:embed */Dockerfile
var dockerfiles embed.FS

// languageDirs maps a pilum build language key (from lib/templates/builds/*.yaml)
// to its example service directory name.
var languageDirs = map[string]string{
	"go":     "go-service",
	"python": "python-service",
	"node":   "nodejs-service",
	"rust":   "rust-service",
}

// DockerfileFor returns the reference Dockerfile content for a build language.
// The second return value is false if there is no example for that language.
func DockerfileFor(language string) ([]byte, bool) {
	dir, ok := languageDirs[language]
	if !ok {
		return nil, false
	}
	data, err := dockerfiles.ReadFile(dir + "/Dockerfile")
	if err != nil {
		return nil, false
	}
	return data, true
}
