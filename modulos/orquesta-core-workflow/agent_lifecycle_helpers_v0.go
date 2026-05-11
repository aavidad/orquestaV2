package orquestacoreworkflow

import "strings"

func normalizeRegisterAgentStartedPayloadV0(payload RegisterAgentStartedCommandPayloadV0) RegisterAgentStartedCommandPayloadV0 {
	return RegisterAgentStartedCommandPayloadV0{
		AgentRequestID: strings.TrimSpace(payload.AgentRequestID),
		LaunchRef:      strings.TrimSpace(payload.LaunchRef),
		AckRef:         strings.TrimSpace(payload.AckRef),
		ReadinessRef:   strings.TrimSpace(payload.ReadinessRef),
		EvidenceRefs:   compactStringsV0(payload.EvidenceRefs),
	}
}

func normalizeAgentStartedPayloadV0(payload AgentStartedPayloadV0) AgentStartedPayloadV0 {
	normalized := normalizeRegisterAgentStartedPayloadV0(RegisterAgentStartedCommandPayloadV0(payload))
	return AgentStartedPayloadV0(normalized)
}

func normalizeRegisterAgentFailedPayloadV0(payload RegisterAgentFailedCommandPayloadV0) RegisterAgentFailedCommandPayloadV0 {
	return RegisterAgentFailedCommandPayloadV0{
		AgentRequestID: strings.TrimSpace(payload.AgentRequestID),
		LaunchRef:      strings.TrimSpace(payload.LaunchRef),
		ReasonCode:     strings.TrimSpace(payload.ReasonCode),
		Retryable:      payload.Retryable,
		EvidenceRefs:   compactStringsV0(payload.EvidenceRefs),
	}
}

func normalizeAgentFailedPayloadV0(payload AgentFailedPayloadV0) AgentFailedPayloadV0 {
	normalized := normalizeRegisterAgentFailedPayloadV0(RegisterAgentFailedCommandPayloadV0(payload))
	return AgentFailedPayloadV0(normalized)
}

func agentStartedPayloadFromCommandV0(payload RegisterAgentStartedCommandPayloadV0) AgentStartedPayloadV0 {
	return AgentStartedPayloadV0(payload)
}

func agentFailedPayloadFromCommandV0(payload RegisterAgentFailedCommandPayloadV0) AgentFailedPayloadV0 {
	return AgentFailedPayloadV0(payload)
}

func agentStartedAlreadyReflectedV0(current OrchestrationRunV0, agentRequestID string) bool {
	return compactRefInListV0(current.StartedAgents, agentRequestID)
}

func agentFailedAlreadyReflectedV0(current OrchestrationRunV0, agentRequestID string) bool {
	return compactRefInListV0(current.FailedAgents, agentRequestID)
}

func compactRefInListV0(values []string, ref string) bool {
	compact := strings.TrimSpace(ref)
	for _, existing := range values {
		if strings.TrimSpace(existing) == compact {
			return true
		}
	}
	return false
}
