package orquestacoreworkflow

import "strings"

func normalizeRequestAgentPayloadV0(payload RequestAgentCommandPayloadV0) RequestAgentCommandPayloadV0 {
	return RequestAgentCommandPayloadV0{
		AgentRequestID:     strings.TrimSpace(payload.AgentRequestID),
		PhaseID:            strings.TrimSpace(payload.PhaseID),
		TaskRef:            strings.TrimSpace(payload.TaskRef),
		CapacityRequestRef: strings.TrimSpace(payload.CapacityRequestRef),
		Role:               strings.TrimSpace(payload.Role),
		Summary:            strings.TrimSpace(payload.Summary),
		EvidenceRefs:       compactStringsV0(payload.EvidenceRefs),
		SkillRefs:          compactUniqueStringsV0(payload.SkillRefs),
	}
}

func normalizeAgentRequestedPayloadV0(payload AgentRequestedPayloadV0) AgentRequestedPayloadV0 {
	normalized := normalizeRequestAgentPayloadV0(RequestAgentCommandPayloadV0(payload))
	return agentRequestedPayloadFromCommandV0(normalized)
}

func agentRequestTextFieldsV0(payload RequestAgentCommandPayloadV0) []string {
	values := []string{payload.AgentRequestID, payload.PhaseID, payload.TaskRef, payload.CapacityRequestRef, payload.Role, payload.Summary}
	values = append(values, payload.EvidenceRefs...)
	return append(values, payload.SkillRefs...)
}
