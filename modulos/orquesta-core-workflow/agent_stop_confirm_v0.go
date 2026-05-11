package orquestacoreworkflow

import "encoding/json"

type RegisterAgentStopConfirmedCommandPayloadV0 struct {
	ConfirmationRef string   `json:"confirmation_ref"`
	AgentRequestID  string   `json:"agent_request_id"`
	ObservedAt      string   `json:"observed_at"`
	Summary         string   `json:"summary"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

type AgentStopConfirmedPayloadV0 struct {
	ConfirmationRef string   `json:"confirmation_ref"`
	AgentRequestID  string   `json:"agent_request_id"`
	ObservedAt      string   `json:"observed_at"`
	Summary         string   `json:"summary"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

func NewRegisterAgentStopConfirmedCommandV0(meta OrchestrationCommandMetaV0, payload RegisterAgentStopConfirmedCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRegisterAgentStopConfirmedV0, normalizeRegisterAgentStopConfirmedPayloadV0(payload))
}

func NewAgentStopConfirmedEventV0(meta OrchestrationEventMetaV0, payload AgentStopConfirmedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventAgentStopConfirmedV0, normalizeAgentStopConfirmedPayloadV0(payload))
}

func decodeRegisterAgentStopConfirmedCommandPayloadV0(raw json.RawMessage) (RegisterAgentStopConfirmedCommandPayloadV0, error) {
	if len(raw) == 0 || len(raw) > maxAgentRequestPayloadBytesV0 {
		return RegisterAgentStopConfirmedCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	var payload RegisterAgentStopConfirmedCommandPayloadV0
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RegisterAgentStopConfirmedCommandPayloadV0{}, commandErrorV0(ErrPayloadInvalidoV0, "payload")
	}
	payload = normalizeRegisterAgentStopConfirmedPayloadV0(payload)
	if err := validateRegisterAgentStopConfirmedPayloadDataV0(payload); err != nil {
		return RegisterAgentStopConfirmedCommandPayloadV0{}, err
	}
	return payload, nil
}
