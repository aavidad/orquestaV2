package orquestacoreworkflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

const maxPhaseCloseEvidenceRefsV0 = 12

func decodeClosePhaseCommandPayloadV0(raw json.RawMessage) (ClosePhaseCommandPayloadV0, error) {
	var payload ClosePhaseCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ClosePhaseCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeClosePhasePayloadV0(payload)
	if err := validateClosePhaseCommandPayloadDataV0(payload); err != nil {
		return ClosePhaseCommandPayloadV0{}, err
	}
	return payload, nil
}

func validatePhaseClosedPayloadV0(event OrchestrationEventV0) error {
	var payload PhaseClosedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validatePhaseClosedPayloadDataV0(normalizePhaseClosedPayloadV0(payload))
}

func handleClosePhaseCommandV0(current OrchestrationRunV0, command OrchestrationCommandV0) (OrchestrationCommandResultV0, error) {
	payload, err := decodeClosePhaseCommandPayloadV0(command.Payload)
	if err != nil {
		return emptyCommandResultV0(), err
	}
	if reflected, err := phaseClosureEffectKnownV0(current, payload.ClosureRef); err != nil {
		return emptyCommandResultV0(), err
	} else if reflected {
		if err := ensurePhaseClosedEffectMatchesV0(current, command, payload); err != nil {
			return emptyCommandResultV0(), err
		}
		return idempotentCommandResultV0(), nil
	}
	if err := ensureActiveRunForCommandV0(current, command); err != nil {
		return emptyCommandResultV0(), err
	}
	phaseID := OrchestrationPhaseIDV0(payload.PhaseID)
	if closePhaseAlreadyReflectedV0(current, command, phaseID) {
		return idempotentCommandResultV0(), nil
	}
	if normalizePhaseIDV0(current.CurrentPhase) != phaseID {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	if !phaseIsCurrentAndActiveV0(current, phaseID) {
		return emptyCommandResultV0(), commandErrorV0(ErrTransicionInvalidaV0, "phases")
	}
	event, err := NewPhaseClosedEventV0(phaseClosedEventMetaV0(current, command, payload), phaseClosedPayloadFromCommandV0(payload))
	return eventCommandResultV0(event, err)
}

func applyPhaseClosedEventV0(current OrchestrationRunV0, event OrchestrationEventV0) (OrchestrationRunV0, error) {
	var payload PhaseClosedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return current, err
	}
	if err := ensureRunCanApplyEventV0(current, event); err != nil {
		return current, err
	}
	if err := ensureEventEffectCompatibleV0(current, event, payload.ClosureRef); err != nil {
		return current, err
	}

	phaseID := OrchestrationPhaseIDV0(strings.TrimSpace(payload.PhaseID))
	if normalizePhaseIDV0(current.CurrentPhase) != phaseID {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	if !runContainsPhaseV0(current, phaseID) {
		return current, issueV0(OrchestrationFaseInvalidaV0, "phases")
	}
	if !phaseIsCurrentAndActiveV0(current, phaseID) {
		return current, eventErrorV0(ErrSecuenciaInvalidaV0, "phase.status")
	}

	next := cloneRunForReducerV0(current)
	for index := range next.Phases {
		phase := &next.Phases[index]
		if normalizePhaseIDV0(phase.ID) == phaseID {
			phase.Status = OrchestrationPhaseStatusClosedV0
			phase.ClosedAt = strings.TrimSpace(event.OccurredAt)
			break
		}
	}
	effects, err := appendCommandEffectFromEventV0(next.CommandEffects, event, payload.ClosureRef)
	if err != nil {
		return current, err
	}
	next.CommandEffects = effects
	next.LastEventID = strings.TrimSpace(event.EventID)
	next.LastSequence = event.Sequence
	return next, nil
}

func phaseClosedPayloadFromCommandV0(payload ClosePhaseCommandPayloadV0) PhaseClosedPayloadV0 {
	return PhaseClosedPayloadV0{
		PhaseID:      payload.PhaseID,
		ClosureRef:   payload.ClosureRef,
		Summary:      payload.Summary,
		EvidenceRefs: cloneStringsV0(payload.EvidenceRefs),
	}
}

func normalizeClosePhasePayloadV0(payload ClosePhaseCommandPayloadV0) ClosePhaseCommandPayloadV0 {
	return ClosePhaseCommandPayloadV0{
		PhaseID:      strings.TrimSpace(payload.PhaseID),
		ClosureRef:   strings.TrimSpace(payload.ClosureRef),
		Summary:      strings.TrimSpace(payload.Summary),
		EvidenceRefs: compactUniqueStringsV0(payload.EvidenceRefs),
	}
}

func normalizePhaseClosedPayloadV0(payload PhaseClosedPayloadV0) PhaseClosedPayloadV0 {
	normalized := normalizeClosePhasePayloadV0(ClosePhaseCommandPayloadV0(payload))
	return PhaseClosedPayloadV0(normalized)
}

func validateClosePhaseCommandPayloadDataV0(payload ClosePhaseCommandPayloadV0) error {
	if err := validatePhaseCloseRequiredV0(payload); err != nil {
		return err
	}
	if phaseCloseHasForbiddenDetailsV0(phaseCloseTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return nil
}

func validatePhaseClosedPayloadDataV0(payload PhaseClosedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"phase_id":    payload.PhaseID,
		"closure_ref": payload.ClosureRef,
		"summary":     payload.Summary,
	}); err != nil {
		return err
	}
	if err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID)); err != nil {
		return err
	}
	if len(payload.EvidenceRefs) > maxPhaseCloseEvidenceRefsV0 || refsContainForbiddenDetailsV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload.evidence_refs")
	}
	return nil
}

func validatePhaseCloseRequiredV0(payload ClosePhaseCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"phase_id":    payload.PhaseID,
		"closure_ref": payload.ClosureRef,
		"summary":     payload.Summary,
	}); err != nil {
		return err
	}
	if err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID)); err != nil {
		return err
	}
	if len(payload.EvidenceRefs) > maxPhaseCloseEvidenceRefsV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	return nil
}

func phaseCloseTextFieldsV0(payload ClosePhaseCommandPayloadV0) []string {
	values := []string{payload.PhaseID, payload.ClosureRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func phaseCloseHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		if containsForbiddenEventTextV0(value) {
			return true
		}
	}
	return false
}

func phaseIsCurrentAndActiveV0(run OrchestrationRunV0, phaseID OrchestrationPhaseIDV0) bool {
	for _, phase := range run.Phases {
		if normalizePhaseIDV0(phase.ID) == phaseID {
			return phase.Status == OrchestrationPhaseStatusActiveV0
		}
	}
	return false
}

func closePhaseAlreadyReflectedV0(current OrchestrationRunV0, command OrchestrationCommandV0, phaseID OrchestrationPhaseIDV0) bool {
	payload, err := decodeClosePhaseCommandPayloadV0(command.Payload)
	if err != nil {
		return false
	}
	for _, phase := range current.Phases {
		if normalizePhaseIDV0(phase.ID) != phaseID {
			continue
		}
		return phase.Status == OrchestrationPhaseStatusClosedV0 &&
			strings.TrimSpace(current.LastEventID) == phaseClosedEventIDV0(command, payload)
	}
	return false
}

func phaseClosedEventMetaV0(current OrchestrationRunV0, command OrchestrationCommandV0, payload ClosePhaseCommandPayloadV0) OrchestrationEventMetaV0 {
	meta := commandEventMetaV0(current, command, OrchestrationEventPhaseClosedV0)
	meta.EventID = phaseClosedEventIDV0(command, payload)
	return meta
}

func phaseClosedEventIDV0(command OrchestrationCommandV0, payload ClosePhaseCommandPayloadV0) string {
	return commandEventIDV0(command, OrchestrationEventPhaseClosedV0) + "-" + closePhasePayloadHashV0(payload)
}

func closePhasePayloadHashV0(payload ClosePhaseCommandPayloadV0) string {
	data, err := json.Marshal(normalizeClosePhasePayloadV0(payload))
	if err != nil {
		return "payload-invalid"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

func compactUniqueStringsV0(values []string) []string {
	compacted := compactStringsV0(values)
	result := make([]string, 0, len(compacted))
	for _, value := range compacted {
		result = appendUniqueCompactRefV0(result, value)
	}
	return result
}
