package orquestacoreworkflow

import (
	"encoding/json"
)

type StopAgentCommandPayloadV0 struct {
	AgentRequestID string   `json:"agent_request_id"`
	ReasonCode     string   `json:"reason_code"`
	Summary        string   `json:"summary"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type AgentStopRequestedPayloadV0 struct {
	AgentRequestID string   `json:"agent_request_id"`
	ReasonCode     string   `json:"reason_code"`
	Summary        string   `json:"summary"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type StopRuntimeAgentRequestV0 struct {
	AgentRequestID string   `json:"agent_request_id"`
	RunID          string   `json:"run_id"`
	ReasonCode     string   `json:"reason_code"`
	Summary        string   `json:"summary"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

func NewStopAgentCommandV0(meta OrchestrationCommandMetaV0, payload StopAgentCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandStopAgentV0, payload)
}

func NewAgentStopRequestedEventV0(meta OrchestrationEventMetaV0, payload AgentStopRequestedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventAgentStopRequestedV0, payload)
}

func decodeStopAgentCommandPayloadV0(raw json.RawMessage) (StopAgentCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxAgentRequestPayloadBytesV0 {
		return StopAgentCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload StopAgentCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return StopAgentCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeStopAgentPayloadV0(payload)
	if err := validateStopAgentCommandPayloadDataV0(payload); err != nil {
		return StopAgentCommandPayloadV0{}, err
	}
	return payload, nil
}
