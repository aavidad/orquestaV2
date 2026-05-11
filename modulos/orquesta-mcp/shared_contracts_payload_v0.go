package orquestamcp

import "strings"

type mcpSharedContractSourceV0 struct {
	Name          string
	Slug          string
	Version       string
	Owner         string
	MCPRole       string
	SummaryKey    string
	CanonicalRefs []string
	Input         string
	Output        string
	PublicErrors  []string
	Guardrails    []string
	ProgressKey   string
}

func toMCPSharedContractCompactV0(source mcpSharedContractSourceV0) MCPSharedContractCompactV0 {
	version := firstNonEmptyMCPV0(source.Version, MCPSharedContractsResourceVersionV0)
	return MCPSharedContractCompactV0{
		Contract:      strings.TrimSpace(source.Name) + " " + version,
		Version:       version,
		ResourceURI:   contractResourceURIMCPV0(source.Slug),
		Owner:         strings.TrimSpace(source.Owner),
		MCPRole:       strings.TrimSpace(source.MCPRole),
		SummaryKey:    strings.TrimSpace(source.SummaryKey),
		CanonicalRefs: compactStringsMCPV0(source.CanonicalRefs),
		Input:         strings.TrimSpace(source.Input),
		Output:        strings.TrimSpace(source.Output),
		PublicErrors:  compactStringsMCPV0(source.PublicErrors),
		Guardrails:    compactStringsMCPV0(source.Guardrails),
		ProgressKey:   strings.TrimSpace(source.ProgressKey),
	}
}

func contractResourceURIMCPV0(slug string) string {
	trimmed := strings.Trim(strings.ToLower(strings.TrimSpace(slug)), "/")
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, "_", "-")
	trimmed = strings.ReplaceAll(trimmed, " ", "-")
	return "orquesta://contracts/" + trimmed + "/v0"
}
