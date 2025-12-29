package serviceinfo

func getString(m map[string]any, key string, def string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}

	return def
}

func getStringSlice(m map[string]any, key string) []string {
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

func mapFromAny(v any) map[string]any {
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
