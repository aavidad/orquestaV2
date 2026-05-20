package orquestadirectortickinput

import (
	"errors"
	"strings"

	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func BuildDirectorSchedulerTickInputV0(
	request DirectorTickInputBuildRequestV0,
) (orquestadirectorscheduler.DirectorSchedulerTickInputV0, error) {
	request.TickRef = strings.TrimSpace(request.TickRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.PendingOutboxRefs = compactTickInputRefsV0(request.PendingOutboxRefs)
	request.EvidenceRefs = compactTickInputRefsV0(request.EvidenceRefs)
	if err := validateDirectorTickInputBuildRequestV0(request); err != nil {
		return orquestadirectorscheduler.DirectorSchedulerTickInputV0{}, err
	}
	output := orquestadirectorscheduler.DirectorSchedulerTickInputV0{
		TickRef:                       request.TickRef,
		RunRef:                        strings.TrimSpace(request.Run.RunID),
		OccurredAt:                    request.OccurredAt,
		Snapshot:                      buildRunSchedulingSnapshotV0(request),
		LeaseActionCandidates:         cloneLeaseCandidatesV0(request.LeaseActionCandidates),
		PhaseArtifactCandidates:       clonePhaseArtifactCandidatesV0(request.PhaseArtifactCandidates),
		DeliveryCandidates:            cloneDeliveryCandidatesV0(request.DeliveryCandidates),
		ReviewGateCandidates:          cloneReviewGateCandidatesV0(request.ReviewGateCandidates),
		ProgressSupervisionCandidates: cloneProgressCandidatesV0(request.ProgressSupervisionCandidates),
		ReplanFollowupCandidates:      cloneReplanCandidatesV0(request.ReplanFollowupCandidates),
		WorkClaims:                    cloneWorkClaimsV0(request.WorkClaims),
		WorkCandidates:                cloneWorkCandidatesV0(request.WorkCandidates),
		EvidenceRefs:                  request.EvidenceRefs,
	}
	output.ProgressSupervisionCandidates = tickInputProgressCandidatesForRunV0(
		output.ProgressSupervisionCandidates,
		output.RunRef,
	)
	output = compactTickInputForActiveLaneV0(output)
	normalized := orquestadirectorscheduler.NormalizeDirectorSchedulerTickInputV0(output)
	normalized = compactTickInputPayloadV0(normalized)
	if err := orquestadirectorscheduler.ValidateDirectorSchedulerTickInputV0(normalized); err != nil {
		return orquestadirectorscheduler.DirectorSchedulerTickInputV0{}, tickInputSchedulerErrorV0(err)
	}
	return normalized, nil
}

func tickInputSchedulerErrorV0(err error) DirectorTickInputBuildErrorV0 {
	if field := tickInputSchedulerErrorFieldV0(err); field != "" {
		return tickInputErrorV0("scheduler_input." + field)
	}
	return tickInputErrorV0("scheduler_input")
}

func tickInputSchedulerErrorFieldV0(err error) string {
	var schedulerErr orquestadirectorscheduler.DirectorSchedulerTickErrorV0
	if errors.As(err, &schedulerErr) {
		return strings.TrimSpace(schedulerErr.Field)
	}
	return ""
}

func buildRunSchedulingSnapshotV0(
	request DirectorTickInputBuildRequestV0,
) orquestadirectorscheduler.RunSchedulingSnapshotV0 {
	run := request.Run
	return orquestadirectorscheduler.RunSchedulingSnapshotV0{
		RunRef:                    strings.TrimSpace(run.RunID),
		CurrentPhaseID:            strings.TrimSpace(string(run.CurrentPhase)),
		Tasks:                     compactTickInputRefsV0(run.Tasks),
		CapacityRequests:          compactTickInputRefsV0(run.CapacityRequests),
		CapacityDecisions:         schedulerCapacityDecisionRefsV0(run.CapacityDecisions),
		ConcurrencyGates:          schedulerConcurrencyGateRefsV0(run.ConcurrencyGates),
		Agents:                    compactTickInputRefsV0(run.Agents),
		StartedAgents:             compactTickInputRefsV0(run.StartedAgents),
		FailedAgents:              compactTickInputRefsV0(run.FailedAgents),
		StoppedAgents:             compactTickInputRefsV0(run.StoppedAgents),
		PhaseArtifacts:            schedulerPhaseArtifactRefsV0(run.PhaseArtifacts),
		Deliveries:                compactTickInputRefsV0(run.Deliveries),
		Reviews:                   compactTickInputRefsV0(run.Reviews),
		ReviewResults:             compactTickInputRefsV0(run.ReviewResults),
		AcceptedReviews:           compactTickInputRefsV0(run.AcceptedReviews),
		ReworkRequests:            compactTickInputRefsV0(run.ReworkRequests),
		AgentAssessments:          compactTickInputRefsV0(run.AgentAssessments),
		DirectorQuestions:         compactTickInputRefsV0(run.DirectorQuestions),
		DirectorAnsweredQuestions: compactTickInputRefsV0(run.DirectorAnsweredQuestions),
		ExpiredLeaseRefs:          schedulerExpiredLeaseRefsV0(run.AgentLeaseExpirations),
		ReplanRefs:                schedulerReplanRefsV0(run.ReplanDecisions),
		BlockingQualityGateRefs:   schedulerBlockingQualityGateRefsV0(run.QualityGates),
		PendingOutboxRefs:         compactTickInputRefsV0(request.PendingOutboxRefs),
	}
}
