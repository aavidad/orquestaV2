package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxDeliveryRegisterPayloadBytesV0 = 2048
	maxDeliveryRegisterStringV0       = 600
	maxDeliveryRegisterEvidenceRefsV0 = 20
)

type RegisterDeliveryCommandPayloadV0 struct {
	DeliveryRef  string   `json:"delivery_ref"`
	PhaseID      string   `json:"phase_id"`
	TaskID       string   `json:"task_id"`
	AgentRef     string   `json:"agent_ref"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type DeliveryRegisteredPayloadV0 struct {
	DeliveryRef  string   `json:"delivery_ref"`
	PhaseID      string   `json:"phase_id"`
	TaskID       string   `json:"task_id"`
	AgentRef     string   `json:"agent_ref"`
	Summary      string   `json:"summary"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

func NewRegisterDeliveryCommandV0(meta OrchestrationCommandMetaV0, payload RegisterDeliveryCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRegisterDeliveryV0, normalizeRegisterDeliveryPayloadV0(payload))
}

func NewDeliveryRegisteredEventV0(meta OrchestrationEventMetaV0, payload DeliveryRegisteredPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventDeliveryRegisteredV0, normalizeDeliveryRegisteredPayloadV0(payload))
}

func decodeRegisterDeliveryCommandPayloadV0(raw json.RawMessage) (RegisterDeliveryCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxDeliveryRegisterPayloadBytesV0 {
		return RegisterDeliveryCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RegisterDeliveryCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RegisterDeliveryCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRegisterDeliveryPayloadV0(payload)
	if err := validateRegisterDeliveryCommandPayloadDataV0(payload); err != nil {
		return RegisterDeliveryCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateDeliveryRegisteredPayloadV0(event OrchestrationEventV0) error {
	var payload DeliveryRegisteredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateDeliveryRegisteredPayloadDataV0(normalizeDeliveryRegisteredPayloadV0(payload))
}

func handleRegisterDeliveryCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRegisterDeliveryCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRegisterDeliveryCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	if deliveryAlreadyReflectedV0(current, payload.DeliveryRef) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventDeliveryRegisteredV0, payload.DeliveryRef, deliveryRegisteredPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewDeliveryRegisteredEventV0(commandEventMetaV0(current, command, OrchestrationEventDeliveryRegisteredV0), deliveryRegisteredPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyDeliveryRegisteredEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload DeliveryRegisteredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeDeliveryRegisteredPayloadV0(payload)
	if err := ensureDeliveryRegisteredEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.DeliveryRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Deliveries = appendUniqueCompactRefV0(next.Deliveries, payload.DeliveryRef)
	next.DeliveredTasks = appendUniqueCompactRefV0(next.DeliveredTasks, payload.TaskID)
	next.DeliveredAgents = appendUniqueCompactRefV0(next.DeliveredAgents, payload.AgentRef)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.DeliveryRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func deliveryRegisteredPayloadFromCommandV0(payload RegisterDeliveryCommandPayloadV0) DeliveryRegisteredPayloadV0 {
	return DeliveryRegisteredPayloadV0{
		DeliveryRef:  payload.DeliveryRef,
		PhaseID:      payload.PhaseID,
		TaskID:       payload.TaskID,
		AgentRef:     payload.AgentRef,
		Summary:      payload.Summary,
		EvidenceRefs: cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRegisterDeliveryPayloadV0(payload RegisterDeliveryCommandPayloadV0) RegisterDeliveryCommandPayloadV0 {
	return RegisterDeliveryCommandPayloadV0{
		DeliveryRef:  strings.TrimSpace(payload.DeliveryRef),
		PhaseID:      strings.TrimSpace(payload.PhaseID),
		TaskID:       strings.TrimSpace(payload.TaskID),
		AgentRef:     strings.TrimSpace(payload.AgentRef),
		Summary:      strings.TrimSpace(payload.Summary),
		EvidenceRefs: compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeDeliveryRegisteredPayloadV0(payload DeliveryRegisteredPayloadV0) DeliveryRegisteredPayloadV0 {
	normalized := normalizeRegisterDeliveryPayloadV0(RegisterDeliveryCommandPayloadV0(payload))
	return deliveryRegisteredPayloadFromCommandV0(normalized)
}

func deliveryAlreadyReflectedV0(current OrchestrationRunV0, deliveryRef string) bool {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, existing := range current.Deliveries {
		if strings.TrimSpace(existing) == deliveryRef {
			return true
		}
	}
	return false
}
