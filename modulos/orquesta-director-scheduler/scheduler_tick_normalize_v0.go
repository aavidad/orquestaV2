package orquestadirectorscheduler

import (
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func NormalizeDirectorSchedulerTickInputV0(input DirectorSchedulerTickInputV0) DirectorSchedulerTickInputV0 {
	runRef := strings.TrimSpace(input.RunRef)
	snapshot := normalizeRunSchedulingSnapshotV0(input.Snapshot, runRef)
	return DirectorSchedulerTickInputV0{
		TickRef:                       strings.TrimSpace(input.TickRef),
		RunRef:                        runRef,
		OccurredAt:                    strings.TrimSpace(input.OccurredAt),
		Snapshot:                      snapshot,
		LeaseActionCandidates:         normalizeSchedulableLeaseActionCandidatesV0(input.LeaseActionCandidates),
		PhaseArtifactCandidates:       normalizeSchedulablePhaseArtifactCandidatesV0(input.PhaseArtifactCandidates),
		DeliveryCandidates:            normalizeSchedulableDeliveryCandidatesV0(input.DeliveryCandidates),
		ReviewGateCandidates:          normalizeSchedulableReviewGateCandidatesV0(input.ReviewGateCandidates),
		ProgressSupervisionCandidates: normalizeSchedulableProgressSupervisionCandidatesV0(input.ProgressSupervisionCandidates),
		ReplanFollowupCandidates: normalizeSchedulableReplanFollowupCandidatesV0(
			input.ReplanFollowupCandidates,
		),
		WorkClaims:     normalizeSchedulerWorkClaimsV0(input.WorkClaims),
		WorkCandidates: normalizeSchedulableWorkCandidatesV0(input.WorkCandidates),
		EvidenceRefs:   compactSchedulerStringsV0(input.EvidenceRefs),
	}
}

func normalizeRunSchedulingSnapshotV0(snapshot RunSchedulingSnapshotV0, fallbackRunRef string) RunSchedulingSnapshotV0 {
	runRef := strings.TrimSpace(snapshot.RunRef)
	if runRef == "" {
		runRef = fallbackRunRef
	}
	return RunSchedulingSnapshotV0{
		RunRef:                    runRef,
		CurrentPhaseID:            strings.TrimSpace(snapshot.CurrentPhaseID),
		Tasks:                     compactSchedulerStringsV0(snapshot.Tasks),
		CapacityRequests:          compactSchedulerStringsV0(snapshot.CapacityRequests),
		CapacityDecisions:         compactSchedulerStringsV0(snapshot.CapacityDecisions),
		ConcurrencyGates:          compactSchedulerStringsV0(snapshot.ConcurrencyGates),
		Agents:                    compactSchedulerStringsV0(snapshot.Agents),
		StartedAgents:             compactSchedulerStringsV0(snapshot.StartedAgents),
		FailedAgents:              compactSchedulerStringsV0(snapshot.FailedAgents),
		StoppedAgents:             compactSchedulerStringsV0(snapshot.StoppedAgents),
		PhaseArtifacts:            compactSchedulerStringsV0(snapshot.PhaseArtifacts),
		Deliveries:                compactSchedulerStringsV0(snapshot.Deliveries),
		Reviews:                   compactSchedulerStringsV0(snapshot.Reviews),
		ReviewResults:             compactSchedulerStringsV0(snapshot.ReviewResults),
		AcceptedReviews:           compactSchedulerStringsV0(snapshot.AcceptedReviews),
		ReworkRequests:            compactSchedulerStringsV0(snapshot.ReworkRequests),
		AgentAssessments:          compactSchedulerStringsV0(snapshot.AgentAssessments),
		DirectorQuestions:         compactSchedulerStringsV0(snapshot.DirectorQuestions),
		DirectorAnsweredQuestions: compactSchedulerStringsV0(snapshot.DirectorAnsweredQuestions),
		ExpiredLeaseRefs:          compactSchedulerStringsV0(snapshot.ExpiredLeaseRefs),
		ReplanRefs:                compactSchedulerStringsV0(snapshot.ReplanRefs),
		BlockingQualityGateRefs:   compactSchedulerStringsV0(snapshot.BlockingQualityGateRefs),
		PendingOutboxRefs:         compactSchedulerStringsV0(snapshot.PendingOutboxRefs),
	}
}

func normalizeSchedulableWorkCandidatesV0(candidates []SchedulableWorkCandidateV0) []SchedulableWorkCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	normalized := make([]SchedulableWorkCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, normalizeSchedulableWorkCandidateV0(candidate))
	}
	return normalized
}

func normalizeSchedulableWorkCandidateV0(candidate SchedulableWorkCandidateV0) SchedulableWorkCandidateV0 {
	return SchedulableWorkCandidateV0{
		CandidateRef:      strings.TrimSpace(candidate.CandidateRef),
		SubjectClaimRefs:  compactSchedulerStringsV0(candidate.SubjectClaimRefs),
		Claims:            normalizeSchedulerWorkClaimsV0(candidate.Claims),
		CapacityCandidate: normalizeSchedulerCapacityCandidateV0(candidate.CapacityCandidate),
		AgentCandidate:    normalizeSchedulerAgentCandidateV0(candidate.AgentCandidate),
		LiveWorkPolicy:    normalizeSchedulerLiveWorkPolicyV0(candidate.LiveWorkPolicy),
		GateCommandMeta:   normalizeSchedulerCommandMetaV0(candidate.GateCommandMeta),
		GateEvidenceRefs:  compactSchedulerStringsV0(candidate.GateEvidenceRefs),
		EvidenceRefs:      compactSchedulerStringsV0(candidate.EvidenceRefs),
	}
}

func normalizeSchedulerWorkClaimsV0(
	claims []orquestacoreconcurrency.WorksetClaimV0,
) []orquestacoreconcurrency.WorksetClaimV0 {
	if len(claims) == 0 {
		return nil
	}
	normalized := make([]orquestacoreconcurrency.WorksetClaimV0, 0, len(claims))
	for _, claim := range claims {
		value, _ := orquestacoreconcurrency.NormalizeWorksetClaimV0(claim)
		normalized = append(normalized, value)
	}
	return normalized
}

func normalizeSchedulerCapacityCandidateV0(candidate *SchedulerCapacityCommandCandidateV0) *SchedulerCapacityCommandCandidateV0 {
	if candidate == nil {
		return nil
	}
	return &SchedulerCapacityCommandCandidateV0{
		CommandMeta: normalizeSchedulerCommandMetaV0(candidate.CommandMeta),
		Payload:     candidate.Payload,
	}
}

func normalizeSchedulerAgentCandidateV0(candidate *SchedulerAgentCommandCandidateV0) *SchedulerAgentCommandCandidateV0 {
	if candidate == nil {
		return nil
	}
	return &SchedulerAgentCommandCandidateV0{
		ClaimRef:    strings.TrimSpace(candidate.ClaimRef),
		CommandMeta: normalizeSchedulerCommandMetaV0(candidate.CommandMeta),
		Payload:     candidate.Payload,
	}
}

func normalizeSchedulablePhaseArtifactCandidatesV0(
	candidates []SchedulablePhaseArtifactCandidateV0,
) []SchedulablePhaseArtifactCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	normalized := make([]SchedulablePhaseArtifactCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, SchedulablePhaseArtifactCandidateV0{
			CandidateRef: strings.TrimSpace(candidate.CandidateRef),
			CommandMeta:  normalizeSchedulerCommandMetaV0(candidate.CommandMeta),
			Payload:      normalizeSchedulerPhaseArtifactPayloadV0(candidate.Payload),
			EvidenceRefs: compactSchedulerStringsV0(candidate.EvidenceRefs),
		})
	}
	return normalized
}

func normalizeSchedulerPhaseArtifactPayloadV0(
	payload orquestacoreworkflow.RegisterPhaseArtifactCommandPayloadV0,
) orquestacoreworkflow.RegisterPhaseArtifactCommandPayloadV0 {
	payload.ArtifactRef = strings.TrimSpace(payload.ArtifactRef)
	payload.PhaseID = strings.TrimSpace(payload.PhaseID)
	payload.AgentRef = strings.TrimSpace(payload.AgentRef)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EvidenceRefs = compactSchedulerStringsV0(payload.EvidenceRefs)
	return payload
}

func normalizeSchedulableDeliveryCandidatesV0(
	candidates []SchedulableDeliveryCandidateV0,
) []SchedulableDeliveryCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	normalized := make([]SchedulableDeliveryCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, SchedulableDeliveryCandidateV0{
			CandidateRef: strings.TrimSpace(candidate.CandidateRef),
			CommandMeta:  normalizeSchedulerCommandMetaV0(candidate.CommandMeta),
			Payload:      normalizeSchedulerDeliveryPayloadV0(candidate.Payload),
			EvidenceRefs: compactSchedulerStringsV0(candidate.EvidenceRefs),
		})
	}
	return normalized
}

func normalizeSchedulerDeliveryPayloadV0(
	payload orquestacoreworkflow.RegisterDeliveryCommandPayloadV0,
) orquestacoreworkflow.RegisterDeliveryCommandPayloadV0 {
	payload.DeliveryRef = strings.TrimSpace(payload.DeliveryRef)
	payload.PhaseID = strings.TrimSpace(payload.PhaseID)
	payload.TaskID = strings.TrimSpace(payload.TaskID)
	payload.AgentRef = strings.TrimSpace(payload.AgentRef)
	payload.Summary = strings.TrimSpace(payload.Summary)
	payload.EvidenceRefs = compactSchedulerStringsV0(payload.EvidenceRefs)
	return payload
}

func normalizeSchedulableLeaseActionCandidatesV0(
	candidates []SchedulableLeaseActionCandidateV0,
) []SchedulableLeaseActionCandidateV0 {
	if len(candidates) == 0 {
		return nil
	}
	normalized := make([]SchedulableLeaseActionCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, normalizeSchedulableLeaseActionCandidateV0(candidate))
	}
	return normalized
}

func normalizeSchedulableLeaseActionCandidateV0(
	candidate SchedulableLeaseActionCandidateV0,
) SchedulableLeaseActionCandidateV0 {
	return SchedulableLeaseActionCandidateV0{
		CandidateRef:         strings.TrimSpace(candidate.CandidateRef),
		PostLeaseActionInput: normalizeSchedulerPostLeaseActionInputV0(candidate.PostLeaseActionInput),
		EvidenceRefs:         compactSchedulerStringsV0(candidate.EvidenceRefs),
	}
}

func normalizeSchedulerPostLeaseActionInputV0(
	input orquestadirector.PostLeaseActionInputV0,
) orquestadirector.PostLeaseActionInputV0 {
	input.CommandMeta = normalizeSchedulerCommandMetaV0(input.CommandMeta)
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.AgentRequestID = strings.TrimSpace(input.AgentRequestID)
	input.LeaseRef = strings.TrimSpace(input.LeaseRef)
	input.ReasonCode = strings.TrimSpace(input.ReasonCode)
	input.ObservedAt = strings.TrimSpace(input.ObservedAt)
	input.RecommendedAction = orquestacoreworkflow.AgentLeaseRecommendedActionV0(
		strings.TrimSpace(string(input.RecommendedAction)),
	)
	input.EvidenceRefs = compactSchedulerStringsV0(input.EvidenceRefs)
	input.QuestionID = strings.TrimSpace(input.QuestionID)
	return input
}

func normalizeSchedulerCommandMetaV0(meta orquestacoreworkflow.OrchestrationCommandMetaV0) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      strings.TrimSpace(meta.CommandID),
		RunID:          strings.TrimSpace(meta.RunID),
		IdempotencyKey: strings.TrimSpace(meta.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(meta.CorrelationID),
		RequestedBy:    strings.TrimSpace(meta.RequestedBy),
		OccurredAt:     strings.TrimSpace(meta.OccurredAt),
	}
}
