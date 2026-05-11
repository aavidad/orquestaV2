package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxFinalValidationPayloadBytesV0 = 2048
	maxFinalValidationStringV0       = 600
	maxFinalValidationEvidenceRefsV0 = 20
)

type RegisterFinalValidationCommandPayloadV0 struct {
	ValidationRef string   `json:"validation_ref"`
	PhaseID       string   `json:"phase_id"`
	ClosedTaskRef string   `json:"closed_task_ref"`
	Summary       string   `json:"summary"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type FinalValidationRegisteredPayloadV0 struct {
	ValidationRef string   `json:"validation_ref"`
	PhaseID       string   `json:"phase_id"`
	ClosedTaskRef string   `json:"closed_task_ref"`
	Summary       string   `json:"summary"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

func NewRegisterFinalValidationCommandV0(meta OrchestrationCommandMetaV0, payload RegisterFinalValidationCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRegisterFinalValidationV0, normalizeRegisterFinalValidationPayloadV0(payload))
}

func NewFinalValidationRegisteredEventV0(meta OrchestrationEventMetaV0, payload FinalValidationRegisteredPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventFinalValidationRegisteredV0, normalizeFinalValidationRegisteredPayloadV0(payload))
}

func decodeRegisterFinalValidationCommandPayloadV0(raw json.RawMessage) (RegisterFinalValidationCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxFinalValidationPayloadBytesV0 {
		return RegisterFinalValidationCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RegisterFinalValidationCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RegisterFinalValidationCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRegisterFinalValidationPayloadV0(payload)
	if err := validateRegisterFinalValidationCommandPayloadDataV0(payload); err != nil {
		return RegisterFinalValidationCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateFinalValidationRegisteredPayloadV0(event OrchestrationEventV0) error {
	var payload FinalValidationRegisteredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateFinalValidationRegisteredPayloadDataV0(normalizeFinalValidationRegisteredPayloadV0(payload))
}

func handleRegisterFinalValidationCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeRegisterFinalValidationCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if err := ensureRegisterFinalValidationCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	if finalValidationAlreadyReflectedV0(current, payload.ValidationRef) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventFinalValidationRegisteredV0, payload.ValidationRef, finalValidationRegisteredPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	event, err := NewFinalValidationRegisteredEventV0(commandEventMetaV0(current, command, OrchestrationEventFinalValidationRegisteredV0), finalValidationRegisteredPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyFinalValidationRegisteredEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload FinalValidationRegisteredPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeFinalValidationRegisteredPayloadV0(payload)
	if err := ensureFinalValidationRegisteredEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.ValidationRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Validations = appendUniqueCompactRefV0(next.Validations, payload.ValidationRef)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.ValidationRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func finalValidationRegisteredPayloadFromCommandV0(payload RegisterFinalValidationCommandPayloadV0) FinalValidationRegisteredPayloadV0 {
	return FinalValidationRegisteredPayloadV0{
		ValidationRef: payload.ValidationRef,
		PhaseID:       payload.PhaseID,
		ClosedTaskRef: payload.ClosedTaskRef,
		Summary:       payload.Summary,
		EvidenceRefs:  cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRegisterFinalValidationPayloadV0(payload RegisterFinalValidationCommandPayloadV0) RegisterFinalValidationCommandPayloadV0 {
	return RegisterFinalValidationCommandPayloadV0{
		ValidationRef: strings.TrimSpace(payload.ValidationRef),
		PhaseID:       strings.TrimSpace(payload.PhaseID),
		ClosedTaskRef: strings.TrimSpace(payload.ClosedTaskRef),
		Summary:       strings.TrimSpace(payload.Summary),
		EvidenceRefs:  compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeFinalValidationRegisteredPayloadV0(payload FinalValidationRegisteredPayloadV0) FinalValidationRegisteredPayloadV0 {
	normalized := normalizeRegisterFinalValidationPayloadV0(RegisterFinalValidationCommandPayloadV0(payload))
	return finalValidationRegisteredPayloadFromCommandV0(normalized)
}

func finalValidationAlreadyReflectedV0(current OrchestrationRunV0, validationRef string) bool {
	validationRef = strings.TrimSpace(validationRef)
	for _, existing := range current.Validations {
		if strings.TrimSpace(existing) == validationRef {
			return true
		}
	}
	return false
}
