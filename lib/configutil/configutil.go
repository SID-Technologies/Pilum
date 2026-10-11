// Package configutil provides helper functions for type-safe extraction
// of values from map[string]any configurations (typically parsed from YAML).
package configutil

import (
	"fmt"
	"strconv"
)

// GetString extracts a string value from a map, returning the default if not
// found or wrong type. A number is read as its string form, so `cpu: 1`
// means the same as `cpu: "1"`.
func GetString(m map[string]any, key string, def string) string {
	val, ok := m[key]
	if !ok {
		return def
	}

	// A bool is not a string value; `name: true` keeps the default.
	if _, isBool := val.(bool); isBool {
		return def
	}

	s, ok := scalarString(val)
	if !ok {
		return def
	}

	return s
}

// GetInt extracts an int value from a map, handling int, int64, and float64 types.
// Returns the default if not found or wrong type.
func GetInt(m map[string]any, key string, def int) int {
	val, ok := m[key]
	if !ok {
		return def
	}

	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}

	return def
}

// GetBool extracts a bool value from a map, returning the default if not found or wrong type.
func GetBool(m map[string]any, key string, def bool) bool {
	val, ok := m[key]
	if !ok {
		return def
	}

	b, ok := val.(bool)
	if !ok {
		return def
	}

	return b
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
			s, ok := item.(string)
			if ok {
				result = append(result, s)
			}
		}
		return result
	}

	return nil
}

// MapFromAny converts an any value to map[string]any.
// Returns an empty map if the value is not a map[string]any.
func MapFromAny(v any) map[string]any {
	m, ok := v.(map[string]any)
	if ok {
		return m
	}

	return map[string]any{}
}

// StringMap reads a KEY: value block such as env_vars or secrets. It accepts
// a map, or a list of {name, value} entries, and turns scalar values
// (numbers, booleans) into strings. Anything it can't read is reported in
// problems instead of being dropped, because a dropped block reads as "no
// values" and deploy targets then clear what is live.
func StringMap(v any) (map[string]string, []string) {
	result := map[string]string{}

	switch block := v.(type) {
	case nil:
		return result, nil
	case map[string]any:
		var problems []string
		for k, raw := range block {
			s, ok := scalarString(raw)
			if !ok {
				problems = append(problems, fmt.Sprintf("%s must be a string, number or boolean", k))
				continue
			}

			result[k] = s
		}

		return result, problems
	case []any:
		return stringMapFromList(block)
	}

	return result, []string{"must be a map of KEY: value or a list of {name, value}"}
}

func stringMapFromList(list []any) (map[string]string, []string) {
	result := map[string]string{}
	var problems []string

	for i, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			problems = append(problems, fmt.Sprintf("entry %d must be {name, value}", i+1))
			continue
		}

		name, ok := entry["name"].(string)
		if !ok || name == "" {
			problems = append(problems, fmt.Sprintf("entry %d is missing name", i+1))
			continue
		}

		s, ok := scalarString(entry["value"])
		if !ok {
			problems = append(problems, fmt.Sprintf("%s must have a string, number or boolean value", name))
			continue
		}

		result[name] = s
	}

	return result, problems
}

// scalarString formats a YAML scalar. A missing value (nil) is not a scalar.
func scalarString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case bool:
		return strconv.FormatBool(val), true
	case int:
		return strconv.Itoa(val), true
	case int64:
		return strconv.FormatInt(val, 10), true
	case uint64:
		return strconv.FormatUint(val, 10), true
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), true
	}

	return "", false
}

// DeepMerge merges src into dst recursively, returning a new map.
// Scalars: src overrides dst. Maps: recursively merged. Slices: src replaces dst.
// Keys in src not in dst are added. Keys in dst not in src are preserved.
func DeepMerge(dst, src map[string]any) map[string]any {
	result := make(map[string]any, len(dst))

	for k, v := range dst {
		result[k] = v
	}

	for k, srcVal := range src {
		dstVal, exists := result[k]
		if !exists {
			result[k] = srcVal
			continue
		}

		if isMap(srcVal) && isMap(dstVal) {
			result[k] = DeepMerge(MapFromAny(dstVal), MapFromAny(srcVal))
		} else {
			result[k] = srcVal
		}
	}

	return result
}

// isMap returns true if the value is a map[string]any.
func isMap(v any) bool {
	_, ok := v.(map[string]any)
	return ok
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
			str, ok := val.(string)
			if ok {
				return str
			}
			return ""
		}

		// Otherwise, navigate deeper into the map
		nested := MapFromAny(val)
		if len(nested) == 0 {
			return ""
		}
		current = nested
	}

	return ""
}
