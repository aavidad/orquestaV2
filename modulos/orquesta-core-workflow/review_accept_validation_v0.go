package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

var forbiddenReviewAcceptFragmentsV0 = forbiddenReviewRequestFragmentsV0

func validateAcceptReviewCommandPayloadDataV0(payload AcceptReviewCommandPayloadV0) error {
	if err := validateAcceptReviewRequiredV0(payload); err != nil {
		return err
	}
	if reviewAcceptStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if reviewAcceptHasLongStringV0(acceptReviewTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if reviewAcceptHasForbiddenDetailsV0(acceptReviewTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateAcceptReviewPayloadSizeV0(payload)
}

func validateReviewAcceptedPayloadDataV0(payload ReviewAcceptedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"accepted_review_ref": payload.AcceptedReviewRef,
		"phase_id":            payload.PhaseID,
		"review_request_id":   payload.ReviewRequestID,
		"delivery_ref":        payload.DeliveryRef,
		"summary":             payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateReviewAcceptedPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if reviewAcceptStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if reviewAcceptHasLongStringV0(reviewAcceptedTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if reviewAcceptHasForbiddenDetailsV0(reviewAcceptedTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateReviewAcceptedPayloadSizeV0(payload)
}

func validateAcceptReviewRequiredV0(payload AcceptReviewCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"accepted_review_ref": payload.AcceptedReviewRef,
		"phase_id":            payload.PhaseID,
		"review_request_id":   payload.ReviewRequestID,
		"delivery_ref":        payload.DeliveryRef,
		"summary":             payload.Summary,
	}); err != nil {
		return err
	}
	return validateAcceptReviewCommandPhaseIDV0(payload.PhaseID)
}

func validateAcceptReviewCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseRevisionV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateReviewAcceptedPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseRevisionV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateAcceptReviewPayloadSizeV0(payload AcceptReviewCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReviewAcceptPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateReviewAcceptedPayloadSizeV0(payload ReviewAcceptedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxReviewAcceptPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func reviewAcceptStringsInvalidV0(values []string) bool {
	if len(values) > maxReviewAcceptEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxReviewAcceptStringV0 {
			return true
		}
	}
	return false
}

func reviewAcceptHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxReviewAcceptStringV0 {
			return true
		}
	}
	return false
}

func reviewAcceptHasForbiddenDetailsV0(values []string) bool {
	for _, value := range values {
		lower := strings.ToLower(value)
		for _, fragment := range forbiddenReviewAcceptFragmentsV0 {
			if containsForbiddenFragmentV0(lower, fragment) {
				return true
			}
		}
	}
	return false
}

func acceptReviewTextFieldsV0(payload AcceptReviewCommandPayloadV0) []string {
	values := []string{payload.AcceptedReviewRef, payload.PhaseID, payload.ReviewRequestID, payload.DeliveryRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func reviewAcceptedTextFieldsV0(payload ReviewAcceptedPayloadV0) []string {
	values := []string{payload.AcceptedReviewRef, payload.PhaseID, payload.ReviewRequestID, payload.DeliveryRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
