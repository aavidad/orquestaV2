package orquestacoreworkflow

import "encoding/json"

func decodeAssessAgentWorkCommandPayloadV0(raw json.RawMessage) (AssessAgentWorkCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxAgentRequestPayloadBytesV0 {
		return AssessAgentWorkCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload AssessAgentWorkCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return AssessAgentWorkCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeAssessAgentWorkPayloadV0(payload)
	if err := validateAssessAgentWorkCommandPayloadDataV0(payload); err != nil {
		return AssessAgentWorkCommandPayloadV0{}, err
	}
	return payload, nil
}
