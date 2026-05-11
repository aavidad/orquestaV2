package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

var forbiddenFinalValidationFragmentsV0 = forbiddenReviewRequestFragmentsV0

func validateRegisterFinalValidationCommandPayloadDataV0(payload RegisterFinalValidationCommandPayloadV0) error {
	if err := validateRegisterFinalValidationRequiredV0(payload); err != nil {
		return err
	}
	if finalValidationStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if finalValidationHasLongStringV0(registerFinalValidationTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if finalValidationHasForbiddenDetailsV0(registerFinalValidationTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateRegisterFinalValidationPayloadSizeV0(payload)
}

func validateFinalValidationRegisteredPayloadDataV0(payload FinalValidationRegisteredPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"validation_ref":  payload.ValidationRef,
		"phase_id":        payload.PhaseID,
		"closed_task_ref": payload.ClosedTaskRef,
		"summary":         payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateFinalValidationRegisteredPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if finalValidationStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if finalValidationHasLongStringV0(finalValidationRegisteredTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if finalValidationHasForbiddenDetailsV0(finalValidationRegisteredTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateFinalValidationRegisteredPayloadSizeV0(payload)
}

func validateRegisterFinalValidationRequiredV0(payload RegisterFinalValidationCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"validation_ref":  payload.ValidationRef,
		"phase_id":        payload.PhaseID,
		"closed_task_ref": payload.ClosedTaskRef,
		"summary":         payload.Summary,
	}); err != nil {
		return err
	}
	return validateRegisterFinalValidationCommandPhaseIDV0(payload.PhaseID)
}

func validateRegisterFinalValidationCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseValidacionFinalV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateFinalValidationRegisteredPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseValidacionFinalV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateRegisterFinalValidationPayloadSizeV0(payload RegisterFinalValidationCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxFinalValidationPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateFinalValidationRegisteredPayloadSizeV0(payload FinalValidationRegisteredPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxFinalValidationPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func finalValidationStringsInvalidV0(values []string) bool {
	if len(values) > maxFinalValidationEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxFinalValidationStringV0 {
			return true
		}
	}
	return false
}

func finalValidationHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxFinalValidationStringV0 {
			return true
		}
	}
	return false
}

func finalValidationHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenFinalValidationFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func registerFinalValidationTextFieldsV0(payload RegisterFinalValidationCommandPayloadV0) []string {
	values := []string{payload.ValidationRef, payload.PhaseID, payload.ClosedTaskRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func finalValidationRegisteredTextFieldsV0(payload FinalValidationRegisteredPayloadV0) []string {
	values := []string{payload.ValidationRef, payload.PhaseID, payload.ClosedTaskRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
