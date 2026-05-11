package orquestacoreworkflow

func handleRegisterAgentStartedCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRegisterAgentStartedCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureAgentLifecycleCommandAllowedV0(current, command, payload.AgentRequestID); err != nil {
		return emptyCommandResultV0(), err
	}
	if agentStartedAlreadyReflectedV0(current, payload.AgentRequestID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventAgentStartedV0, payload.AgentRequestID, agentStartedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	if agentStartHasTerminalConflictV0(current, payload.AgentRequestID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	event, err := NewAgentStartedEventV0(commandEventMetaV0(current, command, OrchestrationEventAgentStartedV0), agentStartedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func handleRegisterAgentFailedCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRegisterAgentFailedCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureAgentLifecycleCommandAllowedV0(current, command, payload.AgentRequestID); err != nil {
		return emptyCommandResultV0(), err
	}
	if agentFailedAlreadyReflectedV0(current, payload.AgentRequestID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventAgentFailedV0, payload.AgentRequestID, agentFailedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	if agentFailureHasTerminalConflictV0(current, payload.AgentRequestID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	event, err := NewAgentFailedEventV0(commandEventMetaV0(current, command, OrchestrationEventAgentFailedV0), agentFailedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func ensureAgentLifecycleCommandAllowedV0(current OrchestrationRunV0, command OrchestrationCommandV0, agentRequestID string) error {
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return err
	}
	if !agentRequestAlreadyReflectedV0(current, agentRequestID) {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	return nil
}
