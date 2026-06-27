package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxCapacityRequestPayloadBytesV0 = 2048
	maxCapacityRequestStringV0       = 600
	maxCapacityRequestEvidenceRefsV0 = 20
)

func decodeRequestCapacityCommandPayloadV0(raw json.RawMessage) (RequestCapacityCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxCapacityRequestPayloadBytesV0 {
		return RequestCapacityCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RequestCapacityCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RequestCapacityCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRequestCapacityPayloadV0(payload)
	if err := validateRequestCapacityCommandPayloadDataV0(payload); err != nil {
		return RequestCapacityCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateCapacityRequestedPayloadV0(event OrchestrationEventV0) error {
	var payload CapacityRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateCapacityRequestedPayloadDataV0(normalizeCapacityRequestedPayloadV0(payload))
}

func normalizeRequestCapacityPayloadV0(payload RequestCapacityCommandPayloadV0) RequestCapacityCommandPayloadV0 {
	return RequestCapacityCommandPayloadV0{
		CapacityRequestID:          strings.TrimSpace(payload.CapacityRequestID),
		PhaseID:                    strings.TrimSpace(payload.PhaseID),
		TaskRef:                    strings.TrimSpace(payload.TaskRef),
		ReasonCode:                 strings.TrimSpace(payload.ReasonCode),
		Summary:                    strings.TrimSpace(payload.Summary),
		MinimumRecommendedCapacity: OrchestrationCapacityRecommendationV0(strings.TrimSpace(string(payload.MinimumRecommendedCapacity))),
		EvidenceRefs:               compactStringsV0(payload.EvidenceRefs),
	}
}

func normalizeCapacityRequestedPayloadV0(payload CapacityRequestedPayloadV0) CapacityRequestedPayloadV0 {
	commandPayload := RequestCapacityCommandPayloadV0(payload)
	normalized := normalizeRequestCapacityPayloadV0(commandPayload)
	return capacityRequestedPayloadFromCommandV0(normalized)
}

func validateRequestCapacityCommandPayloadDataV0(payload RequestCapacityCommandPayloadV0) error {
	if err := validateCapacityRequestRequiredV0(payload); err != nil {
		return err
	}
	if err := validateCapacityRequestCollectionsV0(payload.EvidenceRefs, "payload.evidence_refs", true); err != nil {
		return err
	}
	if capacityRequestHasForbiddenDetailsV0(capacityRequestTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateCapacityRequestPayloadSizeV0(payload)
}

func validateCapacityRequestedPayloadDataV0(payload CapacityRequestedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"capacity_request_id": payload.CapacityRequestID,
		"phase_id":            payload.PhaseID,
		"reason_code":         payload.ReasonCode,
		"summary":             payload.Summary,
	}); err != nil {
		return err
	}
	if err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID)); err != nil {
		return err
	}
	if strings.TrimSpace(string(payload.MinimumRecommendedCapacity)) != "" && !isSupportedCapacityRecommendationV0(payload.MinimumRecommendedCapacity) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.minimum_recommended_capacity")
	}
	if len(payload.EvidenceRefs) > maxCapacityRequestEvidenceRefsV0 || capacityRequestStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if capacityRequestHasForbiddenDetailsV0(capacityRequestedTextFieldsV0(payload)) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateCapacityRequestedPayloadSizeV0(payload)
}

func validateCapacityRequestRequiredV0(payload RequestCapacityCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"capacity_request_id": payload.CapacityRequestID,
		"phase_id":            payload.PhaseID,
		"reason_code":         payload.ReasonCode,
		"summary":             payload.Summary,
	}); err != nil {
		return err
	}
	if err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID)); err != nil {
		return err
	}
	if strings.TrimSpace(string(payload.MinimumRecommendedCapacity)) != "" && !isSupportedCapacityRecommendationV0(payload.MinimumRecommendedCapacity) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.minimum_recommended_capacity")
	}
	return nil
}

func validateCapacityRequestCollectionsV0(values []string, field string, command bool) error {
	if len(values) > maxCapacityRequestEvidenceRefsV0 || capacityRequestStringsInvalidV0(values) {
		if command {
			return commandErrorV0(ErrPayloadInvalidoV0, field)
		}
		return eventErrorV0(ErrPayloadInvalidoV0, field)
	}
	return nil
}

func validateCapacityRequestPayloadSizeV0(payload RequestCapacityCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxCapacityRequestPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateCapacityRequestedPayloadSizeV0(payload CapacityRequestedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxCapacityRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func capacityRequestStringsInvalidV0(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxCapacityRequestStringV0 {
			return true
		}
	}
	return false
}

func capacityRequestHasForbiddenDetailsV0(values []string) bool {
	return textValuesContainForbiddenOperationalSensitiveDetailV0(values)
}

func capacityRequestTextFieldsV0(payload RequestCapacityCommandPayloadV0) []string {
	values := []string{payload.CapacityRequestID, payload.PhaseID, payload.TaskRef, payload.ReasonCode, payload.Summary, string(payload.MinimumRecommendedCapacity)}
	return append(values, payload.EvidenceRefs...)
}

func capacityRequestedTextFieldsV0(payload CapacityRequestedPayloadV0) []string {
	values := []string{payload.CapacityRequestID, payload.PhaseID, payload.TaskRef, payload.ReasonCode, payload.Summary, string(payload.MinimumRecommendedCapacity)}
	return append(values, payload.EvidenceRefs...)
}
