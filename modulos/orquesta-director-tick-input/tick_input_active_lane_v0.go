package orquestadirectortickinput

import orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"

func compactTickInputForActiveLaneV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	if len(input.Snapshot.PendingOutboxRefs) > 0 {
		return compactTickInputWaitingOutboxLaneV0(input)
	}
	switch {
	case len(input.LeaseActionCandidates) > 0:
		return compactTickInputLeaseLaneV0(input)
	case len(input.PhaseArtifactCandidates) > 0:
		return compactTickInputPhaseArtifactLaneV0(input)
	case len(input.DeliveryCandidates) > 0:
		return compactTickInputDeliveryLaneV0(input)
	case len(input.ReviewGateCandidates) > 0:
		return compactTickInputReviewGateLaneV0(input)
	case len(input.ProgressSupervisionCandidates) > 0:
		return compactTickInputProgressLaneV0(input)
	case len(input.ReplanFollowupCandidates) > 0:
		return compactTickInputReplanLaneV0(input)
	default:
		return input
	}
}

func compactTickInputWaitingOutboxLaneV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	input.LeaseActionCandidates = nil
	input.PhaseArtifactCandidates = nil
	input.DeliveryCandidates = nil
	input.ReviewGateCandidates = nil
	input.ProgressSupervisionCandidates = nil
	input.ReplanFollowupCandidates = nil
	input.WorkCandidates = nil
	input.WorkClaims = nil
	return input
}

func compactTickInputLeaseLaneV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	input.PhaseArtifactCandidates = nil
	input.DeliveryCandidates = nil
	input.ReviewGateCandidates = nil
	input.ProgressSupervisionCandidates = nil
	input.ReplanFollowupCandidates = nil
	input.WorkCandidates = nil
	input.WorkClaims = nil
	return input
}

func compactTickInputPhaseArtifactLaneV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	input.DeliveryCandidates = nil
	input.ReviewGateCandidates = nil
	input.ProgressSupervisionCandidates = nil
	input.ReplanFollowupCandidates = nil
	input.WorkCandidates = nil
	input.WorkClaims = nil
	return compactTickInputSnapshotForPhaseArtifactsV0(input)
}

func compactTickInputDeliveryLaneV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	input.ReviewGateCandidates = nil
	input.ProgressSupervisionCandidates = nil
	input.ReplanFollowupCandidates = nil
	input.WorkCandidates = nil
	input.WorkClaims = nil
	return compactTickInputSnapshotForDeliveriesV0(input)
}

func compactTickInputReviewGateLaneV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	input.ProgressSupervisionCandidates = nil
	input.ReplanFollowupCandidates = nil
	input.WorkCandidates = nil
	input.WorkClaims = nil
	return compactTickInputSnapshotForReviewGateV0(input)
}

func compactTickInputProgressLaneV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	input.ReplanFollowupCandidates = nil
	input.WorkCandidates = nil
	input.WorkClaims = nil
	return input
}

func compactTickInputReplanLaneV0(
	input orquestadirectorscheduler.DirectorSchedulerTickInputV0,
) orquestadirectorscheduler.DirectorSchedulerTickInputV0 {
	input.WorkCandidates = nil
	input.WorkClaims = nil
	return input
}
