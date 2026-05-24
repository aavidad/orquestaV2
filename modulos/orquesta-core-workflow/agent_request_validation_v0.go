package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

const (
	maxAgentRequestPayloadBytesV0 = 2048
	maxAgentRequestStringV0       = 600
	maxAgentRequestEvidenceRefsV0 = 20
)

func validateRequestAgentCommandPayloadDataV0(payload RequestAgentCommandPayloadV0) error {
	if err := validateAgentRequestRequiredV0(payload); err != nil {
		return err
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(agentRequestTextFieldsV0(payload)) {
		return commandErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	return validateAgentRequestPayloadSizeV0(payload)
}

func validateAgentRequestedPayloadDataV0(payload AgentRequestedPayloadV0) error {
	if err := requirePayloadFieldsV0(map[string]string{
		"agent_request_id":     payload.AgentRequestID,
		"phase_id":             payload.PhaseID,
		"capacity_request_ref": payload.CapacityRequestRef,
		"role":                 payload.Role,
		"summary":              payload.Summary,
	}); err != nil {
		return err
	}
	if err := ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID)); err != nil {
		return err
	}
	if agentRequestStringsInvalidV0(payload.EvidenceRefs) {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload.evidence_refs")
	}
	if agentRequestHasForbiddenDetailsV0(agentRequestTextFieldsV0(RequestAgentCommandPayloadV0(payload))) {
		return eventErrorV0(ErrDetalleProhibidoV0, "payload")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return eventErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func validateAgentRequestRequiredV0(payload RequestAgentCommandPayloadV0) error {
	if err := requireCommandPayloadFieldsV0(map[string]string{
		"agent_request_id":     payload.AgentRequestID,
		"phase_id":             payload.PhaseID,
		"capacity_request_ref": payload.CapacityRequestRef,
		"role":                 payload.Role,
		"summary":              payload.Summary,
	}); err != nil {
		return err
	}
	return ValidateOrchestrationPhaseIDV0(OrchestrationPhaseIDV0(payload.PhaseID))
}

func validateAgentRequestPayloadSizeV0(payload RequestAgentCommandPayloadV0) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxAgentRequestPayloadBytesV0 {
		return commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	return nil
}

func agentRequestStringsInvalidV0(values []string) bool {
	if len(values) > maxAgentRequestEvidenceRefsV0 {
		return true
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxAgentRequestStringV0 {
			return true
		}
	}
	return false
}

func agentRequestHasForbiddenDetailsV0(values []string) bool {
	return textValuesContainForbiddenOperationalSensitiveDetailV0(values)
}
