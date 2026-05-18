package orquestacoreworkflow

func ensureRegisterAgentLostCommandAllowedV0(
	current OrchestrationRunV0,
	command OrchestrationCommandV0,
	payload RegisterAgentLostCommandPayloadV0,
) error {
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return err
	}
	return ensureAgentLostTransitionAllowedV0(current, payload.AgentRequestID, true)
}

func ensureAgentLostEventAllowedV0(
	current OrchestrationRunV0,
	event OrchestrationEventV0,
	payload AgentLostPayloadV0,
) error {
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return err
	}
	return ensureAgentLostTransitionAllowedForEventV0(current, payload.AgentRequestID)
}

func ensureAgentLostTransitionAllowedV0(
	current OrchestrationRunV0,
	agentRequestID string,
	command bool,
) error {
	if !agentRequestAlreadyReflectedV0(current, agentRequestID) ||
		!agentStartedAlreadyReflectedV0(current, agentRequestID) {
		return agentLostTransitionErrorV0(command)
	}
	if agentFailedAlreadyReflectedV0(current, agentRequestID) ||
		agentStopConfirmedAlreadyReflectedV0(current, agentRequestID) ||
		agentDeliveredAlreadyReflectedV0(current, agentRequestID) {
		return agentLostTransitionErrorV0(command)
	}
	return nil
}

func ensureAgentLostTransitionAllowedForEventV0(
	current OrchestrationRunV0,
	agentRequestID string,
) error {
	if err := ensureAgentLostTransitionAllowedV0(current, agentRequestID, false); err != nil {
		return err
	}
	return nil
}

func agentLostTransitionErrorV0(command bool) error {
	if command {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.agent_request_id")
	}
	return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.agent_request_id")
}

func agentDeliveredAlreadyReflectedV0(current OrchestrationRunV0, agentRequestID string) bool {
	return compactRefInListV0(current.DeliveredAgents, agentRequestID)
}
