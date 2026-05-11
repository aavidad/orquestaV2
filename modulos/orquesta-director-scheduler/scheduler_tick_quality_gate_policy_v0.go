package orquestadirectorscheduler

func schedulerHasBlockingQualityGateBeforeWorkV0(input DirectorSchedulerTickInputV0) bool {
	return len(input.Snapshot.BlockingQualityGateRefs) > 0
}

func schedulerReplanFollowupCandidatesForBlockingQualityGateV0(
	input DirectorSchedulerTickInputV0,
) []SchedulableReplanFollowupCandidateV0 {
	blockingRefs := schedulerStringSetV0(input.Snapshot.BlockingQualityGateRefs)
	candidates := make([]SchedulableReplanFollowupCandidateV0, 0, len(input.ReplanFollowupCandidates))
	for _, candidate := range input.ReplanFollowupCandidates {
		sourceRef := candidate.ReplanFollowupsInput.DecisionPayload.SourceRef
		if blockingRefs[sourceRef] {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func schedulerBlockingQualityGatePlanV0(input DirectorSchedulerTickInputV0) DirectorSchedulerTickPlanV0 {
	return DirectorSchedulerTickPlanV0{
		TickRef:        input.TickRef,
		RunRef:         input.RunRef,
		Status:         SchedulerTickStatusBlockedV0,
		WaitingReasons: []SchedulerWaitingReasonV0{SchedulerWaitingQualityGateFollowupV0},
		BlockedRefs:    compactSchedulerStringsV0(input.Snapshot.BlockingQualityGateRefs),
		Summary:        "scheduler tick blocked by quality gate",
		EvidenceRefs:   input.EvidenceRefs,
	}
}
