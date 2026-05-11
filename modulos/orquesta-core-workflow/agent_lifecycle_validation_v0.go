package orquestacoreworkflow

import "encoding/json"

func validateRegisterAgentStartedPayloadDataV0(payload RegisterAgentStartedCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"agent_request_id": payload.AgentRequestID,
		"launch_ref":       payload.LaunchRef,
		"ack_ref":          payload.AckRef,
		"readiness_ref":    payload.ReadinessRef,
	}); err != nil {
		return err
	}
	return validateAgentLifecycleCommonV0(agentStartedTextFieldsV0(payload), payload.EvidenceRefs, true)
}

func validateAgentStartedPayloadV0(event OrchestrationEventV0) error {
	var payload AgentStartedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	payload = normalizeAgentStartedPayloadV0(payload)
	if err := requirePayloadFieldsV0(map[string]string{
		"agent_request_id": payload.AgentRequestID,
		"launch_ref":       payload.LaunchRef,
		"ack_ref":          payload.AckRef,
		"readiness_ref":    payload.ReadinessRef,
	}); err != nil {
		return err
	}
	return validateAgentLifecycleEventCommonV0(agentStartedTextFieldsV0(RegisterAgentStartedCommandPayloadV0(payload)), payload.EvidenceRefs, payload)
}

func validateRegisterAgentFailedPayloadDataV0(payload RegisterAgentFailedCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"agent_request_id": payload.AgentRequestID,
		"reason_code":      payload.ReasonCode,
	}); err != nil {
		return err
	}
	return validateAgentLifecycleCommonV0(agentFailedTextFieldsV0(payload), payload.EvidenceRefs, false)
}

func validateAgentFailedPayloadV0(event OrchestrationEventV0) error {
	var payload AgentFailedPayloadV0
	if err := decodePayloadV0(event.Payload, &payload); err != nil {
		return err
	}
	payload = normalizeAgentFailedPayloadV0(payload)
	if err := requirePayloadFieldsV0(map[string]string{
		"agent_request_id": payload.AgentRequestID,
		"reason_code":      payload.ReasonCode,
	}); err != nil {
		return err
	}
	return validateAgentLifecycleEventCommonV0(agentFailedTextFieldsV0(RegisterAgentFailedCommandPayloadV0(payload)), payload.EvidenceRefs, payload)
}

func validateAgentLifecycleCommonV0(values []string, evidenceRefs []string, command bool) error {
	if agentRequestStringsInvalidV0(evidenceRefs) {
		if command {
			return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
		}
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(values) {
		if command {
			return commandErrorV0(ErrDetalleProhibidoV0, "payload")
		}
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return nil
}

func validateAgentLifecycleEventCommonV0(values []string, evidenceRefs []string, payload any) error {
	if err := validateAgentLifecycleCommonV0(values, evidenceRefs, false); err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil || len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func agentStartedTextFieldsV0(payload RegisterAgentStartedCommandPayloadV0) []string {
	values := []string{payload.AgentRequestID, payload.LaunchRef, payload.AckRef, payload.ReadinessRef}
	return append(values, payload.EvidenceRefs...)
}

func agentFailedTextFieldsV0(payload RegisterAgentFailedCommandPayloadV0) []string {
	values := []string{payload.AgentRequestID, payload.LaunchRef, payload.ReasonCode}
	return append(values, payload.EvidenceRefs...)
}
