package orquestacoreworkflow

func validateRegisterAgentLostPayloadDataV0(payload RegisterAgentLostCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"agent_request_id": payload.AgentRequestID,
		"loss_ref":         payload.LossRef,
		"reason_code":      payload.ReasonCode,
		"observed_at":      payload.ObservedAt,
	}); err != nil {
		return err
	}
	return validateAgentLostCommonV0(agentLostTextFieldsV0(payload), payload.EvidenceRefs, true)
}

func validateAgentLostPayloadDataV0(payload AgentLostPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"agent_request_id": payload.AgentRequestID,
		"loss_ref":         payload.LossRef,
		"reason_code":      payload.ReasonCode,
		"observed_at":      payload.ObservedAt,
	}); err != nil {
		return err
	}
	return validateAgentLostCommonV0(
		agentLostTextFieldsV0(RegisterAgentLostCommandPayloadV0(payload)),
		payload.EvidenceRefs,
		false,
	)
}

func validateAgentLostCommonV0(values []string, evidenceRefs []string, command bool) error {
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

func agentLostTextFieldsV0(payload RegisterAgentLostCommandPayloadV0) []string {
	values := []string{
		payload.AgentRequestID,
		payload.LossRef,
		payload.ReasonCode,
		payload.ObservedAt,
	}
	return append(values, payload.EvidenceRefs...)
}
