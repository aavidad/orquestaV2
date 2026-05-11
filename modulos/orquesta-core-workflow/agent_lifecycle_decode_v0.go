package orquestacoreworkflow

import "encoding/json"

func decodeRegisterAgentStartedCommandPayloadV0(raw json.RawMessage) (RegisterAgentStartedCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxAgentRequestPayloadBytesV0 {
		return RegisterAgentStartedCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RegisterAgentStartedCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RegisterAgentStartedCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRegisterAgentStartedPayloadV0(payload)
	if err := validateRegisterAgentStartedPayloadDataV0(payload); err != nil {
		return RegisterAgentStartedCommandPayloadV0{}, err
	}
	return payload, nil
}

func decodeRegisterAgentFailedCommandPayloadV0(raw json.RawMessage) (RegisterAgentFailedCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxAgentRequestPayloadBytesV0 {
		return RegisterAgentFailedCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RegisterAgentFailedCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RegisterAgentFailedCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRegisterAgentFailedPayloadV0(payload)
	if err := validateRegisterAgentFailedPayloadDataV0(payload); err != nil {
		return RegisterAgentFailedCommandPayloadV0{}, err
	}
	return payload, nil
}
