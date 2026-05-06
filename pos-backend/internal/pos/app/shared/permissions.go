package shared

import (
	"encoding/json"
	"sort"
)

func PermissionsFromJSON(body string) []string {
	var raw map[string]any
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	add := func(permission string) {
		if permission == "" {
			return
		}
		seen[permission] = struct{}{}
	}
	for key, value := range raw {
		if allowed, ok := value.(bool); ok && allowed {
			add(key)
		}
	}
	if values, ok := raw["permissions"].([]any); ok {
		for _, value := range values {
			if text, ok := value.(string); ok {
				add(text)
			}
		}
	}
	out := make([]string, 0, len(seen))
	for permission := range seen {
		out = append(out, permission)
	}
	sort.Strings(out)
	return out
}

func HasPermission(body, permission string) bool {
	for _, item := range PermissionsFromJSON(body) {
		if item == permission {
			return true
		}
	}
	return false
}
