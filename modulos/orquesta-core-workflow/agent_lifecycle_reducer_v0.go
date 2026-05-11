package orquestacoreworkflow

import "strings"

func applyAgentStartedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload AgentStartedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeAgentStartedPayloadV0(payload)
	if err := ensureAgentLifecycleEventAllowedV0(current, event, payload.AgentRequestID); err != nil {
		return current, err
	}
	if agentStartHasTerminalConflictV0(current, payload.AgentRequestID) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.AgentRequestID); err != nil {
		return current, err
	}
	next := cloneRunForReducerV0(current)
	next.StartedAgents = appendUniqueCompactRefV0(next.StartedAgents, payload.AgentRequestID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.AgentRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func applyAgentFailedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload AgentFailedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeAgentFailedPayloadV0(payload)
	if err := ensureAgentLifecycleEventAllowedV0(current, event, payload.AgentRequestID); err != nil {
		return current, err
	}
	if agentFailureHasTerminalConflictV0(current, payload.AgentRequestID) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.AgentRequestID); err != nil {
		return current, err
	}
	next := cloneRunForReducerV0(current)
	next.FailedAgents = appendUniqueCompactRefV0(next.FailedAgents, payload.AgentRequestID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.AgentRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func ensureAgentLifecycleEventAllowedV0(current OrchestrationRunV0, event OrchestrationEventV0, agentRequestID string) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	if !agentRequestAlreadyReflectedV0(current, agentRequestID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
	}
	return nil
}
