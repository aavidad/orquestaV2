package orquestaappcodexstack

import "strings"

func codexStackOperationalClosureSetV0(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range codexStackOperationalClosureCompactRefsV0(values) {
		out[value] = true
	}
	return out
}

func codexStackOperationalClosureContainsV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func codexStackOperationalClosureCompactRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func codexStackOperationalClosureSafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
