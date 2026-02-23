package schemas_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
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
	enum := provider["enum"].([]any)

	expectedProviders := []string{"gcp", "aws", "azure", "cloudflare", "homebrew", "npm"}
	for _, p := range expectedProviders {
		require.Contains(t, enum, p, "provider enum should include %s", p)
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

	expectedFields := []string{"language", "version", "cmd", "version_var", "output", "output_dir", "env_vars", "flags", "platforms"}
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
