package orquestarunmemory

import "strings"

func compactStringsV0(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func stringSetV0(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range compactStringsV0(values) {
		out[value] = struct{}{}
	}
	return out
}
