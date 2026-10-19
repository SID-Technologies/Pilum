package aws

import (
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/shellutil"
)

// GenerateECRLoginCommand logs docker in to the registry imageName is in.
// It's a shell pipeline so the password goes over stdin and never shows up
// in a process list or in dry-run output. Returns "" for an image that
// isn't in ECR (an image override pointing at another registry).
func GenerateECRLoginCommand(svc serviceinfo.ServiceInfo, imageName string) string {
	host := RegistryOf(imageName)
	if !IsECRHost(host) {
		return ""
	}
	login := withTarget([]string{"aws", "ecr", "get-login-password"}, withRegion(svc, ecrRegion(host)))
	return quoteAll(login) + " | docker login --username AWS --password-stdin " + shellutil.Quote(host)
}

// GenerateCreateRepositoryCommand creates imageName's ECR repository if it
// doesn't exist, with scan on push on. Returns "" unless
// aws.create_repository is set, so an account where Terraform owns the
// repositories never gets one created behind its back.
func GenerateCreateRepositoryCommand(svc serviceinfo.ServiceInfo, imageName string) string {
	host := RegistryOf(imageName)
	if !ParseConfig(svc).CreateRepository || !IsECRHost(host) {
		return ""
	}
	target := withRegion(svc, ecrRegion(host))
	repo := RepositoryName(imageName)
	describe := withTarget([]string{"aws", "ecr", "describe-repositories", "--repository-names", repo}, target)
	create := withTarget([]string{"aws", "ecr", "create-repository", "--repository-name", repo,
		"--image-scanning-configuration", "scanOnPush=true"}, target)
	return quoteAll(describe) + " >/dev/null 2>&1 || " + quoteAll(create) + " >/dev/null"
}

// IsECRHost reports whether host is a private ECR registry
// (<account>.dkr.ecr.<region>.amazonaws.com, or .com.cn in China).
func IsECRHost(host string) bool {
	return strings.Contains(host, ".dkr.ecr.") && strings.Contains(host, ".amazonaws.com")
}

// RepositoryName is the ECR repository of an image reference: everything
// between the registry host and the tag or digest.
func RepositoryName(imageName string) string {
	repo := imageName
	host := RegistryOf(imageName)
	if host != "" {
		repo = strings.TrimPrefix(repo, host+"/")
	}
	repo, _, _ = strings.Cut(repo, "@")
	idx := strings.LastIndex(repo, ":")
	if idx != -1 {
		repo = repo[:idx]
	}
	return repo
}

// RegistryOf returns the registry host of an image reference, or "" when
// it has none (a Docker Hub short name).
func RegistryOf(imageName string) string {
	host, _, found := strings.Cut(imageName, "/")
	if !found || !strings.ContainsAny(host, ".:") {
		return ""
	}
	return host
}

// ecrRegion reads the region out of an ECR host, so login always targets
// the registry the image is in, even for an image override in another
// region than the service.
func ecrRegion(host string) string {
	_, rest, found := strings.Cut(host, ".dkr.ecr.")
	if !found {
		return ""
	}
	region, _, _ := strings.Cut(rest, ".")
	return region
}

func withRegion(svc serviceinfo.ServiceInfo, region string) serviceinfo.ServiceInfo {
	if region != "" {
		svc.Region = region
	}
	return svc
}

func quoteAll(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellutil.Quote(arg)
	}
	return strings.Join(quoted, " ")
}
