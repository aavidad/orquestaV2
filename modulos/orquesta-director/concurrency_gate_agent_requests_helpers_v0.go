package orquestadirector

import (
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func normalizeConcurrencyGateAgentRequestsInputV0(
	input ConcurrencyGateAgentRequestsInputV0,
) ConcurrencyGateAgentRequestsInputV0 {
	input.GateCommandMeta = normalizeConcurrencyGateCommandMetaV0(input.GateCommandMeta)
	input.SubjectClaimRefs = compactConcurrencyGateStringsV0(input.SubjectClaimRefs)
	input.EvidenceRefs = compactConcurrencyGateStringsV0(input.EvidenceRefs)
	for index := range input.CandidateRequests {
		input.CandidateRequests[index].ClaimRef = strings.TrimSpace(input.CandidateRequests[index].ClaimRef)
		input.CandidateRequests[index].CommandMeta = normalizeConcurrencyGateCommandMetaV0(input.CandidateRequests[index].CommandMeta)
		input.CandidateRequests[index].Payload.EvidenceRefs = compactConcurrencyGateStringsV0(input.CandidateRequests[index].Payload.EvidenceRefs)
	}
	return input
}

func normalizeConcurrencyGateCommandMetaV0(
	meta orquestacoreworkflow.OrchestrationCommandMetaV0,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	meta.CommandID = strings.TrimSpace(meta.CommandID)
	meta.RunID = strings.TrimSpace(meta.RunID)
	meta.IdempotencyKey = strings.TrimSpace(meta.IdempotencyKey)
	meta.CorrelationID = strings.TrimSpace(meta.CorrelationID)
	meta.RequestedBy = strings.TrimSpace(meta.RequestedBy)
	meta.OccurredAt = strings.TrimSpace(meta.OccurredAt)
	return meta
}

func recordConcurrencyGatePayloadV0(
	evaluation orquestacoreconcurrency.ConcurrencyGateEvaluationV0,
	evidenceRefs []string,
) orquestacoreworkflow.RecordConcurrencyGateCommandPayloadV0 {
	return orquestacoreworkflow.RecordConcurrencyGateCommandPayloadV0{
		RunRef:           evaluation.RunRef,
		GateRef:          evaluation.GateRef,
		PlanRef:          evaluation.PlanRef,
		SubjectClaimRefs: cloneConcurrencyGateStringsV0(evaluation.SubjectClaimRefs),
		ReadyClaimRefs:   cloneConcurrencyGateStringsV0(evaluation.ReadyClaimRefs),
		BlockedClaimRefs: cloneConcurrencyGateStringsV0(evaluation.BlockedClaimRefs),
		ConflictRefs:     cloneConcurrencyGateStringsV0(evaluation.ConflictRefs),
		Decision:         orquestacoreworkflow.ConcurrencyGateDecisionV0(evaluation.Decision),
		Summary:          evaluation.Summary,
		EvidenceRefs:     cloneConcurrencyGateStringsV0(evidenceRefs),
	}
}

func candidatesByClaimRefV0(candidates []CandidateAgentRequestV0) map[string]CandidateAgentRequestV0 {
	byClaim := make(map[string]CandidateAgentRequestV0, len(candidates))
	for _, candidate := range candidates {
		claimRef := strings.TrimSpace(candidate.ClaimRef)
		if claimRef == "" {
			continue
		}
		byClaim[claimRef] = candidate
	}
	return byClaim
}

func compactConcurrencyGateStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func cloneConcurrencyGateStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func concurrencyGateAgentRequestsIssueV0(
	field string,
	message string,
) ConcurrencyGateAgentRequestsIssueV0 {
	return ConcurrencyGateAgentRequestsIssueV0{
		Code:    ErrDirectorConcurrencyGateAgentRequestsInvalidoV0,
		Field:   field,
		Message: message,
	}
}
