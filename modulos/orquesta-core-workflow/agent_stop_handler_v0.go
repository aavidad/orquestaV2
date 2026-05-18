package orquestacoreworkflow

import "strings"

func handleStopAgentCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeStopAgentCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if agentStopAlreadyReflectedV0(current, payload.AgentRequestID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventAgentStopRequestedV0, payload.AgentRequestID, agentStopRequestedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		if agentLostAlreadyReflectedV0(current, payload.AgentRequestID) {
			return idempotentCommandResultV0(), nil
		}
		if agentStopConfirmedAlreadyReflectedV0(current, payload.AgentRequestID) {
			return idempotentCommandResultV0(), nil
		}
		return pendingStopRuntimeAgentResultV0(current, command, payload)
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRequestID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	if agentFailedAlreadyReflectedV0(current, payload.AgentRequestID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	if agentLostAlreadyReflectedV0(current, payload.AgentRequestID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	event, err := NewAgentStopRequestedEventV0(commandEventMetaV0(current, command, OrchestrationEventAgentStopRequestedV0), agentStopRequestedPayloadFromCommandV0(payload))
	if err != nil {
		return emptyCommandResultV0(), err
	}
	outbox, err := newStopRuntimeAgentOutboxV0(command, event, payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Events: []OrchestrationEventV0{event}, Outbox: []OutboxMessageV0{outbox}}, nil
}

func applyAgentStopRequestedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload AgentStopRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	if !agentRequestAlreadyReflectedV0(current, payload.AgentRequestID) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
	}
	if agentFailedAlreadyReflectedV0(current, payload.AgentRequestID) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.AgentRequestID); err != nil {
		return current, err
	}
	next := cloneRunForReducerV0(current)
	next.StoppedAgents = appendUniqueCompactRefV0(next.StoppedAgents, payload.AgentRequestID)
	next.AgentStopRequests = appendUniqueCompactRefV0(next.AgentStopRequests, AgentStopRequestProjectionRefV0(payload))
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.AgentRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func pendingStopRuntimeAgentResultV0(
	current OrchestrationRunV0,
	command OrchestrationCommandV0,
	payload StopAgentCommandPayloadV0,
) (OrchestrationCommandResultV0, error) {
	event, err := NewAgentStopRequestedEventV0(
		commandEventMetaV0(current, command, OrchestrationEventAgentStopRequestedV0),
		agentStopRequestedPayloadFromCommandV0(payload),
	)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	outbox, err := newStopRuntimeAgentOutboxV0(command, event, payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Outbox: []OutboxMessageV0{outbox}}, nil
}
