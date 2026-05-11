package orquestadirector

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func normalizeReplanFollowupsInputV0(input ReplanFollowupsInputV0) ReplanFollowupsInputV0 {
	input.DecisionCommandMeta = normalizeReplanFollowupCommandMetaV0(input.DecisionCommandMeta)
	input.DecisionPayload.ReplanRef = strings.TrimSpace(input.DecisionPayload.ReplanRef)
	input.DecisionPayload.RunRef = strings.TrimSpace(input.DecisionPayload.RunRef)
	input.DecisionPayload.TaskRef = strings.TrimSpace(input.DecisionPayload.TaskRef)
	input.DecisionPayload.SourceRef = strings.TrimSpace(input.DecisionPayload.SourceRef)
	input.DecisionPayload.AcceptedAction = orquestacoreworkflow.ReplanDecisionActionV0(
		strings.TrimSpace(string(input.DecisionPayload.AcceptedAction)),
	)
	input.DecisionPayload.FollowupRefs = compactReplanFollowupStringsV0(input.DecisionPayload.FollowupRefs)
	input.DecisionPayload.Summary = strings.TrimSpace(input.DecisionPayload.Summary)
	input.DecisionPayload.EvidenceRefs = compactReplanFollowupStringsV0(input.DecisionPayload.EvidenceRefs)
	input.SourceKind = strings.TrimSpace(input.SourceKind)
	input.BlockedAgentRefs = compactReplanFollowupStringsV0(input.BlockedAgentRefs)
	normalizeReplanFollowupCandidatesV0(&input)
	return input
}

func normalizeReplanFollowupCandidatesV0(input *ReplanFollowupsInputV0) {
	if input.OpenPhaseCandidate != nil {
		input.OpenPhaseCandidate.CommandMeta = normalizeReplanFollowupCommandMetaV0(input.OpenPhaseCandidate.CommandMeta)
		input.OpenPhaseCandidate.Payload.PhaseID = strings.TrimSpace(input.OpenPhaseCandidate.Payload.PhaseID)
		input.OpenPhaseCandidate.Payload.Reason = strings.TrimSpace(input.OpenPhaseCandidate.Payload.Reason)
	}
	input.MicrotaskCandidates = normalizeReplanMicrotaskCandidatesV0(input.MicrotaskCandidates)
	if input.CapacityCandidate != nil {
		input.CapacityCandidate.CommandMeta = normalizeReplanFollowupCommandMetaV0(input.CapacityCandidate.CommandMeta)
		input.CapacityCandidate.Payload.EvidenceRefs = compactReplanFollowupStringsV0(input.CapacityCandidate.Payload.EvidenceRefs)
	}
	if input.AgentCandidate != nil {
		input.AgentCandidate.CommandMeta = normalizeReplanFollowupCommandMetaV0(input.AgentCandidate.CommandMeta)
		input.AgentCandidate.Payload.EvidenceRefs = compactReplanFollowupStringsV0(input.AgentCandidate.Payload.EvidenceRefs)
	}
	if input.AskDirectorCandidate != nil {
		input.AskDirectorCandidate.CommandMeta = normalizeReplanFollowupCommandMetaV0(input.AskDirectorCandidate.CommandMeta)
		input.AskDirectorCandidate.Payload.Options = compactReplanFollowupStringsV0(input.AskDirectorCandidate.Payload.Options)
		input.AskDirectorCandidate.Payload.EvidenceRefs = compactReplanFollowupStringsV0(input.AskDirectorCandidate.Payload.EvidenceRefs)
	}
}

func normalizeReplanMicrotaskCandidatesV0(
	candidates []ReplanMicrotaskCandidateV0,
) []ReplanMicrotaskCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	normalized := make([]ReplanMicrotaskCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.CommandMeta = normalizeReplanFollowupCommandMetaV0(candidate.CommandMeta)
		candidate.Payload.Task = orquestacoreworkflow.NormalizeWorkflowTaskV0(candidate.Payload.Task)
		normalized = append(normalized, candidate)
	}
	return normalized
}

func normalizeReplanFollowupCommandMetaV0(
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

func compactReplanFollowupStringsV0(values []string) []string {
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
