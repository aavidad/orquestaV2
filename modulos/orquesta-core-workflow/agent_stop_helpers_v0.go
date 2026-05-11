package orquestacoreworkflow

import "strings"

func normalizeStopAgentPayloadV0(payload StopAgentCommandPayloadV0) StopAgentCommandPayloadV0 {
	return StopAgentCommandPayloadV0{
		AgentRequestID: strings.TrimSpace(payload.AgentRequestID),
		ReasonCode:     strings.TrimSpace(payload.ReasonCode),
		Summary:        strings.TrimSpace(payload.Summary),
		EvidenceRefs:   compactStringsV0(payload.EvidenceRefs),
	}
}

func normalizeAgentStopRequestedPayloadV0(payload AgentStopRequestedPayloadV0) AgentStopRequestedPayloadV0 {
	normalized := normalizeStopAgentPayloadV0(StopAgentCommandPayloadV0(payload))
	return agentStopRequestedPayloadFromCommandV0(normalized)
}

func normalizeStopRuntimeAgentPayloadV0(payload StopRuntimeAgentRequestV0) StopRuntimeAgentRequestV0 {
	return StopRuntimeAgentRequestV0{
		AgentRequestID: strings.TrimSpace(payload.AgentRequestID),
		RunID:          strings.TrimSpace(payload.RunID),
		ReasonCode:     strings.TrimSpace(payload.ReasonCode),
		Summary:        strings.TrimSpace(payload.Summary),
		EvidenceRefs:   compactStringsV0(payload.EvidenceRefs),
	}
}

func agentStopRequestedPayloadFromCommandV0(payload StopAgentCommandPayloadV0) AgentStopRequestedPayloadV0 {
	return AgentStopRequestedPayloadV0{
		AgentRequestID: payload.AgentRequestID,
		ReasonCode:     payload.ReasonCode,
		Summary:        payload.Summary,
		EvidenceRefs:   cloneStringsV0(payload.EvidenceRefs),
	}
}

func stopRuntimeAgentPayloadFromCommandV0(command OrchestrationCommandV0, payload StopAgentCommandPayloadV0) StopRuntimeAgentRequestV0 {
	return StopRuntimeAgentRequestV0{
		AgentRequestID: payload.AgentRequestID,
		RunID:          strings.TrimSpace(command.RunID),
		ReasonCode:     payload.ReasonCode,
		Summary:        payload.Summary,
		EvidenceRefs:   cloneStringsV0(payload.EvidenceRefs),
	}
}

func agentStopAlreadyReflectedV0(current OrchestrationRunV0, agentRequestID string) bool {
	agentRequestID = strings.TrimSpace(agentRequestID)
	for _, existing := range current.StoppedAgents {
		if strings.TrimSpace(existing) == agentRequestID {
			return true
		}
	}
	return false
}
