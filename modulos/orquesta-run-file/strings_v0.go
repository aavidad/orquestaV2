package orquestarunfile

import "strings"

func compactRunFileStringsV0(values []string) []string {
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

func compactRunFileStringsOrEmptyV0(values []string) []string {
	out := compactRunFileStringsV0(values)
	if out == nil {
		return []string{}
	}
	return out
}

func runFileStringSetV0(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range compactRunFileStringsV0(values) {
		out[value] = struct{}{}
	}
	return out
}
