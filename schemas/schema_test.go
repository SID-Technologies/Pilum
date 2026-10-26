package schemas_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/sid-technologies/pilum/recipes"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const schemaPath = "pilum.yaml.schema.json"

func TestSchemaIsValidJSON(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
}

func TestSchemaHasRequiredTopLevelKeys(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)

	require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"])
	require.NotEmpty(t, schema["$id"])
	require.NotEmpty(t, schema["title"])
	require.Equal(t, "object", schema["type"])
	require.NotNil(t, schema["properties"])
	require.NotNil(t, schema["$defs"])
	require.NotNil(t, schema["allOf"])
}

func TestSchemaRequiresNameField(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)

	required, ok := schema["required"].([]any)
	require.True(t, ok)
	require.Contains(t, required, "name")
}

func TestSchemaProviderEnum(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)

	props := schema["properties"].(map[string]any)
	provider := props["provider"].(map[string]any)
	// Examples, not an enum: custom recipes bring their own providers.
	require.NotContains(t, provider, "enum")
	examples := provider["examples"].([]any)

	expectedProviders := []string{"gcp", "aws", "azure", "cloudflare", "homebrew", "npm"}
	for _, p := range expectedProviders {
		require.Contains(t, examples, p, "provider examples should include %s", p)
	}
}

func TestSchemaHasAllProviderDefs(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)

	defs := schema["$defs"].(map[string]any)

	expectedDefs := []string{
		"build",
		"cloud_run",
		"job",
		"lambda",
		"container_app",
		"homebrew",
		"npm",
		"cloudflare",
	}

	for _, name := range expectedDefs {
		def, exists := defs[name]
		require.True(t, exists, "$defs should contain %s", name)

		defMap, ok := def.(map[string]any)
		require.True(t, ok, "$defs/%s should be an object", name)
		require.Equal(t, "object", defMap["type"], "$defs/%s should have type: object", name)
		require.NotNil(t, defMap["properties"], "$defs/%s should have properties", name)
	}
}

func TestSchemaPropertiesReferenceCorrectDefs(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	props := schema["properties"].(map[string]any)

	refChecks := map[string]string{
		"build":         "#/$defs/build",
		"cloud_run":     "#/$defs/cloud_run",
		"job":           "#/$defs/job",
		"lambda":        "#/$defs/lambda",
		"container_app": "#/$defs/container_app",
		"homebrew":      "#/$defs/homebrew",
		"npm":           "#/$defs/npm",
		"cloudflare":    "#/$defs/cloudflare",
	}

	for prop, expectedRef := range refChecks {
		propObj, exists := props[prop].(map[string]any)
		require.True(t, exists, "property %s should exist", prop)
		require.Equal(t, expectedRef, propObj["$ref"], "property %s should reference %s", prop, expectedRef)
	}
}

func TestSchemaConditionalValidation_GCP(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	allOf := schema["allOf"].([]any)

	found := findConditionalRule(allOf, "gcp")
	require.NotNil(t, found, "allOf should contain a conditional rule for gcp")

	then := found["then"].(map[string]any)
	required := then["required"].([]any)
	require.Contains(t, required, "project")
	require.Contains(t, required, "region")
	require.Contains(t, required, "registry_name")
}

func TestSchemaConditionalValidation_AWS(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	allOf := schema["allOf"].([]any)

	found := findConditionalRule(allOf, "aws")
	require.NotNil(t, found, "allOf should contain a conditional rule for aws")

	then := found["then"].(map[string]any)
	required := then["required"].([]any)
	require.Contains(t, required, "region")
}

func TestSchemaConditionalValidation_AllProviders(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	allOf := schema["allOf"].([]any)

	// Every provider in the enum should have a conditional rule
	providers := []string{"gcp", "aws", "azure", "homebrew", "npm", "cloudflare"}
	for _, p := range providers {
		found := findConditionalRule(allOf, p)
		require.NotNil(t, found, "allOf should contain a conditional rule for %s", p)
	}
}

func TestSchemaEnvironmentsBlock(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	props := schema["properties"].(map[string]any)

	envs, exists := props["environments"].(map[string]any)
	require.True(t, exists)
	require.Equal(t, "object", envs["type"])

	// Environment overrides should support all provider-specific refs
	addlProps := envs["additionalProperties"].(map[string]any)
	envProps := addlProps["properties"].(map[string]any)

	expectedRefs := []string{"build", "cloud_run", "job", "lambda", "container_app", "homebrew", "npm", "cloudflare"}
	for _, ref := range expectedRefs {
		_, exists := envProps[ref]
		require.True(t, exists, "environment overrides should support %s", ref)
	}
}

func TestSchemaBuildDefProperties(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	defs := schema["$defs"].(map[string]any)
	build := defs["build"].(map[string]any)
	props := build["properties"].(map[string]any)

	expectedFields := []string{"language", "version", "cmd", "version_var", "output", "output_dir", "env_vars", "flags", "platforms", "warm", "warm_dir"}
	for _, field := range expectedFields {
		_, exists := props[field]
		require.True(t, exists, "build def should have property %s", field)
	}
}

func TestSchemaCloudRunDefProperties(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	defs := schema["$defs"].(map[string]any)
	cloudRun := defs["cloud_run"].(map[string]any)
	props := cloudRun["properties"].(map[string]any)

	expectedFields := []string{"min_instances", "max_instances", "cpu_throttling", "memory", "cpu", "concurrency", "timeout_seconds"}
	for _, field := range expectedFields {
		_, exists := props[field]
		require.True(t, exists, "cloud_run def should have property %s", field)
	}
}

// loadSchema reads and parses the schema file.
func loadSchema(t *testing.T) map[string]any {
	t.Helper()

	data, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))

	return schema
}

// findConditionalRule finds an allOf entry whose if/properties/provider/const matches the given provider.
func findConditionalRule(allOf []any, provider string) map[string]any {
	for _, item := range allOf {
		rule, ok := item.(map[string]any)
		if !ok {
			continue
		}

		ifBlock, ok := rule["if"].(map[string]any)
		if !ok {
			continue
		}

		ifProps, ok := ifBlock["properties"].(map[string]any)
		if !ok {
			continue
		}

		providerBlock, ok := ifProps["provider"].(map[string]any)
		if !ok {
			continue
		}

		if providerBlock["const"] == provider {
			return rule
		}
	}

	return nil
}

func TestSchemaMIGContainer(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	props := schema["properties"].(map[string]any)

	typeEnum := props["type"].(map[string]any)["examples"].([]any)
	require.Contains(t, typeEnum, "gcp-mig-container")
	require.Equal(t, "#/$defs/mig", props["mig"].(map[string]any)["$ref"])

	envProps := props["environments"].(map[string]any)["additionalProperties"].(map[string]any)["properties"].(map[string]any)
	require.Contains(t, envProps, "mig")

	mig := schema["$defs"].(map[string]any)["mig"].(map[string]any)
	migProps := mig["properties"].(map[string]any)
	for _, field := range []string{
		"name", "zone", "region", "template_base", "image_metadata_key",
		"template_region", "max_surge", "max_unavailable", "wait_timeout",
		"machine_type", "network", "subnet", "external_ip", "network_tags", "service_account", "env_vars",
	} {
		require.Contains(t, migProps, field)
	}
	require.Equal(t, []any{"name", "template_base"}, mig["required"])
	require.Len(t, mig["oneOf"], 2, "exactly one of zone or region")
	require.Equal(t, "container-image", migProps["image_metadata_key"].(map[string]any)["default"])
	require.Equal(t, false, mig["additionalProperties"])

	// The type has its own required fields...
	var rule map[string]any
	for _, item := range schema["allOf"].([]any) {
		candidate := item.(map[string]any)
		ifProps := candidate["if"].(map[string]any)["properties"].(map[string]any)
		typeBlock, ok := ifProps["type"].(map[string]any)
		if ok && typeBlock["const"] == "gcp-mig-container" {
			rule = candidate
		}
	}
	require.NotNil(t, rule, "allOf should contain a rule for gcp-mig-container")
	required := rule["then"].(map[string]any)["required"].([]any)
	require.ElementsMatch(t, []any{"project", "region", "image", "template", "mig"}, required)

	// ...and is exempt from the gcp rule's registry_name: it pushes to `image`.
	gcp := findConditionalRule(schema["allOf"].([]any), "gcp")
	require.NotNil(t, gcp["if"].(map[string]any)["not"])
}

// configKeyPattern matches the keys a config parser reads, such as
// configutil.GetString(cloudRunMap, "ingress", "").
var configKeyPattern = regexp.MustCompile(`(?:configutil\.Get\w+|optionalBool)\(\w+, "([a-z0-9_]+)"`)

// Every key a provider's config parser reads must be in the schema, so the
// next new key fails here until the schema documents it. Each schema block
// sets additionalProperties: false, so a missing key is an editor error.
func TestSchemaCoversParsedConfigKeys(t *testing.T) {
	t.Parallel()

	parsers := map[string]string{
		"cloud_run":     "../ingredients/gcp/config.go",
		"job":           "../ingredients/gcp/cloud_run_job_config.go",
		"mig":           "../ingredients/gcp/mig_config.go",
		"npm":           "../ingredients/npm/config.go",
		"cloudflare":    "../ingredients/cloudflare/config.go",
		"container_app": "../ingredients/azure/config.go",
		"homebrew":      "../ingredients/homebrew/config.go",
	}

	defs := loadSchema(t)["$defs"].(map[string]any)

	for def, source := range parsers {
		src, err := os.ReadFile(filepath.Clean(source))
		require.NoError(t, err)

		matches := configKeyPattern.FindAllStringSubmatch(string(src), -1)
		require.NotEmpty(t, matches, "no config keys found in %s", source)

		props := defs[def].(map[string]any)["properties"].(map[string]any)
		for _, m := range matches {
			require.Contains(t, props, m[1], "%s reads %s.%s but the schema doesn't list it", source, def, m[1])
		}
	}
}

// Nested env_vars blocks are merged into the service's env vars
// (service_info.mergeEnvVarSources), so each target block accepts them.
func TestSchemaTargetBlocksAcceptEnvVars(t *testing.T) {
	t.Parallel()

	defs := loadSchema(t)["$defs"].(map[string]any)
	for _, def := range []string{"cloud_run", "container_app", "lambda", "mig"} {
		props := defs[def].(map[string]any)["properties"].(map[string]any)
		require.Contains(t, props, "env_vars", def)
	}
}

// Every built-in recipe shows up in editor autocomplete for type.
func TestSchemaTypeExamplesCoverRecipes(t *testing.T) {
	t.Parallel()

	props := loadSchema(t)["properties"].(map[string]any)
	examples := props["type"].(map[string]any)["examples"].([]any)

	files, err := recipes.FS.ReadDir(".")
	require.NoError(t, err)
	for _, f := range files {
		data, readErr := recipes.FS.ReadFile(f.Name())
		require.NoError(t, readErr)
		var rcp struct {
			Name string `yaml:"name"`
		}
		require.NoError(t, yaml.Unmarshal(data, &rcp))
		require.Contains(t, examples, rcp.Name, "%s: add %q to type examples", f.Name(), rcp.Name)
	}
}
