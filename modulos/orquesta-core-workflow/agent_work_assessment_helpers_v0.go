package orquestacoreworkflow

import "strings"

func ensureAssessAgentWorkCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload AssessAgentWorkCommandPayloadV0) error {
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return err
	}
	if !runContainsPhaseV0(current, OrchestrationPhaseIDV0(payload.PhaseID)) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRequestID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	if payload.DeliveryRef != "" && !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.delivery_ref")
	}
	return nil
}

func normalizeAssessAgentWorkPayloadV0(payload AssessAgentWorkCommandPayloadV0) AssessAgentWorkCommandPayloadV0 {
	return AssessAgentWorkCommandPayloadV0{
		AssessmentRef:  strings.TrimSpace(payload.AssessmentRef),
		PhaseID:        strings.TrimSpace(payload.PhaseID),
		AgentRequestID: strings.TrimSpace(payload.AgentRequestID),
		TaskRef:        strings.TrimSpace(payload.TaskRef),
		DeliveryRef:    strings.TrimSpace(payload.DeliveryRef),
		Verdict:        normalizeAgentAssessmentVerdictV0(payload.Verdict),
		Action:         normalizeAgentAssessmentActionV0(payload.Action),
		Severity:       normalizeAgentAssessmentSeverityV0(payload.Severity),
		Summary:        strings.TrimSpace(payload.Summary),
		EvidenceRefs:   compactStringsV0(payload.EvidenceRefs),
	}
}

func normalizeAgentWorkAssessedPayloadV0(payload AgentWorkAssessedPayloadV0) AgentWorkAssessedPayloadV0 {
	normalized := normalizeAssessAgentWorkPayloadV0(AssessAgentWorkCommandPayloadV0(payload))
	return agentWorkAssessedPayloadFromCommandV0(normalized)
}

func agentWorkAssessedPayloadFromCommandV0(payload AssessAgentWorkCommandPayloadV0) AgentWorkAssessedPayloadV0 {
	return AgentWorkAssessedPayloadV0{
		AssessmentRef:  payload.AssessmentRef,
		PhaseID:        payload.PhaseID,
		AgentRequestID: payload.AgentRequestID,
		TaskRef:        payload.TaskRef,
		DeliveryRef:    payload.DeliveryRef,
		Verdict:        payload.Verdict,
		Action:         payload.Action,
		Severity:       payload.Severity,
		Summary:        payload.Summary,
		EvidenceRefs:   cloneStringsV0(payload.EvidenceRefs),
	}
}

func stopAgentPayloadFromAssessmentV0(payload AssessAgentWorkCommandPayloadV0) StopAgentCommandPayloadV0 {
	return StopAgentCommandPayloadV0{
		AgentRequestID: payload.AgentRequestID,
		ReasonCode:     payload.Verdict,
		Summary:        payload.Summary,
		EvidenceRefs:   cloneStringsV0(payload.EvidenceRefs),
	}
}

func agentAssessmentAlreadyReflectedV0(current OrchestrationRunV0, assessmentRef string) bool {
	assessmentRef = strings.TrimSpace(assessmentRef)
	for _, existing := range current.AgentAssessments {
		if AgentAssessmentProjectionIDV0(existing) == assessmentRef {
			return true
		}
	}
	return false
}
