package orquestaruncoordinator

import "strings"

func stringSetV0(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out[trimmed] = struct{}{}
	}
	return out
}
