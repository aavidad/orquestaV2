package orquestafactory

import "strings"

func compactUniqueV0(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func emptyStringsV0(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func emptyConnectorsV0(values []ConnectorSpecV0) []ConnectorSpecV0 {
	if values == nil {
		return []ConnectorSpecV0{}
	}
	return values
}
