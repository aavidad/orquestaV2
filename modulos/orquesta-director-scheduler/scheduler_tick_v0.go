package orquestadirectorscheduler

func BuildDirectorSchedulerTickV0(input DirectorSchedulerTickInputV0) (DirectorSchedulerTickPlanV0, error) {
	normalized := NormalizeDirectorSchedulerTickInputV0(input)
	if err := ValidateDirectorSchedulerTickInputV0(normalized); err != nil {
		return DirectorSchedulerTickPlanV0{}, err
	}
	if len(normalized.Snapshot.PendingOutboxRefs) > 0 {
		return schedulerWaitingPlanV0(normalized, SchedulerWaitingOutboxPendingV0), nil
	}
	collector := newSchedulerTickCollectorV0(normalized)
	if len(normalized.LeaseActionCandidates) > 0 {
		if err := collector.collectLeaseActionCandidateV0(normalized.LeaseActionCandidates[0]); err != nil {
			return DirectorSchedulerTickPlanV0{}, err
		}
		return collector.planV0(), nil
	}
	if len(normalized.PhaseArtifactCandidates) > 0 {
		if err := collector.collectPhaseArtifactCandidateV0(normalized.PhaseArtifactCandidates[0]); err != nil {
			return DirectorSchedulerTickPlanV0{}, err
		}
		return collector.planV0(), nil
	}
	if len(normalized.DeliveryCandidates) > 0 {
		if err := collector.collectDeliveryCandidateV0(normalized.DeliveryCandidates[0]); err != nil {
			return DirectorSchedulerTickPlanV0{}, err
		}
		return collector.planV0(), nil
	}
	if len(normalized.ReviewGateCandidates) > 0 {
		if err := collector.collectReviewGateCandidateV0(normalized.ReviewGateCandidates[0]); err != nil {
			return DirectorSchedulerTickPlanV0{}, err
		}
		return collector.planV0(), nil
	}
	if len(normalized.ProgressSupervisionCandidates) > 0 {
		if err := collector.collectProgressSupervisionCandidateV0(normalized.ProgressSupervisionCandidates[0]); err != nil {
			return DirectorSchedulerTickPlanV0{}, err
		}
		return collector.planV0(), nil
	}
	if schedulerHasBlockingQualityGateBeforeWorkV0(normalized) {
		candidates := schedulerReplanFollowupCandidatesForBlockingQualityGateV0(normalized)
		if len(candidates) == 0 {
			return schedulerBlockingQualityGatePlanV0(normalized), nil
		}
		for _, candidate := range candidates {
			if err := collector.collectReplanFollowupCandidateV0(candidate); err != nil {
				return DirectorSchedulerTickPlanV0{}, err
			}
		}
		return collector.planV0(), nil
	}
	if len(normalized.ReplanFollowupCandidates) > 0 {
		for _, candidate := range normalized.ReplanFollowupCandidates {
			if err := collector.collectReplanFollowupCandidateV0(candidate); err != nil {
				return DirectorSchedulerTickPlanV0{}, err
			}
		}
		return collector.planV0(), nil
	}
	if len(normalized.WorkCandidates) == 0 {
		if schedulerHasInFlightAgentsV0(normalized.Snapshot) {
			return schedulerWaitingPlanV0(normalized, SchedulerWaitingAgentDeliveryPendingV0), nil
		}
		return schedulerQuiescentPlanV0(normalized), nil
	}
	for _, candidate := range normalized.WorkCandidates {
		if err := collector.collectCandidateV0(candidate); err != nil {
			return DirectorSchedulerTickPlanV0{}, err
		}
	}
	collector.addInFlightAgentWaitIfAnyV0()
	return collector.planV0(), nil
}
