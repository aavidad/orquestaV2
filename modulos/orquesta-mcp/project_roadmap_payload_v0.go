package orquestamcp

import "strings"

type mcpProjectRoadmapItemSourceV0 struct {
	ID            string
	Area          string
	Status        string
	Owner         string
	Focus         string
	SummaryKey    string
	ProgressKey   string
	Contracts     []string
	Dependencies  []string
	Guardrails    []string
	CanonicalRefs []string
	BacklogRefs   []string
	Verification  []string
}

type mcpProjectDecisionSourceV0 struct {
	ID            string
	Status        string
	Decision      string
	DecisionKey   string
	Motivo        string
	MotivoKey     string
	AppliesTo     []string
	Guardrails    []string
	CanonicalRefs []string
}

func toMCPProjectRoadmapItemV0(source mcpProjectRoadmapItemSourceV0) MCPProjectRoadmapItemV0 {
	return MCPProjectRoadmapItemV0{
		ID:            strings.TrimSpace(source.ID),
		Area:          strings.TrimSpace(source.Area),
		Status:        firstNonEmptyMCPV0(source.Status, "pendiente"),
		Owner:         strings.TrimSpace(source.Owner),
		Focus:         strings.TrimSpace(source.Focus),
		SummaryKey:    strings.TrimSpace(source.SummaryKey),
		ProgressKey:   strings.TrimSpace(source.ProgressKey),
		Contracts:     compactStringsMCPV0(source.Contracts),
		Dependencies:  compactStringsMCPV0(source.Dependencies),
		Guardrails:    compactStringsMCPV0(source.Guardrails),
		CanonicalRefs: compactStringsMCPV0(source.CanonicalRefs),
		BacklogRefs:   compactStringsMCPV0(source.BacklogRefs),
		Verification:  compactStringsMCPV0(source.Verification),
	}
}

func toMCPProjectDecisionCompactV0(source mcpProjectDecisionSourceV0) MCPProjectDecisionCompactV0 {
	return MCPProjectDecisionCompactV0{
		ID:            strings.TrimSpace(source.ID),
		Status:        firstNonEmptyMCPV0(source.Status, "aceptada"),
		Decision:      strings.TrimSpace(source.Decision),
		DecisionKey:   strings.TrimSpace(source.DecisionKey),
		Motivo:        strings.TrimSpace(source.Motivo),
		MotivoKey:     strings.TrimSpace(source.MotivoKey),
		AppliesTo:     compactStringsMCPV0(source.AppliesTo),
		Guardrails:    compactStringsMCPV0(source.Guardrails),
		CanonicalRefs: compactStringsMCPV0(source.CanonicalRefs),
	}
}
