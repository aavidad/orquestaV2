package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

var forbiddenReviewReworkFragmentsV0 = forbiddenReviewRequestFragmentsV0

func validateRequestReworkCommandPayloadDataV0(payload RequestReworkCommandPayloadV0) error {
	if err := validateRequestReworkRequiredV0(payload); err != nil {
		return err
	}
	if reworkRequestProjectionFieldUnsafeV0(payload) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.rework_request_ref")
	}
	if reviewReworkStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if reviewReworkHasLongStringV0(requestReworkTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if reviewReworkHasForbiddenDetailsV0(requestReworkTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateRequestReworkPayloadSizeV0(payload)
}

func validateReworkRequestedPayloadDataV0(payload ReworkRequestedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"rework_request_ref": payload.ReworkRequestRef,
		"phase_id":           payload.PhaseID,
		"review_result_ref":  payload.ReviewResultRef,
		"review_request_id":  payload.ReviewRequestID,
		"delivery_ref":       payload.DeliveryRef,
		"summary":            payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateReworkRequestedPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if reworkRequestedProjectionFieldUnsafeV0(payload) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.rework_request_ref")
	}
	if reviewReworkStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if reviewReworkHasLongStringV0(reworkRequestedTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if reviewReworkHasForbiddenDetailsV0(reworkRequestedTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateReworkRequestedPayloadSizeV0(payload)
}

func validateRequestReworkRequiredV0(payload RequestReworkCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"rework_request_ref": payload.ReworkRequestRef,
		"phase_id":           payload.PhaseID,
		"review_result_ref":  payload.ReviewResultRef,
		"review_request_id":  payload.ReviewRequestID,
		"delivery_ref":       payload.DeliveryRef,
		"summary":            payload.Summary,
	}); err != nil {
		return err
	}
	return validateRequestReworkCommandPhaseIDV0(payload.PhaseID)
}

func validateRequestReworkCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseRevisionV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateReworkRequestedPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseRevisionV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateRequestReworkPayloadSizeV0(payload RequestReworkCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReviewReworkPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateReworkRequestedPayloadSizeV0(payload ReworkRequestedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReviewReworkPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func reviewReworkStringsInvalidV0(values []string) bool {
	if len(values) > maxReviewReworkEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxReviewReworkStringV0 {
			return true
		}
	}
	return false
}

func reviewReworkHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxReviewReworkStringV0 {
			return true
		}
	}
	return false
}

func reviewReworkHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenReviewReworkFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func requestReworkTextFieldsV0(payload RequestReworkCommandPayloadV0) []string {
	values := []string{payload.ReworkRequestRef, payload.PhaseID, payload.ReviewResultRef, payload.ReviewRequestID, payload.DeliveryRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func reworkRequestedTextFieldsV0(payload ReworkRequestedPayloadV0) []string {
	values := []string{payload.ReworkRequestRef, payload.PhaseID, payload.ReviewResultRef, payload.ReviewRequestID, payload.DeliveryRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
