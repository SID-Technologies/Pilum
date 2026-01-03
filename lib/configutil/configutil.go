// Package configutil provides helper functions for type-safe extraction
// of values from map[string]any configurations (typically parsed from YAML).
package configutil

// GetString extracts a string value from a map, returning the default if not found or wrong type.
func GetString(m map[string]any, key string, def string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return def
}

// GetInt extracts an int value from a map, handling int, int64, and float64 types.
// Returns the default if not found or wrong type.
func GetInt(m map[string]any, key string, def int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return def
}

// GetBool extracts a bool value from a map, returning the default if not found or wrong type.
func GetBool(m map[string]any, key string, def bool) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return def
}

// GetStringSlice extracts a string slice from a map, handling both []string and []any types.
// Returns nil if not found or wrong type.
func GetStringSlice(m map[string]any, key string) []string {
	val, ok := m[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}

	return nil
}

// MapFromAny converts an any value to map[string]any.
// Handles both map[string]any and map[any]any (from yaml.v2).
// Returns an empty map if the value cannot be converted.
func MapFromAny(v any) map[string]any {
	// Handle map[string]any directly
	if m, ok := v.(map[string]any); ok {
		return m
	}

	// Handle map[interface{}]interface{} from yaml.v2
	if m, ok := v.(map[any]any); ok {
		result := make(map[string]any)
		for k, val := range m {
			if keyStr, ok := k.(string); ok {
				result[keyStr] = val
			}
		}
		return result
	}

	return map[string]any{}
}

// GetNestedString extracts a string value from a nested map structure.
// e.g., GetNestedString(config, "homebrew", "project_url") returns config["homebrew"]["project_url"]
func GetNestedString(config map[string]any, keys ...string) string {
	if len(keys) == 0 || config == nil {
		return ""
	}

	current := config
	for i, key := range keys {
		val, exists := current[key]
		if !exists {
			return ""
		}

		// If this is the last key, try to return it as a string
		if i == len(keys)-1 {
			if str, ok := val.(string); ok {
				return str
			}
			return ""
		}

		// Otherwise, navigate deeper into the map
		if nested, ok := val.(map[string]any); ok {
			current = nested
		} else {
			return ""
		}
	}

	return ""
}
