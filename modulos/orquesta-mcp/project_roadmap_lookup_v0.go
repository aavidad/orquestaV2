package orquestamcp

import "strings"

func MCPProjectRoadmapItemByKeyV0(value string) (MCPProjectRoadmapItemV0, bool) {
	for _, source := range mcpProjectRoadmapSourcesV0() {
		item := toMCPProjectRoadmapItemV0(source)
		if sameProjectRoadmapLookupMCPV0(value, item.ID) ||
			sameProjectRoadmapLookupMCPV0(value, item.Area) ||
			sameProjectRoadmapLookupMCPV0(value, item.ProgressKey) {
			return item, true
		}
		for _, alias := range mcpProjectRoadmapLookupAliasesV0(item) {
			if sameProjectRoadmapLookupMCPV0(value, alias) {
				return item, true
			}
		}
		for _, contract := range item.Contracts {
			if sameProjectRoadmapLookupMCPV0(value, contract) {
				return item, true
			}
		}
	}
	return MCPProjectRoadmapItemV0{}, false
}

func MCPProjectDecisionByKeyV0(value string) (MCPProjectDecisionCompactV0, bool) {
	for _, source := range mcpProjectDecisionSourcesV0() {
		decision := toMCPProjectDecisionCompactV0(source)
		if sameProjectRoadmapLookupMCPV0(value, decision.ID) ||
			sameProjectRoadmapLookupMCPV0(value, decision.DecisionKey) {
			return decision, true
		}
		for _, contract := range decision.AppliesTo {
			if sameProjectRoadmapLookupMCPV0(value, contract) {
				return decision, true
			}
		}
	}
	return MCPProjectDecisionCompactV0{}, false
}

func mcpProjectRoadmapLookupAliasesV0(item MCPProjectRoadmapItemV0) []string {
	switch item.ID {
	case "CORE-ROADMAP-005":
		return []string{"domain_work_consumidores"}
	default:
		return nil
	}
}

func normalizeProjectRoadmapLookupMCPV0(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "orquesta://project/roadmap/")
	normalized = strings.TrimSuffix(normalized, "/v0")
	normalized = strings.TrimSuffix(normalized, " v0")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	return normalized
}

func sameProjectRoadmapLookupMCPV0(left, right string) bool {
	leftNormalized := normalizeProjectRoadmapLookupMCPV0(left)
	rightNormalized := normalizeProjectRoadmapLookupMCPV0(right)
	return leftNormalized == rightNormalized ||
		strings.ReplaceAll(leftNormalized, "-", "") == strings.ReplaceAll(rightNormalized, "-", "")
}
