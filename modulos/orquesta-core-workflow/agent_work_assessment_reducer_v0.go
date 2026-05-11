package orquestacoreworkflow

import "strings"

func validateAgentWorkAssessedPayloadV0(event OrchestrationEventV0) error {
	var payload AgentWorkAssessedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateAgentWorkAssessedPayloadDataV0(normalizeAgentWorkAssessedPayloadV0(payload))
}

func applyAgentWorkAssessedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload AgentWorkAssessedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeAgentWorkAssessedPayloadV0(payload)
	if err := ensureAgentWorkAssessedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.AssessmentRef); err != nil {
		return current, err
	}
	next := cloneRunForReducerV0(current)
	next.AgentAssessments = appendUniqueCompactRefV0(next.AgentAssessments, AgentAssessmentProjectionRefV0(payload))
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.AssessmentRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func ensureAgentWorkAssessedEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, payload AgentWorkAssessedPayloadV0) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if !runPhaseIsCurrentV0(current, OrchestrationPhaseIDV0(payload.PhaseID)) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRequestID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
	}
	if payload.DeliveryRef != "" && !deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.delivery_ref")
	}
	return nil
}
