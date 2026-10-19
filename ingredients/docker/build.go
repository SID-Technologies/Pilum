package docker

import (
	"slices"
	"strings"

	"github.com/sid-technologies/pilum/ingredients/aws"
	service "github.com/sid-technologies/pilum/lib/service_info"
)

// DefaultPlatform is the image platform when build.platform is unset.
const DefaultPlatform = "linux/amd64"

// cacheTag names the registry tag that holds an image's layer cache.
const cacheTag = "buildcache"

func GenerateDockerBuildCommand(service service.ServiceInfo, imageName, templatePath string) []string {
	platform := service.BuildConfig.Platform
	if platform == "" {
		platform = aws.LambdaPlatform(service)
	}
	if platform == "" {
		platform = DefaultPlatform
	}

	cmd := []string{"docker", "build"}
	if service.BuildConfig.DockerCache {
		// buildx can read and write a registry cache; --load keeps the image
		// in the local store for the push step. A missing cache only warns,
		// and ignore-error does the same for a failed upload, so a cache
		// problem costs speed, never the build.
		cache := "type=registry,ref=" + imageRepository(imageName) + ":" + cacheTag
		cacheTo := cache + ",mode=max,ignore-error=true"
		if aws.IsECRHost(aws.RegistryOf(imageName)) {
			// ECR rejects buildx's default cache manifest list; it takes
			// the cache as an OCI image manifest.
			cacheTo += ",image-manifest=true,oci-mediatypes=true"
		}
		cmd = []string{
			"docker", "buildx", "build",
			"--cache-from", cache,
			"--cache-to", cacheTo,
			"--load",
		}
	}
	cmd = append(cmd, "--platform", platform, "-t", imageName)

	// Lambda only takes a single-platform image manifest, and BuildKit's
	// provenance attestation turns the image into an index.
	if aws.IsLambdaImage(service) {
		cmd = append(cmd, "--provenance=false")
	}

	// Also tag :latest locally when the primary tag is versioned.
	// Push only publishes the explicit tag the remote :latest pointer still moves exclusively via CI.
	if latest := latestAlias(imageName); latest != "" {
		cmd = append(cmd, "-t", latest)
	}

	cmd = append(cmd, "--build-arg", "SERVICE_NAME="+service.Path)

	// One --build-arg per variable: docker reads "A=1,B=2" as A="1,B=2".
	envArgs := make([]string, 0, len(service.EnvVars))
	for _, env := range service.EnvVars {
		envArgs = append(envArgs, env.Name+"="+env.Value)
	}
	slices.Sort(envArgs)
	for _, arg := range envArgs {
		cmd = append(cmd, "--build-arg", arg)
	}

	cmd = append(cmd, "-f", templatePath, ".")

	return cmd
}

// latestAlias returns imageName with its tag swapped for :latest, or "" when there is nothing to alias (no tag, or the tag already is latest). The last
// ":" is only a tag separator if no "/" follows it — a ":" before a slash is a registry port (e.g. localhost:5000/img).
func latestAlias(imageName string) string {
	idx := strings.LastIndex(imageName, ":")
	if idx == -1 || strings.Contains(imageName[idx:], "/") {
		return ""
	}

	if imageName[idx+1:] == "latest" {
		return ""
	}

	return imageName[:idx] + ":latest"
}

// imageRepository returns imageName without its tag. A ":" followed by a
// "/" is a registry port, not a tag.
func imageRepository(imageName string) string {
	idx := strings.LastIndex(imageName, ":")
	if idx == -1 || strings.Contains(imageName[idx:], "/") {
		return imageName
	}
	return imageName[:idx]
}
