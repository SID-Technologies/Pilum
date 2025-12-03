package serviceinfo

func getString(m map[string]any, key string, def string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}

	return def
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
