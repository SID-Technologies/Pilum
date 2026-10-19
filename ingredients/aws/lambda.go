package aws

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/sid-technologies/pilum/lib/configutil"
	"github.com/sid-technologies/pilum/lib/errors"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

const (
	// DefaultLambdaAlias is the alias each deploy moves to its new version.
	// Point triggers (API Gateway, event sources, other services) at
	// <function>:live, so a deploy and a rollback are one alias move.
	DefaultLambdaAlias = "live"
	// DefaultLambdaArtifact is what a zip function deploys: the service's
	// dist/ directory, zipped, or a .zip file.
	DefaultLambdaArtifact = "dist"
	// DefaultLambdaWaitTimeout is how long, in seconds, a deploy waits for
	// each function update to finish.
	DefaultLambdaWaitTimeout = 300
	// LambdaVersionPrefix starts each published version's description,
	// followed by the release tag, so rollback --to <tag> can find it.
	LambdaVersionPrefix = "pilum "
)

// LambdaConfig is the lambda: block of an aws-lambda or aws-lambda-image
// service. Zero values keep what the function has.
type LambdaConfig struct {
	Function     string // Function name or ARN. Defaults to the service name.
	Alias        string
	Memory       int    // MB
	Timeout      int    // Seconds
	Runtime      string // Zip functions only
	Handler      string // Zip functions only
	Architecture string // x86_64 or arm64
	Artifact     string // Zip functions: file or directory, relative to the service directory
	S3Bucket     string // Upload zips through this bucket; needed over 50 MB
	WaitTimeout  int
}

// ParseLambdaConfig reads the lambda: block of a service.
func ParseLambdaConfig(svc serviceinfo.ServiceInfo) LambdaConfig {
	m := configutil.MapFromAny(svc.Config["lambda"])
	return LambdaConfig{
		Function:     configutil.GetString(m, "function", svc.Name),
		Alias:        configutil.GetString(m, "alias", DefaultLambdaAlias),
		Memory:       configutil.GetInt(m, "memory", 0),
		Timeout:      configutil.GetInt(m, "timeout", 0),
		Runtime:      configutil.GetString(m, "runtime", ""),
		Handler:      configutil.GetString(m, "handler", ""),
		Architecture: configutil.GetString(m, "architecture", ""),
		Artifact:     configutil.GetString(m, "artifact", DefaultLambdaArtifact),
		S3Bucket:     configutil.GetString(m, "s3_bucket", ""),
		WaitTimeout:  configutil.GetInt(m, "wait_timeout", DefaultLambdaWaitTimeout),
	}
}

// IsLambdaImage reports whether a service deploys a container image to Lambda.
func IsLambdaImage(svc serviceinfo.ServiceInfo) bool {
	return svc.RecipeKey() == "aws-lambda-image"
}

// LambdaPlatform is the docker platform for a Lambda image: linux/arm64
// for an arm64 function, else "" (the build's default, linux/amd64).
func LambdaPlatform(svc serviceinfo.ServiceInfo) string {
	if IsLambdaImage(svc) && ParseLambdaConfig(svc).Architecture == "arm64" {
		return "linux/arm64"
	}
	return ""
}

// GenerateLambdaDeployArgs returns the arguments of the hidden `pilum
// lambda-deploy` command (everything after the executable). For an image
// function image is the pushed image; a zip function deploys its artifact.
func GenerateLambdaDeployArgs(svc serviceinfo.ServiceInfo, image, tag string) []string {
	cfg := ParseLambdaConfig(svc)
	args := []string{"lambda-deploy",
		"--function", cfg.Function,
		"--alias", cfg.Alias,
		"--version-description", LambdaVersionPrefix + tag,
		"--wait-timeout", strconv.Itoa(cfg.WaitTimeout),
	}
	if IsLambdaImage(svc) {
		args = append(args, "--image", image)
	} else {
		args = append(args, "--zip", LambdaArtifactPath(svc))
	}
	args = append(args, TargetOf(svc).Flags()...)

	optional := []struct{ flag, value string }{
		{"--runtime", cfg.Runtime},
		{"--handler", cfg.Handler},
		{"--architecture", cfg.Architecture},
		{"--s3-bucket", cfg.S3Bucket},
	}
	for _, o := range optional {
		if o.value != "" {
			args = append(args, o.flag, o.value)
		}
	}
	if cfg.Memory > 0 {
		args = append(args, "--memory", strconv.Itoa(cfg.Memory))
	}
	if cfg.Timeout > 0 {
		args = append(args, "--timeout-seconds", strconv.Itoa(cfg.Timeout))
	}

	env := make([]string, 0, len(svc.EnvVars))
	for _, e := range svc.EnvVars {
		env = append(env, e.Name+"="+e.Value)
	}
	slices.Sort(env)
	for _, e := range env {
		args = append(args, "--env", e)
	}
	return args
}

// LambdaArtifactPath is the zip function's artifact, relative to where
// pilum runs.
func LambdaArtifactPath(svc serviceinfo.ServiceInfo) string {
	artifact := ParseLambdaConfig(svc).Artifact
	if filepath.IsAbs(artifact) {
		return artifact
	}
	return filepath.Join(svc.Path, artifact)
}

// GetFunctionConfigurationCommand reads a function's (or a version's or
// alias's) configuration.
func GetFunctionConfigurationCommand(target Target, function, qualifier string) []string {
	cmd := []string{"aws", "lambda", "get-function-configuration", "--function-name", function, "--output", "json"}
	if qualifier != "" {
		cmd = append(cmd, "--qualifier", qualifier)
	}
	return append(cmd, target.Flags()...)
}

// ListAliasesCommand lists a function's aliases.
func ListAliasesCommand(target Target, function string) []string {
	cmd := []string{"aws", "lambda", "list-aliases", "--function-name", function, "--output", "json"}
	return append(cmd, target.Flags()...)
}

// ListVersionsCommand lists a function's published versions (and $LATEST).
func ListVersionsCommand(target Target, function string) []string {
	cmd := []string{"aws", "lambda", "list-versions-by-function", "--function-name", function, "--output", "json"}
	return append(cmd, target.Flags()...)
}

// UpdateAliasCommand points an alias at a version.
func UpdateAliasCommand(target Target, function, alias, version string) []string {
	cmd := []string{"aws", "lambda", "update-alias", "--function-name", function, "--name", alias,
		"--function-version", version, "--output", "json"}
	return append(cmd, target.Flags()...)
}

// CreateAliasCommand creates an alias pointing at a version.
func CreateAliasCommand(target Target, function, alias, version string) []string {
	cmd := []string{"aws", "lambda", "create-alias", "--function-name", function, "--name", alias,
		"--function-version", version, "--output", "json"}
	return append(cmd, target.Flags()...)
}

// GenerateLambdaStatusCommand describes the version the alias points at.
func GenerateLambdaStatusCommand(svc serviceinfo.ServiceInfo) []string {
	cfg := ParseLambdaConfig(svc)
	return GetFunctionConfigurationCommand(TargetOf(svc), cfg.Function, cfg.Alias)
}

// ValidateLambda catches Lambda configs that would deploy wrong. Lambda has
// no secret references (unlike ECS), and a SAM template next to an
// aws-lambda service means it was written for the SAM recipe, which had
// this name before v0.12.
func ValidateLambda(svc serviceinfo.ServiceInfo) error {
	key := svc.RecipeKey()
	if key != "aws-lambda" && key != "aws-lambda-image" {
		return nil
	}
	if len(svc.Secrets) > 0 {
		return errors.New("lambda has no secret references: read secrets at run time " +
			"(Secrets Manager, or the Parameters and Secrets Lambda extension) and remove secrets from pilum.yaml")
	}
	if key == "aws-lambda" {
		for _, name := range []string{"template.yaml", "template.yml"} {
			_, err := os.Stat(filepath.Join(svc.Path, name))
			if err == nil {
				return errors.New("%s is a SAM template: use type: aws-lambda-sam (aws-lambda now deploys to an existing function without SAM)", name)
			}
		}
	}
	return nil
}
