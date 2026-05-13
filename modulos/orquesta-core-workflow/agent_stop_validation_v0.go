package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func validateAgentStopRequestedPayloadV0(event OrchestrationEventV0) error {
	var payload AgentStopRequestedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	return validateAgentStopRequestedPayloadDataV0(normalizeAgentStopRequestedPayloadV0(payload))
}

func validateStopAgentCommandPayloadDataV0(payload StopAgentCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(stopAgentRequiredFieldsV0(payload)); err != nil {
		return err
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(stopAgentTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	if agentStopProjectionFieldUnsafeV0(payload) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return validateStopAgentPayloadSizeV0(payload)
}

func validateAgentStopRequestedPayloadDataV0(payload AgentStopRequestedPayloadV0) error {
	if err := requirePayloadFieldsV0(stopEventRequiredFieldsV0(payload)); err != nil {
		return err
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(stopAgentTextFieldsV0(StopAgentCommandPayloadV0(payload))) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	if agentStopProjectionFieldUnsafeV0(StopAgentCommandPayloadV0(payload)) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return validateAgentStopPayloadSizeV0(payload)
}

func validateStopRuntimeAgentPayloadDataV0(payload StopRuntimeAgentRequestV0) error {
	if strings.TrimSpace(payload.RunID) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.run_id")
	}
	if strings.TrimSpace(payload.AgentRequestID) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.agent_request_id")
	}
	if strings.TrimSpace(payload.ReasonCode) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.reason_code")
	}
	if strings.TrimSpace(payload.Summary) == "" {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.summary")
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(stopRuntimeAgentTextFieldsV0(payload)) {
		return outboxErrorV0(ErrOutboxDetalleProhibidoV0, "payload")
	}
	return validateStopRuntimeAgentPayloadSizeV0(payload)
}

func stopAgentRequiredFieldsV0(payload StopAgentCommandPayloadV0) map[string]string {
	return map[string]string{"agent_request_id": payload.AgentRequestID, "reason_code": payload.ReasonCode, "summary": payload.Summary}
}

func stopEventRequiredFieldsV0(payload AgentStopRequestedPayloadV0) map[string]string {
	return map[string]string{"agent_request_id": payload.AgentRequestID, "reason_code": payload.ReasonCode, "summary": payload.Summary}
}

func validateStopAgentPayloadSizeV0(payload StopAgentCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateAgentStopPayloadSizeV0(payload AgentStopRequestedPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateStopRuntimeAgentPayloadSizeV0(payload StopRuntimeAgentRequestV0) error {
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return outboxErrorV0(ErrOutboxPayloadInvalidoV0, "payload")
	}
	return nil
}

func stopAgentTextFieldsV0(payload StopAgentCommandPayloadV0) []string {
	values := []string{payload.AgentRequestID, payload.ReasonCode, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}

func stopRuntimeAgentTextFieldsV0(payload StopRuntimeAgentRequestV0) []string {
	values := []string{payload.AgentRequestID, payload.RunID, payload.ReasonCode, payload.Summary}
	return append(values, payload.EvidenceRefs...)
}
