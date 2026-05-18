package orquestacoreworkflow

import "strings"

func handleRequestAgentCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRequestAgentCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if agentRequestAlreadyReflectedV0(current, payload.AgentRequestID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventAgentRequestedV0, payload.AgentRequestID, agentRequestedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		if !capacityDecisionAlreadyReflectedV0(current, payload.CapacityRequestRef) {
			return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.capacity_request_ref")
		}
		if agentLaunchTerminalAlreadyReflectedV0(current, payload.AgentRequestID) {
			return idempotentCommandResultV0(), nil
		}
		return pendingLaunchRuntimeAgentResultV0(current, command, payload)
	}
	if !runPhaseIsCurrentV0(current, OrchestrationPhaseIDV0(payload.PhaseID)) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	if !capacityDecisionAlreadyReflectedV0(current, payload.CapacityRequestRef) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.capacity_request_ref")
	}
	event, err := NewAgentRequestedEventV0(commandEventMetaV0(current, command, OrchestrationEventAgentRequestedV0), agentRequestedPayloadFromCommandV0(payload))
	if err != nil {
		return emptyCommandResultV0(), err
	}
	outbox, err := newLaunchRuntimeAgentOutboxV0(command, event, payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Events: []OrchestrationEventV0{event}, Outbox: []OutboxMessageV0{outbox}}, nil
}

func pendingLaunchRuntimeAgentResultV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RequestAgentCommandPayloadV0) (OrchestrationCommandResultV0, error) {
	event, err := NewAgentRequestedEventV0(commandEventMetaV0(current, command, OrchestrationEventAgentRequestedV0), agentRequestedPayloadFromCommandV0(payload))
	if err != nil {
		return emptyCommandResultV0(), err
	}
	outbox, err := newLaunchRuntimeAgentOutboxV0(command, event, payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Outbox: []OutboxMessageV0{outbox}}, nil
}

func agentLaunchTerminalAlreadyReflectedV0(current OrchestrationRunV0, agentRequestID string) bool {
	return agentStartedAlreadyReflectedV0(current, agentRequestID) ||
		agentFailedAlreadyReflectedV0(current, agentRequestID) ||
		agentLostAlreadyReflectedV0(current, agentRequestID) ||
		agentStopAlreadyReflectedV0(current, agentRequestID)
}

func agentRequestAlreadyReflectedV0(current OrchestrationRunV0, agentRequestID string) bool {
	agentRequestID = strings.TrimSpace(agentRequestID)
	for _, existing := range current.Agents {
		if strings.TrimSpace(existing) == agentRequestID {
			return true
		}
	}
	return false
}
