// Package lambdadeploy ships a new version of an existing Lambda function.
//
// Terraform (or whoever set the function up) owns the function, its role
// and its triggers. Pilum updates the code (a zip or a container image) and
// the settings pilum.yaml sets, publishes a version, and moves an alias
// (live by default) to it. Triggers that call <function>:live switch to the
// new version in one step, and a rollback is moving the alias back.
package lambdadeploy

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/lib/errors"
)

// maxDirectUpload is the largest zip update-function-code takes inline.
// Bigger ones go through S3.
const maxDirectUpload = 50 << 20

// Runner executes an aws command and returns its stdout.
type Runner func(cmd []string) (string, error)

// Clock lets tests run waits without sleeping.
type Clock struct {
	Now   func() time.Time
	Sleep func(time.Duration)
}

// Request is the function to deploy and what changes.
type Request struct {
	aws.Target
	Function string
	Alias    string
	Image    string // Image functions: the image URI to deploy
	Zip      string // Zip functions: a .zip file, or a directory to zip

	S3Bucket     string // Upload the zip here first (required over 50 MB)
	Memory       int    // MB, 0 keeps the current setting
	Timeout      int    // Seconds, 0 keeps the current setting
	Runtime      string // "" keeps the current setting
	Handler      string // "" keeps the current setting
	Architecture string // "" keeps the current setting

	Env    []string // KEY=VALUE
	SetEnv bool     // Replace the function's environment (otherwise kept as is)

	Description string // Published version's description ("pilum <tag>")
	WaitTimeout time.Duration
	Interval    time.Duration
}

// Result says what Deploy did.
type Result struct {
	Version  string // Version the alias points at now
	Previous string // Version it pointed at before, "" if the alias is new
}

// function is the part of get-function-configuration output Pilum reads.
type function struct {
	FunctionName           string `json:"FunctionName"`
	Version                string `json:"Version"`
	PackageType            string `json:"PackageType"`
	CodeSha256             string `json:"CodeSha256"`
	State                  string `json:"State"`
	StateReason            string `json:"StateReason"`
	LastUpdateStatus       string `json:"LastUpdateStatus"`
	LastUpdateStatusReason string `json:"LastUpdateStatusReason"`
	Description            string `json:"Description"`
	LastModified           string `json:"LastModified"`
}

// Deploy updates the function's code and settings, publishes a version and
// points the alias at it. Lambda only publishes a new version when code or
// settings changed, so a rerun moves the alias to the version it already
// published instead of making another.
func Deploy(req Request, run Runner, clock Clock) (Result, error) {
	if req.Function == "" || req.Alias == "" || (req.Image == "") == (req.Zip == "") {
		return Result{}, errors.New("function, alias and one of image or zip are required")
	}

	current, err := ready(req, run, clock)
	if err != nil {
		return Result{}, err
	}
	wantType := "Zip"
	if req.Image != "" {
		wantType = "Image"
	}
	if current.PackageType != "" && current.PackageType != wantType {
		return Result{}, errors.New("function %s is a %s function; use type %s", req.Function, current.PackageType,
			map[string]string{"Zip": "aws-lambda", "Image": "aws-lambda-image"}[current.PackageType])
	}

	codeArgs, cleanup, err := codeSource(req, run)
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	code := []string{"aws", "lambda", "update-function-code", "--function-name", req.Function, "--output", "json"}
	code = append(code, codeArgs...)
	if req.Architecture != "" {
		code = append(code, "--architectures", req.Architecture)
	}
	_, err = run(append(code, req.Flags()...))
	if err != nil {
		return Result{}, errors.Wrap(err, "updating the code of %s", req.Function)
	}
	updated, err := ready(req, run, clock)
	if err != nil {
		return Result{}, err
	}

	settings := configurationArgs(req)
	if len(settings) > 0 {
		cmd := append([]string{"aws", "lambda", "update-function-configuration", "--function-name", req.Function,
			"--output", "json"}, settings...)
		_, err = run(append(cmd, req.Flags()...))
		if err != nil {
			return Result{}, errors.Wrap(err, "updating the configuration of %s", req.Function)
		}
		updated, err = ready(req, run, clock)
		if err != nil {
			return Result{}, err
		}
	}

	// --code-sha256 refuses to publish if someone else changed the code
	// since this deploy uploaded it.
	publish := []string{"aws", "lambda", "publish-version", "--function-name", req.Function,
		"--code-sha256", updated.CodeSha256, "--description", req.Description, "--output", "json"}
	published, err := run(append(publish, req.Flags()...))
	if err != nil {
		return Result{}, errors.Wrap(err, "publishing a version of %s", req.Function)
	}
	var version function
	err = json.Unmarshal([]byte(published), &version)
	if err != nil || version.Version == "" {
		return Result{}, errors.New("publish-version returned no version for %s", req.Function)
	}

	previous, err := moveAlias(req.Target, req.Function, req.Alias, version.Version, run)
	if err != nil {
		return Result{}, err
	}
	return Result{Version: version.Version, Previous: previous}, nil
}

// codeSource returns update-function-code's code arguments, and a cleanup
// for the zip it may have written.
func codeSource(req Request, run Runner) ([]string, func(), error) {
	noop := func() {}
	if req.Image != "" {
		return []string{"--image-uri", req.Image}, noop, nil
	}

	path, cleanup, err := Package(req.Zip)
	if err != nil {
		return nil, noop, err
	}
	info, err := os.Stat(path)
	if err != nil {
		cleanup()
		return nil, noop, errors.Wrap(err, "reading %s", path)
	}

	if req.S3Bucket == "" {
		if info.Size() > maxDirectUpload {
			cleanup()
			return nil, noop, errors.New("%s is %d MB; Lambda takes zips over 50 MB only from S3, so set lambda.s3_bucket",
				req.Zip, info.Size()>>20)
		}
		return []string{"--zip-file", "fileb://" + path}, cleanup, nil
	}

	sum, err := fileSHA256(path)
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	key := req.Function + "/" + sum + ".zip"
	_, err = run(append([]string{"aws", "s3", "cp", path, "s3://" + req.S3Bucket + "/" + key, "--only-show-errors"},
		req.Flags()...))
	if err != nil {
		cleanup()
		return nil, noop, errors.Wrap(err, "uploading %s to s3://%s", req.Zip, req.S3Bucket)
	}
	return []string{"--s3-bucket", req.S3Bucket, "--s3-key", key}, cleanup, nil
}

// configurationArgs returns update-function-configuration's arguments for
// the settings the service sets, or nil when it sets none.
func configurationArgs(req Request) []string {
	var args []string
	if req.Memory > 0 {
		args = append(args, "--memory-size", strconv.Itoa(req.Memory))
	}
	if req.Timeout > 0 {
		args = append(args, "--timeout", strconv.Itoa(req.Timeout))
	}
	if req.Runtime != "" {
		args = append(args, "--runtime", req.Runtime)
	}
	if req.Handler != "" {
		args = append(args, "--handler", req.Handler)
	}
	if req.SetEnv {
		args = append(args, "--environment", environmentJSON(req.Env))
	}
	return args
}

// environmentJSON is the --environment value for KEY=VALUE pairs.
func environmentJSON(pairs []string) string {
	vars := map[string]string{}
	for _, pair := range pairs {
		name, value, _ := strings.Cut(pair, "=")
		vars[name] = value
	}
	data, err := json.Marshal(map[string]any{"Variables": vars})
	if err != nil {
		return `{"Variables":{}}`
	}
	return string(data)
}

// ready waits until the function has no update in progress, and fails if
// the last one failed. Lambda rejects an update while another is running,
// and runs a new one asynchronously, so each step waits for the one before.
func ready(req Request, run Runner, clock Clock) (function, error) {
	deadline := clock.Now().Add(req.WaitTimeout)
	for {
		described, err := run(aws.GetFunctionConfigurationCommand(req.Target, req.Function, ""))
		if err != nil {
			return function{}, errors.Wrap(err, "reading function %s (Pilum deploys to existing functions; create it first, e.g. with Terraform)",
				req.Function)
		}
		var fn function
		err = json.Unmarshal([]byte(described), &fn)
		if err != nil {
			return function{}, errors.Wrap(err, "invalid get-function-configuration JSON")
		}

		switch {
		case fn.LastUpdateStatus == "Failed":
			return function{}, errors.New("update of %s failed: %s", req.Function, fn.LastUpdateStatusReason)
		case fn.State == "Failed":
			return function{}, errors.New("function %s failed: %s", req.Function, fn.StateReason)
		case fn.LastUpdateStatus != "InProgress" && fn.State != "Pending":
			return fn, nil
		}

		if !clock.Now().Before(deadline) {
			return function{}, errors.New("function %s was still updating after %s", req.Function, req.WaitTimeout)
		}
		clock.Sleep(req.Interval)
	}
}

// moveAlias points alias at version, creating the alias on the first
// deploy, and returns the version it pointed at before.
func moveAlias(target aws.Target, fn, alias, version string, run Runner) (string, error) {
	current, found, err := AliasVersion(target, fn, alias, run)
	if err != nil {
		return "", err
	}
	if !found {
		_, err = run(aws.CreateAliasCommand(target, fn, alias, version))
		if err != nil {
			return "", errors.Wrap(err, "creating alias %s:%s", fn, alias)
		}
		return "", nil
	}
	if current == version {
		return current, nil
	}
	_, err = run(aws.UpdateAliasCommand(target, fn, alias, version))
	if err != nil {
		return "", errors.Wrap(err, "moving alias %s:%s to version %s", fn, alias, version)
	}
	return current, nil
}

// AliasVersion returns the version an alias points at, and false when the
// function has no such alias. It lists aliases rather than getting the
// one, so a missing alias (every first deploy) isn't a failed command.
func AliasVersion(target aws.Target, fn, alias string, run Runner) (string, bool, error) {
	listed, err := run(aws.ListAliasesCommand(target, fn))
	if err != nil {
		return "", false, errors.Wrap(err, "listing aliases of %s", fn)
	}
	var aliases struct {
		Aliases []struct {
			Name            string `json:"Name"`
			FunctionVersion string `json:"FunctionVersion"`
			RoutingConfig   *struct {
				AdditionalVersionWeights map[string]float64 `json:"AdditionalVersionWeights"`
			} `json:"RoutingConfig"`
		} `json:"Aliases"`
	}
	err = json.Unmarshal([]byte(listed), &aliases)
	if err != nil {
		return "", false, errors.Wrap(err, "invalid list-aliases JSON")
	}
	for _, a := range aliases.Aliases {
		if a.Name != alias {
			continue
		}
		if a.RoutingConfig != nil && len(a.RoutingConfig.AdditionalVersionWeights) > 0 {
			return "", false, errors.New("alias %s:%s splits traffic between versions; finish or undo that split first", fn, alias)
		}
		return a.FunctionVersion, true, nil
	}
	return "", false, nil
}

// Version is one published version of a function.
type Version struct {
	Number      int
	Description string
}

// Tag returns the release tag Pilum published the version with, or "".
func (v Version) Tag() string {
	tag, found := strings.CutPrefix(v.Description, aws.LambdaVersionPrefix)
	if !found {
		return ""
	}
	return tag
}

// ParseVersions reads list-versions-by-function output, skipping $LATEST,
// newest first.
func ParseVersions(listJSON string) ([]Version, error) {
	var listed struct {
		Versions []function `json:"Versions"`
	}
	err := json.Unmarshal([]byte(listJSON), &listed)
	if err != nil {
		return nil, errors.Wrap(err, "invalid list-versions-by-function JSON")
	}
	versions := make([]Version, 0, len(listed.Versions))
	for _, v := range listed.Versions {
		n, convErr := strconv.Atoi(v.Version)
		if convErr != nil {
			continue // $LATEST
		}
		versions = append(versions, Version{Number: n, Description: v.Description})
	}
	slices.SortFunc(versions, func(a, b Version) int { return b.Number - a.Number })
	return versions, nil
}

// FormatVersion names a version the way rollback output shows it.
func FormatVersion(fn string, number int) string {
	return fmt.Sprintf("%s:%d", fn, number)
}
