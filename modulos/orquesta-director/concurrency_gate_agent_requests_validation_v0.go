package orquestadirector

import (
	"strconv"
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
)

func validateConcurrencyGateAgentRequestsInputV0(input ConcurrencyGateAgentRequestsInputV0) error {
	if input.GateCommandMeta.RunID == "" {
		return concurrencyGateAgentRequestsErrorV0(input, "command_meta.run_id", "run_id requerido", nil)
	}
	if len(input.Claims) == 0 {
		return concurrencyGateAgentRequestsErrorV0(input, "claims", "claims requeridos", nil)
	}
	if len(input.SubjectClaimRefs) == 0 {
		return concurrencyGateAgentRequestsErrorV0(input, "subject_claim_refs", "subject refs requeridos", nil)
	}
	for index, claim := range input.Claims {
		normalized, issues := orquestacoreconcurrency.NormalizeWorksetClaimV0(claim)
		if len(issues) > 0 {
			return concurrencyGateAgentRequestsErrorV0(
				input,
				"claims",
				"workset claim invalido",
				[]ConcurrencyGateAgentRequestsIssueV0{worksetClaimIssueV0(index, issues[0])},
			)
		}
		if normalized.RunRef != input.GateCommandMeta.RunID {
			return concurrencyGateAgentRequestsErrorV0(input, "claims.run_ref", "run_ref no coincide", nil)
		}
	}
	for index, candidate := range input.CandidateRequests {
		if candidate.ClaimRef == "" {
			return concurrencyGateAgentRequestsErrorV0(input, "candidate_requests.claim_ref", "claim_ref requerido", nil)
		}
		if candidate.CommandMeta.RunID != input.GateCommandMeta.RunID {
			return concurrencyGateAgentRequestsErrorV0(input, "candidate_requests.command_meta.run_id", "run_id no coincide", nil)
		}
		if strings.TrimSpace(candidate.Payload.AgentRequestID) == "" {
			return concurrencyGateAgentRequestsErrorV0(input, "candidate_requests.payload.agent_request_id", "agent_request_id requerido", nil)
		}
		if candidate.CommandMeta.CommandID == input.GateCommandMeta.CommandID {
			return concurrencyGateAgentRequestsErrorV0(input, "candidate_requests.command_meta.command_id", "command_id duplicado", nil)
		}
		for previous := 0; previous < index; previous++ {
			if input.CandidateRequests[previous].ClaimRef == candidate.ClaimRef {
				return concurrencyGateAgentRequestsErrorV0(input, "candidate_requests.claim_ref", "claim_ref duplicado", nil)
			}
		}
	}
	return nil
}

func validateAllowConcurrencyGateCandidatesV0(
	input ConcurrencyGateAgentRequestsInputV0,
	evaluation orquestacoreconcurrency.ConcurrencyGateEvaluationV0,
) error {
	candidates := candidatesByClaimRefV0(input.CandidateRequests)
	for _, claimRef := range evaluation.SubjectClaimRefs {
		candidate, ok := candidates[claimRef]
		if !ok {
			return concurrencyGateAgentRequestsErrorV0(input, "candidate_requests", "candidate request ausente", nil)
		}
		if strings.TrimSpace(candidate.Payload.AgentRequestID) == "" {
			return concurrencyGateAgentRequestsErrorV0(input, "candidate_requests.payload.agent_request_id", "agent_request_id requerido", nil)
		}
	}
	return nil
}

func worksetClaimIssueV0(
	index int,
	issue orquestacoreconcurrency.WorksetClaimIssueV0,
) ConcurrencyGateAgentRequestsIssueV0 {
	return ConcurrencyGateAgentRequestsIssueV0{
		Code:    string(issue.Code),
		Field:   "claims." + strconv.Itoa(index) + "." + issue.Field,
		Message: "workset claim invalido",
	}
}

func concurrencyGateAgentRequestsErrorV0(
	input ConcurrencyGateAgentRequestsInputV0,
	field string,
	message string,
	issues []ConcurrencyGateAgentRequestsIssueV0,
) ConcurrencyGateAgentRequestsErrorV0 {
	if len(issues) == 0 {
		issues = []ConcurrencyGateAgentRequestsIssueV0{concurrencyGateAgentRequestsIssueV0(field, message)}
	}
	return ConcurrencyGateAgentRequestsErrorV0{
		Code:          ErrDirectorConcurrencyGateAgentRequestsInvalidoV0,
		Message:       message,
		Field:         field,
		Issues:        issues,
		CorrelationID: input.GateCommandMeta.CorrelationID,
	}
}
