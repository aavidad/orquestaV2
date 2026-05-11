package orquestacoreworkflow

func handleRegisterCapacityDecisionCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRegisterCapacityDecisionCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureExistingRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if !capacityRequestAlreadyReflectedV0(current, payload.CapacityRequestID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.capacity_request_id")
	}
	matches, conflicts := capacityDecisionMatchesPayloadV0(current, payload)
	if conflicts {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.decision_ref")
	}
	if matches {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventCapacityDecidedV0, payload.CapacityRequestID, capacityDecidedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewCapacityDecidedEventV0(commandEventMetaV0(current, command, OrchestrationEventCapacityDecidedV0), capacityDecidedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}
