package orquestadirector

import (
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const ErrDirectorConcurrencyGateAgentRequestsInvalidoV0 = "director_concurrency_gate_agent_requests_invalido"

type ConcurrencyGateAgentRequestsInputV0 struct {
	GateCommandMeta   orquestacoreworkflow.OrchestrationCommandMetaV0 `json:"gate_command_meta"`
	Claims            []orquestacoreconcurrency.WorksetClaimV0        `json:"claims"`
	SubjectClaimRefs  []string                                        `json:"subject_claim_refs"`
	EvidenceRefs      []string                                        `json:"evidence_refs,omitempty"`
	CandidateRequests []CandidateAgentRequestV0                       `json:"candidate_requests,omitempty"`
}

type CandidateAgentRequestV0 struct {
	ClaimRef    string                                            `json:"claim_ref"`
	CommandMeta orquestacoreworkflow.OrchestrationCommandMetaV0   `json:"command_meta"`
	Payload     orquestacoreworkflow.RequestAgentCommandPayloadV0 `json:"payload"`
}

type ConcurrencyGateAgentRequestsResultV0 struct {
	Evaluation       orquestacoreconcurrency.ConcurrencyGateEvaluationV0 `json:"evaluation"`
	GateCommand      orquestacoreworkflow.OrchestrationCommandV0         `json:"gate_command"`
	AgentCommands    []orquestacoreworkflow.OrchestrationCommandV0       `json:"agent_commands,omitempty"`
	BlockedClaimRefs []string                                            `json:"blocked_claim_refs,omitempty"`
	ConflictRefs     []string                                            `json:"conflict_refs,omitempty"`
	Issues           []ConcurrencyGateAgentRequestsIssueV0               `json:"issues,omitempty"`
}

type ConcurrencyGateAgentRequestsIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type ConcurrencyGateAgentRequestsErrorV0 struct {
	Code          string                                `json:"code"`
	Message       string                                `json:"message"`
	Field         string                                `json:"field,omitempty"`
	Issues        []ConcurrencyGateAgentRequestsIssueV0 `json:"issues,omitempty"`
	CorrelationID string                                `json:"correlation_id,omitempty"`
}

func (err ConcurrencyGateAgentRequestsErrorV0) Error() string {
	return err.Code
}
