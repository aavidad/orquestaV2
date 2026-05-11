package orquestacoreworkflow

import "strings"

func applyCapacityDecidedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload CapacityDecidedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeCapacityDecidedPayloadV0(payload)
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	commandPayload := RegisterCapacityDecisionCommandPayloadV0(payload)
	if !capacityRequestAlreadyReflectedV0(current, commandPayload.CapacityRequestID) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.capacity_request_id")
	}
	matches, conflicts := capacityDecisionMatchesPayloadV0(current, commandPayload)
	if conflicts {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.decision_ref")
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.CapacityRequestID); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	if !matches {
		next.CapacityDecisions = appendUniqueCompactRefV0(next.CapacityDecisions, capacityDecisionProjectionRefV0(commandPayload))
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.CapacityRequestID)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}
