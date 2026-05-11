package orquestacoreworkflow

import "encoding/json"

func decodeRequestAgentCommandPayloadV0(raw json.RawMessage) (RequestAgentCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxAgentRequestPayloadBytesV0 {
		return RequestAgentCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RequestAgentCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RequestAgentCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRequestAgentPayloadV0(payload)
	if err := validateRequestAgentCommandPayloadDataV0(payload); err != nil {
		return RequestAgentCommandPayloadV0{}, err
	}
	return payload, nil
}

func validateAgentRequestedPayloadV0(event OrchestrationEventV0) error {
	var payload AgentRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateAgentRequestedPayloadDataV0(normalizeAgentRequestedPayloadV0(payload))
}
