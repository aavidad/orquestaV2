package orquestacoreworkflow

import "strings"

func handleRegisterAgentStopConfirmedCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRegisterAgentStopConfirmedCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if agentStopConfirmedAlreadyReflectedV0(current, payload.AgentRequestID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventAgentStopConfirmedV0, payload.AgentRequestID, agentStopConfirmedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	if !agentStopAlreadyReflectedV0(current, payload.AgentRequestID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	if agentLostAlreadyReflectedV0(current, payload.AgentRequestID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	event, err := NewAgentStopConfirmedEventV0(commandEventMetaV0(current, command, OrchestrationEventAgentStopConfirmedV0), agentStopConfirmedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyAgentStopConfirmedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload AgentStopConfirmedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeAgentStopConfirmedPayloadV0(payload)
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	if !agentStopAlreadyReflectedV0(current, payload.AgentRequestID) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
	}
	if agentLostAlreadyReflectedV0(current, payload.AgentRequestID) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.AgentRequestID); err != nil {
		return current, err
	}
	next := cloneRunForReducerV0(current)
	next.ConfirmedStoppedAgents = appendUniqueCompactRefV0(next.ConfirmedStoppedAgents, payload.AgentRequestID)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.AgentRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}
