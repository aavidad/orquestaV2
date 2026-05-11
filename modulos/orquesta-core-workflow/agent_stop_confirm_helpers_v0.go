package orquestacoreworkflow

import "strings"

func normalizeRegisterAgentStopConfirmedPayloadV0(payload RegisterAgentStopConfirmedCommandPayloadV0) RegisterAgentStopConfirmedCommandPayloadV0 {
	return RegisterAgentStopConfirmedCommandPayloadV0{
		ConfirmationRef: strings.TrimSpace(payload.ConfirmationRef),
		AgentRequestID:  strings.TrimSpace(payload.AgentRequestID),
		ObservedAt:      strings.TrimSpace(payload.ObservedAt),
		Summary:         strings.TrimSpace(payload.Summary),
		EvidenceRefs:    compactStringsV0(payload.EvidenceRefs),
	}
}

func normalizeAgentStopConfirmedPayloadV0(payload AgentStopConfirmedPayloadV0) AgentStopConfirmedPayloadV0 {
	normalized := normalizeRegisterAgentStopConfirmedPayloadV0(RegisterAgentStopConfirmedCommandPayloadV0(payload))
	return agentStopConfirmedPayloadFromCommandV0(normalized)
}

func agentStopConfirmedPayloadFromCommandV0(payload RegisterAgentStopConfirmedCommandPayloadV0) AgentStopConfirmedPayloadV0 {
	return AgentStopConfirmedPayloadV0{
		ConfirmationRef: payload.ConfirmationRef,
		AgentRequestID:  payload.AgentRequestID,
		ObservedAt:      payload.ObservedAt,
		Summary:         payload.Summary,
		EvidenceRefs:    cloneStringsV0(payload.EvidenceRefs),
	}
}

func agentStopConfirmedAlreadyReflectedV0(current OrchestrationRunV0, agentRequestID string) bool {
	agentRequestID = strings.TrimSpace(agentRequestID)
	for _, existing := range current.ConfirmedStoppedAgents {
		if strings.TrimSpace(existing) == agentRequestID {
			return true
		}
	}
	return false
}

func agentStopConfirmedTextFieldsV0(payload RegisterAgentStopConfirmedCommandPayloadV0) []string {
	values := []string{payload.ConfirmationRef, payload.AgentRequestID, payload.ObservedAt, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
