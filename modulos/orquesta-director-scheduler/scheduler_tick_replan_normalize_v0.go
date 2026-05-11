package orquestadirectorscheduler

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func normalizeSchedulableReplanFollowupCandidatesV0(
	candidates []SchedulableReplanFollowupCandidateV0,
) []SchedulableReplanFollowupCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	normalized := make([]SchedulableReplanFollowupCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, normalizeSchedulableReplanFollowupCandidateV0(candidate))
	}
	return normalized
}

func normalizeSchedulableReplanFollowupCandidateV0(
	candidate SchedulableReplanFollowupCandidateV0,
) SchedulableReplanFollowupCandidateV0 {
	return SchedulableReplanFollowupCandidateV0{
		CandidateRef: strings.TrimSpace(candidate.CandidateRef),
		ReplanFollowupsInput: normalizeSchedulerReplanFollowupsInputV0(
			candidate.ReplanFollowupsInput,
		),
		EvidenceRefs: compactSchedulerStringsV0(candidate.EvidenceRefs),
	}
}

func normalizeSchedulerReplanFollowupsInputV0(
	input orquestadirector.ReplanFollowupsInputV0,
) orquestadirector.ReplanFollowupsInputV0 {
	input.DecisionCommandMeta = normalizeSchedulerCommandMetaV0(input.DecisionCommandMeta)
	input.DecisionPayload = normalizeSchedulerReplanDecisionPayloadV0(input.DecisionPayload)
	input.BlockedAgentRefs = compactSchedulerStringsV0(input.BlockedAgentRefs)
	if input.OpenPhaseCandidate != nil {
		input.OpenPhaseCandidate.CommandMeta = normalizeSchedulerCommandMetaV0(input.OpenPhaseCandidate.CommandMeta)
		input.OpenPhaseCandidate.Payload.PhaseID = strings.TrimSpace(input.OpenPhaseCandidate.Payload.PhaseID)
		input.OpenPhaseCandidate.Payload.Reason = strings.TrimSpace(input.OpenPhaseCandidate.Payload.Reason)
	}
	input.MicrotaskCandidates = normalizeSchedulerReplanMicrotasksV0(input.MicrotaskCandidates)
	if input.CapacityCandidate != nil {
		input.CapacityCandidate.CommandMeta = normalizeSchedulerCommandMetaV0(input.CapacityCandidate.CommandMeta)
		input.CapacityCandidate.Payload.EvidenceRefs = compactSchedulerStringsV0(input.CapacityCandidate.Payload.EvidenceRefs)
	}
	if input.AgentCandidate != nil {
		input.AgentCandidate.CommandMeta = normalizeSchedulerCommandMetaV0(input.AgentCandidate.CommandMeta)
		input.AgentCandidate.Payload.EvidenceRefs = compactSchedulerStringsV0(input.AgentCandidate.Payload.EvidenceRefs)
	}
	if input.AskDirectorCandidate != nil {
		input.AskDirectorCandidate.CommandMeta = normalizeSchedulerCommandMetaV0(input.AskDirectorCandidate.CommandMeta)
		input.AskDirectorCandidate.Payload.Options = compactSchedulerStringsV0(input.AskDirectorCandidate.Payload.Options)
		input.AskDirectorCandidate.Payload.EvidenceRefs = compactSchedulerStringsV0(input.AskDirectorCandidate.Payload.EvidenceRefs)
	}
	return input
}

func normalizeSchedulerReplanMicrotasksV0(
	candidates []orquestadirector.ReplanMicrotaskCandidateV0,
) []orquestadirector.ReplanMicrotaskCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	normalized := make([]orquestadirector.ReplanMicrotaskCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.CommandMeta = normalizeSchedulerCommandMetaV0(candidate.CommandMeta)
		candidate.Payload.Task = orquestacoreworkflow.NormalizeWorkflowTaskV0(candidate.Payload.Task)
		normalized = append(normalized, candidate)
	}
	return normalized
}

func normalizeSchedulerReplanDecisionPayloadV0(
	payload orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0,
) orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0 {
	return orquestacoreworkflow.RecordReplanDecisionCommandPayloadV0{
		ReplanRef:      strings.TrimSpace(payload.ReplanRef),
		RunRef:         strings.TrimSpace(payload.RunRef),
		TaskRef:        strings.TrimSpace(payload.TaskRef),
		SourceRef:      strings.TrimSpace(payload.SourceRef),
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionV0(strings.TrimSpace(string(payload.AcceptedAction))),
		FollowupRefs:   compactSchedulerStringsV0(payload.FollowupRefs),
		Summary:        strings.TrimSpace(payload.Summary),
		EvidenceRefs:   compactSchedulerStringsV0(payload.EvidenceRefs),
	}
}
