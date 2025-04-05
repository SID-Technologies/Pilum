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
	if m, ok := v.(map[string]any); ok {
		return m
	}

	return map[string]any{}
}
