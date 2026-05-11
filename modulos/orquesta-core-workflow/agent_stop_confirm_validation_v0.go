package orquestacoreworkflow

import "encoding/json"

func validateAgentStopConfirmedPayloadV0(event OrchestrationEventV0) error {
	var payload AgentStopConfirmedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateAgentStopConfirmedPayloadDataV0(normalizeAgentStopConfirmedPayloadV0(payload))
}

func validateRegisterAgentStopConfirmedPayloadDataV0(payload RegisterAgentStopConfirmedCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(agentStopConfirmedRequiredFieldsV0(payload)); err != nil {
		return err
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(agentStopConfirmedTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateRegisterAgentStopConfirmedPayloadSizeV0(payload)
}

func validateAgentStopConfirmedPayloadDataV0(payload AgentStopConfirmedPayloadV0) error {
	if err := requirePayloadFieldsV0(agentStopConfirmedEventRequiredFieldsV0(payload)); err != nil {
		return err
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(agentStopConfirmedTextFieldsV0(RegisterAgentStopConfirmedCommandPayloadV0(payload))) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateAgentStopConfirmedPayloadSizeV0(payload)
}

func agentStopConfirmedRequiredFieldsV0(payload RegisterAgentStopConfirmedCommandPayloadV0) map[string]string {
	return map[string]string{
		"confirmation_ref": payload.ConfirmationRef,
		"agent_request_id": payload.AgentRequestID,
		"observed_at":      payload.ObservedAt,
		"summary":          payload.Summary,
	}
}

func agentStopConfirmedEventRequiredFieldsV0(payload AgentStopConfirmedPayloadV0) map[string]string {
	return agentStopConfirmedRequiredFieldsV0(RegisterAgentStopConfirmedCommandPayloadV0(payload))
}

func validateRegisterAgentStopConfirmedPayloadSizeV0(payload RegisterAgentStopConfirmedCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateAgentStopConfirmedPayloadSizeV0(payload AgentStopConfirmedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}
