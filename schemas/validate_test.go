package schemas_test

import (
	"testing"

	"github.com/sid-technologies/pilum/schemas"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func validate(t *testing.T, config string) []schemas.Problem {
	t.Helper()

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(config), &parsed))
	problems, err := schemas.Validate(parsed)
	require.NoError(t, err)
	return problems
}

func problemStrings(problems []schemas.Problem) []string {
	out := make([]string, 0, len(problems))
	for _, p := range problems {
		out = append(out, p.String())
	}
	return out
}

func TestValidateValidConfig(t *testing.T) {
	t.Parallel()

	problems := validate(t, `
name: api
provider: gcp
type: gcp-cloud-run
project: p
region: us-central1
registry_name: r
template: go-api
cloud_run:
  min_instances: 0
  cpu_throttling: false
env_vars:
  - name: PORT
    value: 8080
build:
  language: go
  env_vars:
    CGO_ENABLED: 0
custom_recipe_field: allowed at the top level
`)
	require.Empty(t, problems)
}

func TestValidateUnknownBlockKey(t *testing.T) {
	t.Parallel()

	problems := validate(t, "name: api\ncloud_run:\n  cpu_throtling: false\n")
	require.Equal(t, []string{"cloud_run.cpu_throtling: unknown key - did you mean 'cpu_throttling'?"}, problemStrings(problems))
	require.False(t, problems[0].Warning)
}

func TestValidateWrongType(t *testing.T) {
	t.Parallel()

	// Pilum ignores a quoted number for an integer setting.
	problems := validate(t, "name: api\ncloud_run:\n  min_instances: \"2\"\n")
	require.Equal(t, []string{"cloud_run.min_instances: got string, want integer"}, problemStrings(problems))
	require.False(t, problems[0].Warning)
}

func TestValidateNumberForStringIsWarning(t *testing.T) {
	t.Parallel()

	problems := validate(t, "name: api\nbuild:\n  version: 1.20\n")
	require.Len(t, problems, 1)
	require.Equal(t, "build.version", problems[0].Path)
	require.True(t, problems[0].Warning)
	require.Contains(t, problems[0].Message, "quote it")
}

func TestValidateStringMapReportsTheShapeUsed(t *testing.T) {
	t.Parallel()

	// A map with a bad value reports the value, not "want array".
	problems := validate(t, "name: api\nenv_vars:\n  A: [1]\n")
	require.Equal(t, []string{"env_vars.A: got array, want boolean or number or string"}, problemStrings(problems))

	// Neither a map nor a list.
	problems = validate(t, "name: api\nenv_vars: PORT=8080\n")
	require.Equal(t, []string{"env_vars: got string, want object or array"}, problemStrings(problems))
}

func TestValidateEnvironmentBlocks(t *testing.T) {
	t.Parallel()

	problems := validate(t, "name: api\nenvironments:\n  prod:\n    cloud_run:\n      max_instance: 5\n")
	require.Len(t, problems, 1)
	require.Equal(t, "environments.prod.cloud_run.max_instance", problems[0].Path)
	require.Contains(t, problems[0].Message, "max_instances")
}

func TestTopLevelKeys(t *testing.T) {
	t.Parallel()

	keys, err := schemas.TopLevelKeys()
	require.NoError(t, err)
	require.Contains(t, keys, "cloud_run")
	require.Contains(t, keys, "health_check")
}

func TestSuggestKey(t *testing.T) {
	t.Parallel()

	known := []string{"cpu", "cpu_throttling", "secrets", "max_instances"}
	require.Equal(t, "did you mean 'cpu_throttling'?", schemas.SuggestKey("cpu_throtling", known))
	require.Equal(t, "did you mean 'secrets'?", schemas.SuggestKey("secret", known))
	require.Empty(t, schemas.SuggestKey("type", known))
}

func TestValidateCustomRecipe(t *testing.T) {
	t.Parallel()

	// A custom recipe names its own provider and type.
	problems := validate(t, `
name: api
provider: fly
type: fly-machines
`)
	require.Empty(t, problems)
}
