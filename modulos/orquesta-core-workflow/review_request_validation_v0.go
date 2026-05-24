package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func validateRequestReviewCommandPayloadDataV0(payload RequestReviewCommandPayloadV0) error {
	if err := validateRequestReviewRequiredV0(payload); err != nil {
		return err
	}
	if reviewRequestStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if reviewRequestHasLongStringV0(requestReviewTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if reviewRequestHasForbiddenDetailsV0(requestReviewTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateRequestReviewPayloadSizeV0(payload)
}

func validateReviewRequestedPayloadDataV0(payload ReviewRequestedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"review_request_id": payload.ReviewRequestID,
		"phase_id":          payload.PhaseID,
		"delivery_ref":      payload.DeliveryRef,
		"summary":           payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateReviewRequestedPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if reviewRequestStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if reviewRequestHasLongStringV0(reviewRequestedTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if reviewRequestHasForbiddenDetailsV0(reviewRequestedTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateReviewRequestedPayloadSizeV0(payload)
}

func validateRequestReviewRequiredV0(payload RequestReviewCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"review_request_id": payload.ReviewRequestID,
		"phase_id":          payload.PhaseID,
		"delivery_ref":      payload.DeliveryRef,
		"summary":           payload.Summary,
	}); err != nil {
		return err
	}
	return validateRequestReviewCommandPhaseIDV0(payload.PhaseID)
}

func validateRequestReviewCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseRevisionV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateReviewRequestedPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseRevisionV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateRequestReviewPayloadSizeV0(payload RequestReviewCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReviewRequestPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateReviewRequestedPayloadSizeV0(payload ReviewRequestedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReviewRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func reviewRequestStringsInvalidV0(values []string) bool {
	if len(values) > maxReviewRequestEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxReviewRequestStringV0 {
			return true
		}
	}
	return false
}

func reviewRequestHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxReviewRequestStringV0 {
			return true
		}
	}
	return false
}

func reviewRequestHasForbiddenDetailsV0(values []string) bool {
	return textValuesContainForbiddenOperationalSensitiveDetailV0(values)
}

func requestReviewTextFieldsV0(payload RequestReviewCommandPayloadV0) []string {
	values := []string{payload.ReviewRequestID, payload.PhaseID, payload.DeliveryRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func reviewRequestedTextFieldsV0(payload ReviewRequestedPayloadV0) []string {
	values := []string{payload.ReviewRequestID, payload.PhaseID, payload.DeliveryRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
