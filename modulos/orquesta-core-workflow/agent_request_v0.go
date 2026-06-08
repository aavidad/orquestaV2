package orquestacoreworkflow

type RequestAgentCommandPayloadV0 struct {
	AgentRequestID     string   `json:"agent_request_id"`
	PhaseID            string   `json:"phase_id"`
	TaskRef            string   `json:"task_ref,omitempty"`
	CapacityRequestRef string   `json:"capacity_request_ref"`
	Role               string   `json:"role"`
	Summary            string   `json:"summary"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
	SkillRefs          []string `json:"skill_refs,omitempty"`
}

type AgentRequestedPayloadV0 struct {
	AgentRequestID     string   `json:"agent_request_id"`
	PhaseID            string   `json:"phase_id"`
	TaskRef            string   `json:"task_ref,omitempty"`
	CapacityRequestRef string   `json:"capacity_request_ref"`
	Role               string   `json:"role"`
	Summary            string   `json:"summary"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
	SkillRefs          []string `json:"skill_refs,omitempty"`
}

func NewRequestAgentCommandV0(meta OrchestrationCommandMetaV0, payload RequestAgentCommandPayloadV0) (OrchestrationCommandV0, error) {
	return newOrchestrationCommandV0(meta, OrchestrationCommandRequestAgentV0, payload)
}

func NewAgentRequestedEventV0(meta OrchestrationEventMetaV0, payload AgentRequestedPayloadV0) (OrchestrationEventV0, error) {
	return newOrchestrationEventV0(meta, OrchestrationEventAgentRequestedV0, payload)
}

func agentRequestedPayloadFromCommandV0(payload RequestAgentCommandPayloadV0) AgentRequestedPayloadV0 {
	return AgentRequestedPayloadV0{
		AgentRequestID:     payload.AgentRequestID,
		PhaseID:            payload.PhaseID,
		TaskRef:            payload.TaskRef,
		CapacityRequestRef: payload.CapacityRequestRef,
		Role:               payload.Role,
		Summary:            payload.Summary,
		EvidenceRefs:       cloneStringsV0(payload.EvidenceRefs),
		SkillRefs:          compactUniqueStringsV0(payload.SkillRefs),
	}
}
