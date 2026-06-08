package orquestadirectorcandidates

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const ErrDirectorCandidateInvalidoV0 = "director_candidate_invalido"

type SchedulableWorkCandidateInputV0 struct {
	CandidateRef     string                                   `json:"candidate_ref"`
	RunRef           string                                   `json:"run_ref"`
	PhaseID          string                                   `json:"phase_id"`
	TaskRef          string                                   `json:"task_ref"`
	SubjectClaimRefs []string                                 `json:"subject_claim_refs"`
	Claims           []orquestacoreconcurrency.WorksetClaimV0 `json:"claims,omitempty"`
	ScopeClaims      []WorkCandidateScopeClaimV0              `json:"scope_claims,omitempty"`
	Commands         WorkCandidateCommandsV0                  `json:"commands"`
	Capacity         WorkCandidateCapacityInputV0             `json:"capacity"`
	Agent            WorkCandidateAgentInputV0                `json:"agent"`
	GateEvidenceRefs []string                                 `json:"gate_evidence_refs,omitempty"`
	EvidenceRefs     []string                                 `json:"evidence_refs,omitempty"`
}

type WorkCandidateCommandsV0 struct {
	CapacityCommandID      string `json:"capacity_command_id"`
	CapacityIdempotencyKey string `json:"capacity_idempotency_key"`
	GateCommandID          string `json:"gate_command_id"`
	GateIdempotencyKey     string `json:"gate_idempotency_key"`
	AgentCommandID         string `json:"agent_command_id"`
	AgentIdempotencyKey    string `json:"agent_idempotency_key"`
	CorrelationID          string `json:"correlation_id,omitempty"`
	RequestedBy            string `json:"requested_by,omitempty"`
	OccurredAt             string `json:"occurred_at"`
}

type WorkCandidateCapacityInputV0 struct {
	CapacityRequestID          string                                                     `json:"capacity_request_id"`
	ReasonCode                 string                                                     `json:"reason_code"`
	Summary                    string                                                     `json:"summary"`
	MinimumRecommendedCapacity orquestacoreworkflow.OrchestrationCapacityRecommendationV0 `json:"minimum_recommended_capacity,omitempty"`
	EvidenceRefs               []string                                                   `json:"evidence_refs,omitempty"`
}

type WorkCandidateAgentInputV0 struct {
	AgentRequestID string   `json:"agent_request_id"`
	ClaimRef       string   `json:"claim_ref"`
	Role           string   `json:"role"`
	Summary        string   `json:"summary"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
	SkillRefs      []string `json:"skill_refs,omitempty"`
}

type WorkCandidateScopeClaimV0 struct {
	ClaimRef       string   `json:"claim_ref"`
	AgentRequestID string   `json:"agent_request_id"`
	GroupRef       string   `json:"group_ref,omitempty"`
	ReadScopes     []string `json:"read_scopes,omitempty"`
	WriteScopes    []string `json:"write_scopes"`
	DependsOn      []string `json:"depends_on,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type DirectorCandidateErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err DirectorCandidateErrorV0) Error() string {
	return err.Code
}
