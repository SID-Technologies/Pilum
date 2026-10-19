package lambdadeploy_test

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/lib/lambdadeploy"

	"github.com/stretchr/testify/require"
)

const image = "123456789012.dkr.ecr.us-east-1.amazonaws.com/fn:v2"

// fakeLambda answers aws lambda commands by subcommand and records them.
type fakeLambda struct {
	responses map[string][]string // subcommand -> answers, last one repeats
	errs      map[string]error
	calls     [][]string
}

func (f *fakeLambda) run(cmd []string) (string, error) {
	f.calls = append(f.calls, cmd)
	sub := cmd[2]
	if err, ok := f.errs[sub]; ok {
		return "", err
	}
	answers := f.responses[sub]
	if len(answers) == 0 {
		return "", fmt.Errorf("unexpected command: %s", strings.Join(cmd, " "))
	}
	answer := answers[0]
	if len(answers) > 1 {
		f.responses[sub] = answers[1:]
	}
	return answer, nil
}

func (f *fakeLambda) called(sub string) [][]string {
	var out [][]string
	for _, c := range f.calls {
		if c[2] == sub {
			out = append(out, c)
		}
	}
	return out
}

func config(packageType, updateStatus string) string {
	return fmt.Sprintf(`{"FunctionName": "fn", "Version": "$LATEST", "PackageType": %q, "State": "Active",
	  "LastUpdateStatus": %q, "LastUpdateStatusReason": "the image manifest is not supported", "CodeSha256": "abc123"}`,
		packageType, updateStatus)
}

func fakeClock() lambdadeploy.Clock {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	return lambdadeploy.Clock{Now: func() time.Time { return now }, Sleep: func(d time.Duration) { now = now.Add(d) }}
}

func imageRequest() lambdadeploy.Request {
	return lambdadeploy.Request{
		Target:      aws.Target{Region: "us-east-1"},
		Function:    "fn",
		Alias:       "live",
		Image:       image,
		Description: "pilum v2",
		WaitTimeout: time.Minute,
		Interval:    3 * time.Second,
	}
}

func TestDeployImage(t *testing.T) {
	t.Parallel()

	fake := &fakeLambda{responses: map[string][]string{
		// Before the code update, then in progress, then done.
		"get-function-configuration": {config("Image", "Successful"), config("Image", "InProgress"), config("Image", "Successful")},
		"update-function-code":       {`{}`},
		"publish-version":            {`{"Version": "13"}`},
		"list-aliases":               {`{"Aliases": [{"Name": "live", "FunctionVersion": "12"}]}`},
		"update-alias":               {`{}`},
	}}

	result, err := lambdadeploy.Deploy(imageRequest(), fake.run, fakeClock())
	require.NoError(t, err)
	require.Equal(t, lambdadeploy.Result{Version: "13", Previous: "12"}, result)

	code := strings.Join(fake.called("update-function-code")[0], " ")
	require.Contains(t, code, "--function-name fn")
	require.Contains(t, code, "--image-uri "+image)
	require.Contains(t, code, "--region us-east-1")
	// Nothing to configure: no configuration update.
	require.Empty(t, fake.called("update-function-configuration"))

	publish := strings.Join(fake.called("publish-version")[0], " ")
	require.Contains(t, publish, "--code-sha256 abc123 --description pilum v2")
	require.Contains(t, strings.Join(fake.called("update-alias")[0], " "), "--name live --function-version 13")
}

func TestDeploySettingsAndNewAlias(t *testing.T) {
	t.Parallel()

	fake := &fakeLambda{
		responses: map[string][]string{
			"get-function-configuration":    {config("Image", "Successful")},
			"update-function-code":          {`{}`},
			"update-function-configuration": {`{}`},
			"publish-version":               {`{"Version": "1"}`},
			"create-alias":                  {`{}`},
			"list-aliases":                  {`{"Aliases": []}`},
		},
	}

	req := imageRequest()
	req.Memory = 512
	req.Timeout = 30
	req.Architecture = "arm64"
	req.SetEnv = true
	req.Env = []string{"LOG_LEVEL=debug", "A=1"}

	result, err := lambdadeploy.Deploy(req, fake.run, fakeClock())
	require.NoError(t, err)
	require.Equal(t, lambdadeploy.Result{Version: "1"}, result)

	require.Contains(t, strings.Join(fake.called("update-function-code")[0], " "), "--architectures arm64")
	settings := fake.called("update-function-configuration")[0]
	require.Equal(t, []string{"aws", "lambda", "update-function-configuration", "--function-name", "fn", "--output", "json",
		"--memory-size", "512", "--timeout", "30", "--environment", `{"Variables":{"A":"1","LOG_LEVEL":"debug"}}`,
		"--region", "us-east-1"}, settings)
	require.Contains(t, strings.Join(fake.called("create-alias")[0], " "), "--name live --function-version 1")
}

func TestDeployRerunKeepsAlias(t *testing.T) {
	t.Parallel()

	// Unchanged code: Lambda returns the version it already published.
	fake := &fakeLambda{responses: map[string][]string{
		"get-function-configuration": {config("Image", "Successful")},
		"update-function-code":       {`{}`},
		"publish-version":            {`{"Version": "13"}`},
		"list-aliases":               {`{"Aliases": [{"Name": "live", "FunctionVersion": "13"}]}`},
	}}
	result, err := lambdadeploy.Deploy(imageRequest(), fake.run, fakeClock())
	require.NoError(t, err)
	require.Equal(t, lambdadeploy.Result{Version: "13", Previous: "13"}, result)
	require.Empty(t, fake.called("update-alias"))
}

func TestDeployFailures(t *testing.T) {
	t.Parallel()

	fake := &fakeLambda{responses: map[string][]string{"get-function-configuration": {config("Zip", "Successful")}}}
	_, err := lambdadeploy.Deploy(imageRequest(), fake.run, fakeClock())
	require.ErrorContains(t, err, "function fn is a Zip function; use type aws-lambda")

	fake = &fakeLambda{responses: map[string][]string{
		"get-function-configuration": {config("Image", "Successful"), config("Image", "Failed")},
		"update-function-code":       {`{}`},
	}}
	_, err = lambdadeploy.Deploy(imageRequest(), fake.run, fakeClock())
	require.ErrorContains(t, err, "update of fn failed: the image manifest is not supported")
	require.Empty(t, fake.called("publish-version"))

	fake = &fakeLambda{responses: map[string][]string{
		"get-function-configuration": {config("Image", "Successful"), config("Image", "InProgress")},
		"update-function-code":       {`{}`},
	}}
	_, err = lambdadeploy.Deploy(imageRequest(), fake.run, fakeClock())
	require.ErrorContains(t, err, "function fn was still updating after 1m0s")

	fake = &fakeLambda{errs: map[string]error{"get-function-configuration": fmt.Errorf("ResourceNotFoundException")}}
	_, err = lambdadeploy.Deploy(imageRequest(), fake.run, fakeClock())
	require.ErrorContains(t, err, "Pilum deploys to existing functions")

	split := &fakeLambda{responses: map[string][]string{
		"get-function-configuration": {config("Image", "Successful")},
		"update-function-code":       {`{}`},
		"publish-version":            {`{"Version": "13"}`},
		"list-aliases":               {`{"Aliases": [{"Name": "live", "FunctionVersion": "12", "RoutingConfig": {"AdditionalVersionWeights": {"13": 0.1}}}]}`},
	}}
	_, err = lambdadeploy.Deploy(imageRequest(), split.run, fakeClock())
	require.ErrorContains(t, err, "splits traffic between versions")
}

func buildDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bootstrap"), []byte("#!/bin/sh\n"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "static"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "static", "index.html"), []byte("<p>hi</p>"), 0o644))
	return dir
}

func TestDeployZip(t *testing.T) {
	t.Parallel()

	fake := &fakeLambda{responses: map[string][]string{
		"get-function-configuration":    {config("Zip", "Successful")},
		"update-function-code":          {`{}`},
		"update-function-configuration": {`{}`},
		"publish-version":               {`{"Version": "2"}`},
		"list-aliases":                  {`{"Aliases": [{"Name": "live", "FunctionVersion": "1"}]}`},
		"update-alias":                  {`{}`},
	}}
	req := imageRequest()
	req.Image = ""
	req.Zip = buildDir(t)
	req.Runtime = "provided.al2023"
	req.Handler = "bootstrap"

	_, err := lambdadeploy.Deploy(req, fake.run, fakeClock())
	require.NoError(t, err)

	code := fake.called("update-function-code")[0]
	idx := indexOf(code, "--zip-file")
	require.NotEqual(t, -1, idx)
	zipPath := strings.TrimPrefix(code[idx+1], "fileb://")
	// The temp zip is removed after the deploy.
	_, statErr := os.Stat(zipPath)
	require.True(t, os.IsNotExist(statErr))

	require.Contains(t, strings.Join(fake.called("update-function-configuration")[0], " "),
		"--runtime provided.al2023 --handler bootstrap")
}

func TestDeployZipThroughS3(t *testing.T) {
	t.Parallel()

	fake := &fakeLambda{responses: map[string][]string{
		"get-function-configuration": {config("Zip", "Successful")},
		"cp":                         {""},
		"update-function-code":       {`{}`},
		"publish-version":            {`{"Version": "2"}`},
		"list-aliases":               {`{"Aliases": [{"Name": "live", "FunctionVersion": "2"}]}`},
	}}
	req := imageRequest()
	req.Image = ""
	req.Zip = buildDir(t)
	req.S3Bucket = "artifacts"

	_, err := lambdadeploy.Deploy(req, fake.run, fakeClock())
	require.NoError(t, err)

	cp := fake.called("cp")[0]
	require.Equal(t, "s3", cp[1])
	require.True(t, strings.HasPrefix(cp[4], "s3://artifacts/fn/"), cp[4])
	key := strings.TrimPrefix(cp[4], "s3://artifacts/")
	require.Contains(t, strings.Join(fake.called("update-function-code")[0], " "), "--s3-bucket artifacts --s3-key "+key)
}

func TestPackage(t *testing.T) {
	t.Parallel()

	dir := buildDir(t)
	first, cleanup, err := lambdadeploy.Package(dir)
	require.NoError(t, err)
	defer cleanup()

	zr, err := zip.OpenReader(first)
	require.NoError(t, err)
	defer zr.Close()
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
		if f.Name == "bootstrap" {
			require.Equal(t, os.FileMode(0o755), f.Mode().Perm(), "bootstrap must stay executable")
		}
	}
	require.Equal(t, []string{"bootstrap", "static/index.html"}, names)

	// Same files, same zip: Lambda then sees unchanged code.
	second, cleanup2, err := lambdadeploy.Package(dir)
	require.NoError(t, err)
	defer cleanup2()
	require.Equal(t, sum(t, first), sum(t, second))

	// A .zip is used as is.
	path, _, err := lambdadeploy.Package(first)
	require.NoError(t, err)
	require.Equal(t, first, path)

	_, _, err = lambdadeploy.Package(filepath.Join(dir, "bootstrap"))
	require.ErrorContains(t, err, "not a .zip")
	_, _, err = lambdadeploy.Package(t.TempDir())
	require.ErrorContains(t, err, "has no files")
	_, _, err = lambdadeploy.Package(filepath.Join(dir, "missing"))
	require.ErrorContains(t, err, "set lambda.artifact")
}

func TestParseVersions(t *testing.T) {
	t.Parallel()

	versions, err := lambdadeploy.ParseVersions(`{"Versions": [{"Version": "$LATEST"}, {"Version": "2", "Description": "pilum v2"},
	  {"Version": "10", "Description": "hand made"}]}`)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	require.Equal(t, 10, versions[0].Number)
	require.Empty(t, versions[0].Tag())
	require.Equal(t, "v2", versions[1].Tag())
}

func sum(t *testing.T, path string) [32]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return sha256.Sum256(data)
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}
