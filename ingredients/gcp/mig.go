package gcp

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// DefaultImageMetadataKey is the instance metadata key that holds the image a
// MIG's VMs run. The template's startup script reads it from the metadata
// server and runs `docker run` with it; Pilum changes only this value. That
// keeps the script (ports, env, flags) owned by whoever set up the MIG.
const DefaultImageMetadataKey = "container-image"

// EnvMetadataKey holds the container's env vars as KEY=VALUE lines, for the
// startup script to pass to `docker run --env-file`.
const EnvMetadataKey = "container-env"

// SecretsMetadataKey holds NAME=<Secret Manager version resource> lines. Only
// references are stored in metadata, never values: the startup script reads
// each secret from Secret Manager on boot.
const SecretsMetadataKey = "container-secrets"

// LegacyContainerDeclarationKey is the metadata the retired container startup
// agent (konlet, `create-with-container`) read. Since July 31, 2026 GCP no
// longer creates VMs from it, so a template that still uses it is refused.
const LegacyContainerDeclarationKey = "gce-container-declaration"

const maxResourceNameLen = 63

var (
	invalidNameChars = regexp.MustCompile(`[^a-z0-9-]+`)
	resourceName     = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
)

// MIGTemplateName returns the instance template name for a release: <base>-<tag>,
// with the tag lowercased and anything GCP rejects (dots, underscores, plus)
// turned into "-", so v1.4.2 becomes <base>-v1-4-2.
//
// Names are never truncated. Two tags that differ only after the cut would map
// to the same template, and the second deploy would silently ship the first.
func MIGTemplateName(base, tag string) (string, error) {
	if base == "" {
		return "", errors.New("mig.template_base is required")
	}
	if tag == "" || tag == "latest" {
		return "", errors.New("gcp-mig-container needs a unique --tag per release (got %q): "+
			"templates are named <base>-<tag>, and reusing one would make the rollout a no-op", tag)
	}

	suffix := strings.Trim(invalidNameChars.ReplaceAllString(strings.ToLower(tag), "-"), "-")
	name := base + "-" + suffix
	if len(name) > maxResourceNameLen {
		return "", errors.New("instance template name %q is %d characters; GCP allows %d (shorten mig.template_base or the tag)",
			name, len(name), maxResourceNameLen)
	}
	if !resourceName.MatchString(name) {
		return "", errors.New("instance template name %q is invalid: lowercase letters, digits and hyphens, starting with a letter", name)
	}
	return name, nil
}

// MIGTemplateRef is how gcloud refers to a template: a bare name for global
// templates, a resource path for regional ones (bare names resolve as global).
func MIGTemplateRef(svc serviceinfo.ServiceInfo, name string) string {
	cfg := ParseMIGConfig(svc.Config)
	if cfg.TemplateRegion == "" {
		return name
	}
	return fmt.Sprintf("projects/%s/regions/%s/instanceTemplates/%s", svc.Project, cfg.TemplateRegion, name)
}

// GenerateMIGRollingUpdateCommand starts a rolling update of the MIG to template.
// It returns once the update is accepted, not when it finishes.
func GenerateMIGRollingUpdateCommand(svc serviceinfo.ServiceInfo, template string) []string {
	cfg := ParseMIGConfig(svc.Config)
	cmd := []string{
		"gcloud", "compute", "instance-groups", "managed", "rolling-action", "start-update", cfg.Name,
		"--version", "template=" + MIGTemplateRef(svc, template),
		"--max-surge", cfg.MaxSurge,
		"--max-unavailable", cfg.MaxUnavailable,
	}
	cmd = append(cmd, cfg.LocationFlags()...)
	return withProject(cmd, svc.Project)
}

// GenerateMIGWaitStableCommand blocks until the MIG has no pending actions.
// gcloud exits non-zero if that doesn't happen within mig.wait_timeout.
func GenerateMIGWaitStableCommand(svc serviceinfo.ServiceInfo) []string {
	cfg := ParseMIGConfig(svc.Config)
	cmd := []string{
		"gcloud", "compute", "instance-groups", "managed", "wait-until", cfg.Name,
		"--stable",
		"--timeout", strconv.Itoa(cfg.WaitTimeout),
	}
	cmd = append(cmd, cfg.LocationFlags()...)
	return withProject(cmd, svc.Project)
}

// GenerateMIGDescribeCommand describes the MIG as JSON.
func GenerateMIGDescribeCommand(svc serviceinfo.ServiceInfo) []string {
	cfg := ParseMIGConfig(svc.Config)
	cmd := []string{
		"gcloud", "compute", "instance-groups", "managed", "describe", cfg.Name,
		"--format", "json",
	}
	cmd = append(cmd, cfg.LocationFlags()...)
	return withProject(cmd, svc.Project)
}

// GenerateListMIGTemplatesCommand lists the templates in the base's family:
// the base itself (often what the setup script created) and every <base>-<tag>.
// Lists global and regional templates alike.
func GenerateListMIGTemplatesCommand(svc serviceinfo.ServiceInfo) []string {
	base := ParseMIGConfig(svc.Config).TemplateBase
	cmd := []string{
		"gcloud", "compute", "instance-templates", "list",
		"--filter", fmt.Sprintf("name=%s OR name~^%s-", base, regexp.QuoteMeta(base)),
		"--format", "json",
	}
	return withProject(cmd, svc.Project)
}

// GenerateDescribeTemplateCommand describes a template by name or self link.
func GenerateDescribeTemplateCommand(project, template string) []string {
	cmd := []string{"gcloud", "compute", "instance-templates", "describe", template, "--format", "json"}
	return withProject(cmd, project)
}

// GenerateMIGTemplateArgs returns the arguments of the hidden `pilum
// mig-template` command for a release (everything after the executable).
// Env vars, secrets and VM overrides are only passed when the service sets
// them; anything not passed is kept from the MIG's current template.
func GenerateMIGTemplateArgs(svc serviceinfo.ServiceInfo, name, image string) []string {
	cfg := ParseMIGConfig(svc.Config)
	args := []string{"mig-template",
		"--project", svc.Project,
		"--mig", cfg.Name,
		"--name", name,
		"--image", image,
		"--image-key", cfg.ImageMetadataKey,
	}
	args = append(args, cfg.LocationFlags()...)

	optional := []struct{ flag, value string }{
		{"--template-region", cfg.TemplateRegion},
		{"--machine-type", cfg.MachineType},
		{"--network", cfg.Network},
		{"--subnet", cfg.Subnet},
		{"--service-account", cfg.ServiceAccount},
	}
	for _, o := range optional {
		if o.value != "" {
			args = append(args, o.flag, o.value)
		}
	}
	if cfg.Subnet != "" {
		args = append(args, "--subnet-region", cfg.SubnetRegion())
	}
	if cfg.ExternalIP != nil {
		args = append(args, "--external-ip="+strconv.FormatBool(*cfg.ExternalIP))
	}
	if cfg.SetNetworkTags {
		args = append(args, "--network-tags="+strings.Join(cfg.NetworkTags, ","))
	}

	env := make([]string, 0, len(svc.EnvVars))
	for _, e := range svc.EnvVars {
		env = append(env, e.Name+"="+e.Value)
	}
	slices.Sort(env)
	for _, e := range env {
		args = append(args, "--env", e)
	}

	secrets := make([]string, 0, len(svc.Secrets))
	for _, sec := range svc.Secrets {
		secrets = append(secrets, sec.Name+"="+SecretVersionResource(svc.Project, sec.Value))
	}
	slices.Sort(secrets)
	for _, sec := range secrets {
		args = append(args, "--secret", sec)
	}

	return args
}

// SecretVersionResource turns a secret reference into the Secret Manager
// version resource the startup script reads:
// projects/<p>/secrets/<name>/versions/<version>. It accepts the forms Cloud
// Run services take: "name", "name:version", or a full resource path.
func SecretVersionResource(project, value string) string {
	if strings.HasPrefix(value, "projects/") {
		if strings.Contains(value, "/versions/") {
			return value
		}
		return value + "/versions/latest"
	}
	name, version, found := strings.Cut(value, ":")
	if !found || version == "" {
		version = "latest"
	}
	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", project, name, version)
}
