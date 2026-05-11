package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxCloseRunPayloadBytesV0 = 2048
	maxCloseRunStringV0       = 600
	maxCloseRunEvidenceRefsV0 = 20
)

type CloseRunCommandPayloadV0 struct {
	ClosureRef    string   `json:"closure_ref"`
	PhaseID       string   `json:"phase_id"`
	ValidationRef string   `json:"validation_ref"`
	Summary       string   `json:"summary"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type RunClosedPayloadV0 struct {
	ClosureRef    string   `json:"closure_ref"`
	PhaseID       string   `json:"phase_id"`
	ValidationRef string   `json:"validation_ref"`
	Summary       string   `json:"summary"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

func NewCloseRunCommandV0(meta OrchestrationCommandMetaV0, payload CloseRunCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandCloseRunV0, normalizeCloseRunPayloadV0(payload))
}

func NewRunClosedEventV0(meta OrchestrationEventMetaV0, payload RunClosedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventRunClosedV0, normalizeRunClosedPayloadV0(payload))
}

func decodeCloseRunCommandPayloadV0(raw json.RawMessage) (CloseRunCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxCloseRunPayloadBytesV0 {
		return CloseRunCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload CloseRunCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return CloseRunCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeCloseRunPayloadV0(payload)
	if err := validateCloseRunCommandPayloadDataV0(payload); err != nil {
		return CloseRunCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateRunClosedPayloadV0(event OrchestrationEventV0) error {
	var payload RunClosedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateRunClosedPayloadDataV0(normalizeRunClosedPayloadV0(payload))
}

func handleCloseRunCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeCloseRunCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if runClosureAlreadyReflectedV0(current, payload.ClosureRef) {
		if err := ensureCommandEffectMatchesV0(current, command, OrchestrationEventRunClosedV0, payload.ClosureRef, runClosedPayloadFromCommandV0(payload)); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	if err := ensureCloseRunCommandAllowedV0(current, command, payload); err != nil {
		return emptyCommandResultV0(), err
	}
	event, err := NewRunClosedEventV0(commandEventMetaV0(current, command, OrchestrationEventRunClosedV0), runClosedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyRunClosedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload RunClosedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	payload = normalizeRunClosedPayloadV0(payload)
	if err := ensureRunClosedEventAllowedV0(current, event, payload); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.ClosureRef); err != nil {
		return current, err
	}

	next := cloneRunForReducerV0(current)
	next.Status = OrchestrationRunStatusClosedV0
	next.Closures = appendUniqueCompactRefV0(next.Closures, payload.ClosureRef)
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.ClosureRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func runClosedPayloadFromCommandV0(payload CloseRunCommandPayloadV0) RunClosedPayloadV0 {
	return RunClosedPayloadV0{
		ClosureRef:    payload.ClosureRef,
		PhaseID:       payload.PhaseID,
		ValidationRef: payload.ValidationRef,
		Summary:       payload.Summary,
		EvidenceRefs:  cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeCloseRunPayloadV0(payload CloseRunCommandPayloadV0) CloseRunCommandPayloadV0 {
	return CloseRunCommandPayloadV0{
		ClosureRef:    strings.TrimSpace(payload.ClosureRef),
		PhaseID:       strings.TrimSpace(payload.PhaseID),
		ValidationRef: strings.TrimSpace(payload.ValidationRef),
		Summary:       strings.TrimSpace(payload.Summary),
		EvidenceRefs:  compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizeRunClosedPayloadV0(payload RunClosedPayloadV0) RunClosedPayloadV0 {
	normalized := normalizeCloseRunPayloadV0(CloseRunCommandPayloadV0(payload))
	return runClosedPayloadFromCommandV0(normalized)
}

func runClosureAlreadyReflectedV0(current OrchestrationRunV0, closureRef string) bool {
	closureRef = strings.TrimSpace(closureRef)
	for _, existing := range current.Closures {
		if strings.TrimSpace(existing) == closureRef {
			return true
		}
	}
	return false
}
