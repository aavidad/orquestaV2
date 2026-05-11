package orquestamcp

import "strings"

func MCPSharedContractResourceByNameV0(name string) (MCPSharedContractCompactV0, bool) {
	needle := normalizeContractLookupMCPV0(name)
	for _, source := range mcpSharedContractSourcesV0() {
		resource := toMCPSharedContractCompactV0(source)
		if needle == normalizeContractLookupMCPV0(resource.Contract) ||
			needle == normalizeContractLookupMCPV0(source.Name) ||
			needle == normalizeContractLookupMCPV0(source.Slug) ||
			needle == normalizeContractLookupMCPV0(resource.ResourceURI) {
			return resource, true
		}
	}
	return MCPSharedContractCompactV0{}, false
}

func normalizeContractLookupMCPV0(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "orquesta://contracts/")
	normalized = strings.TrimSuffix(normalized, "/v0")
	normalized = strings.TrimSuffix(normalized, " v0")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	return normalized
}
