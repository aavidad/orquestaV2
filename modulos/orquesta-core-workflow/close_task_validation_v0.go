package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func validateCloseTaskCommandPayloadDataV0(payload CloseTaskCommandPayloadV0) error {
	if err := validateCloseTaskRequiredV0(payload); err != nil {
		return err
	}
	if closeTaskStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if closeTaskHasLongStringV0(closeTaskTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if closeTaskHasForbiddenDetailsV0(closeTaskTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateCloseTaskPayloadSizeV0(payload)
}

func validateTaskClosedPayloadDataV0(payload TaskClosedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"task_id":             payload.TaskID,
		"phase_id":            payload.PhaseID,
		"delivery_ref":        payload.DeliveryRef,
		"accepted_review_ref": payload.AcceptedReviewRef,
		"summary":             payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateTaskClosedPhaseIDV0(payload.PhaseID); err != nil {
		return err
	}
	if closeTaskStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if closeTaskHasLongStringV0(taskClosedTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if closeTaskHasForbiddenDetailsV0(taskClosedTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateTaskClosedPayloadSizeV0(payload)
}

func validateCloseTaskRequiredV0(payload CloseTaskCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"task_id":             payload.TaskID,
		"phase_id":            payload.PhaseID,
		"delivery_ref":        payload.DeliveryRef,
		"accepted_review_ref": payload.AcceptedReviewRef,
		"summary":             payload.Summary,
	}); err != nil {
		return err
	}
	return validateCloseTaskCommandPhaseIDV0(payload.PhaseID)
}

func validateCloseTaskCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseRevisionV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateTaskClosedPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseRevisionV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateCloseTaskPayloadSizeV0(payload CloseTaskCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxCloseTaskPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateTaskClosedPayloadSizeV0(payload TaskClosedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxCloseTaskPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func closeTaskStringsInvalidV0(values []string) bool {
	if len(values) > maxCloseTaskEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxCloseTaskStringV0 {
			return true
		}
	}
	return false
}

func closeTaskHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxCloseTaskStringV0 {
			return true
		}
	}
	return false
}

func closeTaskHasForbiddenDetailsV0(values []string) bool {
	return textValuesContainForbiddenOperationalSensitiveDetailV0(values)
}

func closeTaskTextFieldsV0(payload CloseTaskCommandPayloadV0) []string {
	values := []string{payload.TaskID, payload.PhaseID, payload.DeliveryRef, payload.AcceptedReviewRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func taskClosedTextFieldsV0(payload TaskClosedPayloadV0) []string {
	values := []string{payload.TaskID, payload.PhaseID, payload.DeliveryRef, payload.AcceptedReviewRef, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
