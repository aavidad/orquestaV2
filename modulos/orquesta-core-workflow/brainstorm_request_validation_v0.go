package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func validateRequestBrainstormCommandPayloadDataV0(payload RequestBrainstormCommandPayloadV0) error {
	if err := validateBrainstormRequestRequiredV0(payload); err != nil {
		return err
	}
	if brainstormRequestStringsInvalidV0(payload.EvidenceRefs) || brainstormRequestHasLongStringV0(brainstormRequestTextFieldsV0(payload)) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if brainstormRequestHasForbiddenDetailsV0(brainstormRequestTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateBrainstormRequestPayloadSizeV0(payload)
}

func validateBrainstormRequestedPayloadDataV0(payload BrainstormRequestedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"brainstorm_request_id": payload.BrainstormRequestID,
		"phase_id":              payload.PhaseID,
		"topic_ref":             payload.TopicRef,
		"summary":               payload.Summary,
	}); err != nil {
		return err
	}
	if err := validateBrainstormRequestEventCommonV0(payload.PhaseID, payload.MinimumRecommendedCapacity); err != nil {
		return err
	}
	if brainstormRequestStringsInvalidV0(payload.EvidenceRefs) || brainstormRequestHasLongStringV0(brainstormRequestedTextFieldsV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if brainstormRequestHasForbiddenDetailsV0(brainstormRequestedTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateBrainstormRequestedPayloadSizeV0(payload)
}

func validateBrainstormRequestRequiredV0(payload RequestBrainstormCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"brainstorm_request_id": payload.BrainstormRequestID,
		"phase_id":              payload.PhaseID,
		"topic_ref":             payload.TopicRef,
		"summary":               payload.Summary,
	}); err != nil {
		return err
	}
	return validateBrainstormRequestCommandCommonV0(payload.PhaseID, payload.MinimumRecommendedCapacity)
}

func validateBrainstormRequestCommandCommonV0(phaseID string, capacity OrchestrationCapacityRecommendationV0) error {
	if err := validateBrainstormCommandPhaseIDV0(phaseID); err != nil {
		return err
	}
	if strings.TrimSpace(string(capacity)) != "" && !isSupportedCapacityRecommendationV0(capacity) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.minimum_recommended_capacity")
	}
	return nil
}

func validateBrainstormRequestEventCommonV0(phaseID string, capacity OrchestrationCapacityRecommendationV0) error {
	if err := validateBrainstormEventPhaseIDV0(phaseID); err != nil {
		return err
	}
	if strings.TrimSpace(string(capacity)) != "" && !isSupportedCapacityRecommendationV0(capacity) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.minimum_recommended_capacity")
	}
	return nil
}

func validateBrainstormCommandPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseBrainstormingArquitecturaV0 {
		return commandErrorV0(ErrTransicionInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateBrainstormEventPhaseIDV0(phaseID string) error {
	phase := OrchestrationPhaseIDV0(strings.TrimSpace(phaseID))
	if err := ValidateOrchestrationPhaseIDV0(phase); err != nil {
		return err
	}
	if phase != OrchestrationPhaseBrainstormingArquitecturaV0 {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "payload.phase_id")
	}
	return nil
}

func validateBrainstormRequestPayloadSizeV0(payload RequestBrainstormCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxBrainstormRequestPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateBrainstormRequestedPayloadSizeV0(payload BrainstormRequestedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxBrainstormRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func brainstormRequestStringsInvalidV0(values []string) bool {
	if len(values) > maxBrainstormRequestEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxBrainstormRequestStringV0 {
			return true
		}
	}
	return false
}

func brainstormRequestHasLongStringV0(values []string) bool {
	for _, value := range values {
		if len(value) > maxBrainstormRequestStringV0 {
			return true
		}
	}
	return false
}

func brainstormRequestHasForbiddenDetailsV0(values []string) bool {
	return textValuesContainForbiddenOperationalSensitiveDetailV0(values)
}

func brainstormRequestTextFieldsV0(payload RequestBrainstormCommandPayloadV0) []string {
	values := []string{payload.BrainstormRequestID, payload.PhaseID, payload.TopicRef, payload.Summary, string(payload.MinimumRecommendedCapacity)}
	return append(values, payload.EvidenceRefs...)
}

func brainstormRequestedTextFieldsV0(payload BrainstormRequestedPayloadV0) []string {
	values := []string{payload.BrainstormRequestID, payload.PhaseID, payload.TopicRef, payload.Summary, string(payload.MinimumRecommendedCapacity)}
	return append(values, payload.EvidenceRefs...)
}
