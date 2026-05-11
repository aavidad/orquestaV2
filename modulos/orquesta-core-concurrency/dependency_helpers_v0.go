package orquestacoreconcurrency

import (
	"sort"
	"strings"
)

func sortedUniqueStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	compact := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		compact = append(compact, value)
	}
	sort.Strings(compact)
	return compact
}

func sortedMapKeysV0(values map[string]bool) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key, ok := range values {
		if ok && strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
