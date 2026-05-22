package orquestadirectoroperativo

import "strings"

func compactStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func clampPositiveV0(value int, fallback int, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func clampOptionalPositiveMaxV0(value int, max int) int {
	if value <= 0 {
		return 0
	}
	if value > max {
		return max
	}
	return value
}
