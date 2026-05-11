package orquestacoreworkflow

import "strings"

func handleRequestCapacityCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRequestCapacityCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	if capacityRequestAlreadyReflectedV0(current, payload.CapacityRequestID) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventCapacityRequestedV0, payload.CapacityRequestID, capacityRequestedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		if !capacityDecisionAlreadyReflectedV0(current, payload.CapacityRequestID) {
			return pendingRequestCapacityDecisionResultV0(current, command, payload)
		}
		return idempotentCommandResultV0(), nil
	}
	if !runPhaseIsCurrentV0(current, OrchestrationPhaseIDV0(payload.PhaseID)) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	event, err := NewCapacityRequestedEventV0(commandEventMetaV0(current, command, OrchestrationEventCapacityRequestedV0), capacityRequestedPayloadFromCommandV0(payload))
	if err != nil {
		return emptyCommandResultV0(), err
	}
	outbox, err := newRequestCapacityDecisionOutboxV0(command, event, payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Events: []OrchestrationEventV0{event}, Outbox: []OutboxMessageV0{outbox}}, nil
}

func pendingRequestCapacityDecisionResultV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload RequestCapacityCommandPayloadV0) (OrchestrationCommandResultV0, error) {
	event, err := NewCapacityRequestedEventV0(commandEventMetaV0(current, command, OrchestrationEventCapacityRequestedV0), capacityRequestedPayloadFromCommandV0(payload))
	if err != nil {
		return emptyCommandResultV0(), err
	}
	outbox, err := newRequestCapacityDecisionOutboxV0(command, event, payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	return OrchestrationCommandResultV0{Outbox: []OutboxMessageV0{outbox}}, nil
}

func capacityRequestAlreadyReflectedV0(current OrchestrationRunV0, capacityRequestID string) bool {
	capacityRequestID = strings.TrimSpace(capacityRequestID)
	for _, existing := range current.CapacityRequests {
		if strings.TrimSpace(existing) == capacityRequestID {
			return true
		}
	}
	return false
}
