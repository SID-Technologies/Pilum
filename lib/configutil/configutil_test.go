package configutil_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/configutil"

	"github.com/stretchr/testify/require"
)

func TestGetString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		m        map[string]any
		key      string
		def      string
		expected string
	}{
		{
			name:     "key exists with string value",
			m:        map[string]any{"name": "test"},
			key:      "name",
			def:      "default",
			expected: "test",
		},
		{
			name:     "key does not exist",
			m:        map[string]any{"other": "value"},
			key:      "name",
			def:      "default",
			expected: "default",
		},
		{
			name:     "key exists but wrong type",
			m:        map[string]any{"name": 123},
			key:      "name",
			def:      "default",
			expected: "default",
		},
		{
			name:     "empty map",
			m:        map[string]any{},
			key:      "name",
			def:      "default",
			expected: "default",
		},
		{
			name:     "nil map",
			m:        nil,
			key:      "name",
			def:      "default",
			expected: "default",
		},
		{
			name:     "empty string value",
			m:        map[string]any{"name": ""},
			key:      "name",
			def:      "default",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := configutil.GetString(tt.m, tt.key, tt.def)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		m        map[string]any
		key      string
		def      int
		expected int
	}{
		{
			name:     "key exists with int value",
			m:        map[string]any{"count": 42},
			key:      "count",
			def:      0,
			expected: 42,
		},
		{
			name:     "key exists with int64 value",
			m:        map[string]any{"count": int64(100)},
			key:      "count",
			def:      0,
			expected: 100,
		},
		{
			name:     "key exists with float64 value",
			m:        map[string]any{"count": float64(55.9)},
			key:      "count",
			def:      0,
			expected: 55,
		},
		{
			name:     "key does not exist",
			m:        map[string]any{"other": 10},
			key:      "count",
			def:      -1,
			expected: -1,
		},
		{
			name:     "key exists but wrong type",
			m:        map[string]any{"count": "not a number"},
			key:      "count",
			def:      -1,
			expected: -1,
		},
		{
			name:     "zero value",
			m:        map[string]any{"count": 0},
			key:      "count",
			def:      -1,
			expected: 0,
		},
		{
			name:     "negative value",
			m:        map[string]any{"count": -5},
			key:      "count",
			def:      0,
			expected: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := configutil.GetInt(tt.m, tt.key, tt.def)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		m        map[string]any
		key      string
		def      bool
		expected bool
	}{
		{
			name:     "key exists with true",
			m:        map[string]any{"enabled": true},
			key:      "enabled",
			def:      false,
			expected: true,
		},
		{
			name:     "key exists with false",
			m:        map[string]any{"enabled": false},
			key:      "enabled",
			def:      true,
			expected: false,
		},
		{
			name:     "key does not exist returns default true",
			m:        map[string]any{"other": true},
			key:      "enabled",
			def:      true,
			expected: true,
		},
		{
			name:     "key does not exist returns default false",
			m:        map[string]any{"other": true},
			key:      "enabled",
			def:      false,
			expected: false,
		},
		{
			name:     "key exists but wrong type",
			m:        map[string]any{"enabled": "true"},
			key:      "enabled",
			def:      false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := configutil.GetBool(tt.m, tt.key, tt.def)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected []string
	}{
		{
			name:     "key exists with []string",
			m:        map[string]any{"tags": []string{"a", "b", "c"}},
			key:      "tags",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "key exists with []any containing strings",
			m:        map[string]any{"tags": []any{"x", "y", "z"}},
			key:      "tags",
			expected: []string{"x", "y", "z"},
		},
		{
			name:     "key exists with []any containing mixed types",
			m:        map[string]any{"tags": []any{"a", 123, "b"}},
			key:      "tags",
			expected: []string{"a", "b"},
		},
		{
			name:     "key does not exist",
			m:        map[string]any{"other": []string{"a"}},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "key exists but wrong type",
			m:        map[string]any{"tags": "not a slice"},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "empty slice",
			m:        map[string]any{"tags": []string{}},
			key:      "tags",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := configutil.GetStringSlice(tt.m, tt.key)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMapFromAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected map[string]any
	}{
		{
			name:     "map[string]any input",
			input:    map[string]any{"key": "value"},
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "map[any]any input",
			input:    map[any]any{"key": "value", "num": 42},
			expected: map[string]any{"key": "value", "num": 42},
		},
		{
			name:     "map[any]any with non-string keys filtered out",
			input:    map[any]any{"key": "value", 123: "ignored"},
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: map[string]any{},
		},
		{
			name:     "wrong type input",
			input:    "not a map",
			expected: map[string]any{},
		},
		{
			name:     "empty map[string]any",
			input:    map[string]any{},
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := configutil.MapFromAny(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDeepMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dst      map[string]any
		src      map[string]any
		expected map[string]any
	}{
		{
			name:     "scalar override",
			dst:      map[string]any{"project": "acme-dev", "region": "us-central1"},
			src:      map[string]any{"project": "acme-prod"},
			expected: map[string]any{"project": "acme-prod", "region": "us-central1"},
		},
		{
			name: "deep merge nested maps",
			dst: map[string]any{
				"cloud_run": map[string]any{
					"min_instances": 0,
					"max_instances": 10,
				},
			},
			src: map[string]any{
				"cloud_run": map[string]any{
					"min_instances": 2,
				},
			},
			expected: map[string]any{
				"cloud_run": map[string]any{
					"min_instances": 2,
					"max_instances": 10,
				},
			},
		},
		{
			name: "slice replacement",
			dst: map[string]any{
				"depends_on": []any{"svc-a", "svc-b"},
			},
			src: map[string]any{
				"depends_on": []any{"svc-c"},
			},
			expected: map[string]any{
				"depends_on": []any{"svc-c"},
			},
		},
		{
			name:     "new key added",
			dst:      map[string]any{"name": "api"},
			src:      map[string]any{"extra": "value"},
			expected: map[string]any{"name": "api", "extra": "value"},
		},
		{
			name:     "empty src returns dst copy",
			dst:      map[string]any{"name": "api"},
			src:      map[string]any{},
			expected: map[string]any{"name": "api"},
		},
		{
			name:     "empty dst returns src copy",
			dst:      map[string]any{},
			src:      map[string]any{"name": "api"},
			expected: map[string]any{"name": "api"},
		},
		{
			name: "yaml.v2 map[any]any in nested src",
			dst: map[string]any{
				"cloud_run": map[string]any{"min_instances": 0},
			},
			src: map[string]any{
				"cloud_run": map[any]any{"min_instances": 5},
			},
			expected: map[string]any{
				"cloud_run": map[string]any{"min_instances": 5},
			},
		},
		{
			name: "regions array replaced entirely",
			dst: map[string]any{
				"regions": []any{"us-central1", "us-east1"},
			},
			src: map[string]any{
				"regions": []any{"eu-west1"},
			},
			expected: map[string]any{
				"regions": []any{"eu-west1"},
			},
		},
		{
			name: "multi-level nesting",
			dst: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"a": "original",
						"b": "kept",
					},
				},
			},
			src: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"a": "overridden",
					},
				},
			},
			expected: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"a": "overridden",
						"b": "kept",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := configutil.DeepMerge(tt.dst, tt.src)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDeepMergeDoesNotMutateDst(t *testing.T) {
	t.Parallel()

	dst := map[string]any{"project": "dev", "region": "us-central1"}
	src := map[string]any{"project": "prod"}

	result := configutil.DeepMerge(dst, src)

	require.Equal(t, "prod", result["project"])
	require.Equal(t, "dev", dst["project"]) // original unchanged
}

func TestGetNestedString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   map[string]any
		keys     []string
		expected string
	}{
		{
			name: "single level",
			config: map[string]any{
				"name": "test",
			},
			keys:     []string{"name"},
			expected: "test",
		},
		{
			name: "two levels deep",
			config: map[string]any{
				"homebrew": map[string]any{
					"project_url": "https://github.com/org/project",
				},
			},
			keys:     []string{"homebrew", "project_url"},
			expected: "https://github.com/org/project",
		},
		{
			name: "three levels deep",
			config: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": "deep value",
					},
				},
			},
			keys:     []string{"level1", "level2", "level3"},
			expected: "deep value",
		},
		{
			name: "key does not exist at first level",
			config: map[string]any{
				"other": "value",
			},
			keys:     []string{"homebrew", "project_url"},
			expected: "",
		},
		{
			name: "key does not exist at nested level",
			config: map[string]any{
				"homebrew": map[string]any{
					"tap_url": "https://github.com/org/tap",
				},
			},
			keys:     []string{"homebrew", "project_url"},
			expected: "",
		},
		{
			name: "intermediate value is not a map",
			config: map[string]any{
				"homebrew": "not a map",
			},
			keys:     []string{"homebrew", "project_url"},
			expected: "",
		},
		{
			name: "final value is not a string",
			config: map[string]any{
				"settings": map[string]any{
					"count": 42,
				},
			},
			keys:     []string{"settings", "count"},
			expected: "",
		},
		{
			name:     "nil config",
			config:   nil,
			keys:     []string{"any", "key"},
			expected: "",
		},
		{
			name: "empty keys",
			config: map[string]any{
				"name": "test",
			},
			keys:     []string{},
			expected: "",
		},
		{
			name: "yaml.v2 nested map (map[interface{}]interface{})",
			config: map[string]any{
				"cloudflare": map[any]any{
					"account_id": "abc123",
					"token_env":  "CF_TOKEN",
				},
			},
			keys:     []string{"cloudflare", "account_id"},
			expected: "abc123",
		},
		{
			name: "yaml.v2 deeply nested map",
			config: map[string]any{
				"level1": map[any]any{
					"level2": map[any]any{
						"value": "deep",
					},
				},
			},
			keys:     []string{"level1", "level2", "value"},
			expected: "deep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := configutil.GetNestedString(tt.config, tt.keys...)
			require.Equal(t, tt.expected, result)
		})
	}
}
