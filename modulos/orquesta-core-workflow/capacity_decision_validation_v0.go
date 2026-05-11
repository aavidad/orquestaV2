package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func decodeRegisterCapacityDecisionCommandPayloadV0(raw json.RawMessage) (RegisterCapacityDecisionCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxCapacityRequestPayloadBytesV0 {
		return RegisterCapacityDecisionCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RegisterCapacityDecisionCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RegisterCapacityDecisionCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRegisterCapacityDecisionPayloadV0(payload)
	if err := validateRegisterCapacityDecisionPayloadDataV0(payload); err != nil {
		return RegisterCapacityDecisionCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateCapacityDecidedPayloadV0(event OrchestrationEventV0) error {
	var payload CapacityDecidedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateCapacityDecidedPayloadDataV0(normalizeCapacityDecidedPayloadV0(payload))
}

func validateRegisterCapacityDecisionPayloadDataV0(payload RegisterCapacityDecisionCommandPayloadV0) error {
	if err := validateCapacityDecisionRequiredForCommandV0(payload); err != nil {
		return err
	}
	if err := validateCapacityDecisionCommonV0(payload, true); err != nil {
		return err
	}
	return validateRegisterCapacityDecisionPayloadSizeV0(payload)
}

func validateCapacityDecidedPayloadDataV0(payload CapacityDecidedPayloadV0) error {
	commandPayload := RegisterCapacityDecisionCommandPayloadV0(payload)
	if err := requirePayloadFieldsV0(map[string]string{
		"capacity_request_id": commandPayload.CapacityRequestID,
		"decision_ref":        commandPayload.DecisionRef,
		"tier":                string(commandPayload.Tier),
		"reasoning_effort":    string(commandPayload.ReasoningEffort),
	}); err != nil {
		return err
	}
	if err := validateCapacityDecisionCommonV0(commandPayload, false); err != nil {
		return err
	}
	return validateCapacityDecidedPayloadSizeV0(payload)
}

func validateCapacityDecisionRequiredForCommandV0(payload RegisterCapacityDecisionCommandPayloadV0) error {
	return requireCommandPayloadFieldsV0(map[string]string{
		"capacity_request_id": payload.CapacityRequestID,
		"decision_ref":        payload.DecisionRef,
		"tier":                string(payload.Tier),
		"reasoning_effort":    string(payload.ReasoningEffort),
	})
}

func validateCapacityDecisionCommonV0(payload RegisterCapacityDecisionCommandPayloadV0, command bool) error {
	if !isSupportedCapacityRecommendationV0(payload.Tier) {
		return capacityDecisionPayloadErrorV0(command, "payload.tier")
	}
	if !isSupportedCapacityRecommendationV0(payload.ReasoningEffort) {
		return capacityDecisionPayloadErrorV0(command, "payload.reasoning_effort")
	}
	if strings.Contains(payload.CapacityRequestID, capacityDecisionProjectionSeparatorV0) ||
		strings.Contains(payload.DecisionRef, capacityDecisionProjectionSeparatorV0) {
		return capacityDecisionPayloadErrorV0(command, "payload")
	}
	if err := validateCapacityRequestCollectionsV0(payload.EvidenceRefs, "payload.evidence_refs", command); err != nil {
		return err
	}
	if capacityDecisionStringsInvalidV0(payload) {
		return capacityDecisionPayloadErrorV0(command, "payload")
	}
	if capacityRequestHasForbiddenDetailsV0(capacityDecisionTextFieldsV0(payload)) {
		if command {
			return commandErrorV0(ErrDetalleProhibidoV0, "payload")
		}
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return nil
}

func capacityDecisionPayloadErrorV0(command bool, field string) error {
	if command {
		return commandErrorV0(ErrPayloadInvalidoV0, field)
	}
	return eventErrorV0(ErrPayloadInvalidoV0, field)
}

func capacityDecisionStringsInvalidV0(payload RegisterCapacityDecisionCommandPayloadV0) bool {
	values := []string{payload.CapacityRequestID, payload.DecisionRef, string(payload.Tier), string(payload.ReasoningEffort)}
	if strings.TrimSpace(payload.Summary) != "" {
		values = append(values, payload.Summary)
	}
	for _, value := range values {
		if len(value) > maxCapacityRequestStringV0 {
			return true
		}
	}
	return false
}

func validateRegisterCapacityDecisionPayloadSizeV0(payload RegisterCapacityDecisionCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxCapacityRequestPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateCapacityDecidedPayloadSizeV0(payload CapacityDecidedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxCapacityRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}
