package orquestacoreworkflow

type RegisterAgentStartedCommandPayloadV0 struct {
	AgentRequestID string   `json:"agent_request_id"`
	LaunchRef      string   `json:"launch_ref"`
	AckRef         string   `json:"ack_ref"`
	ReadinessRef   string   `json:"readiness_ref"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type AgentStartedPayloadV0 RegisterAgentStartedCommandPayloadV0

type RegisterAgentFailedCommandPayloadV0 struct {
	AgentRequestID string   `json:"agent_request_id"`
	LaunchRef      string   `json:"launch_ref,omitempty"`
	ReasonCode     string   `json:"reason_code"`
	Retryable      bool     `json:"retryable"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type AgentFailedPayloadV0 RegisterAgentFailedCommandPayloadV0

func NewRegisterAgentStartedCommandV0(meta OrchestrationCommandMetaV0, payload RegisterAgentStartedCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRegisterAgentStartedV0, normalizeRegisterAgentStartedPayloadV0(payload))
}

func NewRegisterAgentFailedCommandV0(meta OrchestrationCommandMetaV0, payload RegisterAgentFailedCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRegisterAgentFailedV0, normalizeRegisterAgentFailedPayloadV0(payload))
}

func NewAgentStartedEventV0(meta OrchestrationEventMetaV0, payload AgentStartedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventAgentStartedV0, normalizeAgentStartedPayloadV0(payload))
}

func NewAgentFailedEventV0(meta OrchestrationEventMetaV0, payload AgentFailedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventAgentFailedV0, normalizeAgentFailedPayloadV0(payload))
}
