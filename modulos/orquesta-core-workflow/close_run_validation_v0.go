package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

var forbiddenCloseRunFragmentsV0 = forbiddenReviewRequestFragmentsV0

func validateCloseRunCommandPayloadDataV0(payload CloseRunCommandPayloadV0) error {
	if err := validateCloseRunRequiredV0(payload); err != nil {
		return err
	}
	if closeRunStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if closeRunHasLongStringV0(closeRunTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if closeRunHasForbiddenDetailsV0(closeRunTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateCloseRunPayloadSizeV0(payload)
}

func validateRunClosedPayloadDataV0(payload RunClosedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"closure_ref":    payload.ClosureRef,
		"phase_id":       payload.PhaseID,
		"validation_ref": payload.ValidationRef,
		"summary":        payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateRunClosedPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if closeRunStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if closeRunHasLongStringV0(runClosedTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if closeRunHasForbiddenDetailsV0(runClosedTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateRunClosedPayloadSizeV0(payload)
}

func validateCloseRunRequiredV0(payload CloseRunCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"closure_ref":    payload.ClosureRef,
		"phase_id":       payload.PhaseID,
		"validation_ref": payload.ValidationRef,
		"summary":        payload.Summary,
	}); err != nil {
		return err
	}
	return validateCloseRunCommandPhaseIDV0(payload.PhaseID)
}

func validateCloseRunCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseCierreV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateRunClosedPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseCierreV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateCloseRunPayloadSizeV0(payload CloseRunCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxCloseRunPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateRunClosedPayloadSizeV0(payload RunClosedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxCloseRunPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func closeRunStringsInvalidV0(values []string) bool {
	if len(values) > maxCloseRunEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxCloseRunStringV0 {
			return true
		}
	}
	return false
}

func closeRunHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxCloseRunStringV0 {
			return true
		}
	}
	return false
}

func closeRunHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenCloseRunFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func closeRunTextFieldsV0(payload CloseRunCommandPayloadV0) []string {
	values := []string{payload.ClosureRef, payload.PhaseID, payload.ValidationRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func runClosedTextFieldsV0(payload RunClosedPayloadV0) []string {
	values := []string{payload.ClosureRef, payload.PhaseID, payload.ValidationRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
