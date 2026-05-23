package orquestadirectorscheduler

import (
	"strconv"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type schedulerLiveWorkOverlapV0 struct {
	liveClaimRefs     []string
	liveAgentRefs     []string
	conflictRefs      []string
	hasDependencyWait bool
}

func (collector *schedulerTickCollectorV0) collectWorkCandidateWithLivePolicyV0(
	candidate SchedulableWorkCandidateV0,
) error {
	overlap := collector.liveWorkOverlapForCandidateV0(candidate)
	if len(overlap.liveClaimRefs) == 0 {
		collector.addWorkSequenceDecisionV0(newWorkSequenceDecisionV0(
			candidate,
			SchedulerWorkSequenceActionExecuteNowV0,
			overlap,
		))
		return collector.collectCandidateV0(candidate)
	}
	action := liveWorkActionForCandidateV0(candidate, overlap)
	collector.addWorkSequenceDecisionV0(newWorkSequenceDecisionV0(candidate, action, overlap))

	switch action {
	case SchedulerWorkSequenceActionCreateReviewTaskV0:
		return collector.collectLiveWorkMicrotaskV0(candidate.LiveWorkPolicy.ReviewTaskCandidate)
	case SchedulerWorkSequenceActionCreateStudyTaskV0:
		return collector.collectLiveWorkMicrotaskV0(candidate.LiveWorkPolicy.StudyTaskCandidate)
	default:
		collector.addWaitingV0(SchedulerWaitingLiveWorkOverlapV0)
		return nil
	}
}

func (collector *schedulerTickCollectorV0) collectLiveWorkMicrotaskV0(
	candidate *SchedulerCreateMicrotaskCandidateV0,
) error {
	if candidate == nil {
		collector.addNeedsDirectorV0(SchedulerWaitingCandidateMissingV0)
		return nil
	}
	command, err := orquestacoreworkflow.NewCreateMicrotaskCommandV0(
		candidate.CommandMeta,
		candidate.Payload,
	)
	if err != nil {
		return err
	}
	taskRef := candidate.Payload.Task.TaskID
	if collector.tasks[taskRef] || collector.plannedTasks[taskRef] {
		return nil
	}
	collector.addReadyCommandV0(command)
	collector.plannedTasks[taskRef] = true
	return nil
}

func liveWorkActionForCandidateV0(
	candidate SchedulableWorkCandidateV0,
	overlap schedulerLiveWorkOverlapV0,
) SchedulerWorkSequenceActionV0 {
	if candidate.LiveWorkPolicy != nil && candidate.LiveWorkPolicy.OnOverlap != "" {
		return candidate.LiveWorkPolicy.OnOverlap
	}
	if overlap.hasDependencyWait {
		return SchedulerWorkSequenceActionQueueAfterLiveWorkV0
	}
	return SchedulerWorkSequenceActionWaitLiveWorkV0
}

func (collector *schedulerTickCollectorV0) liveWorkOverlapForCandidateV0(
	candidate SchedulableWorkCandidateV0,
) schedulerLiveWorkOverlapV0 {
	subjects := schedulerStringSetV0(candidate.SubjectClaimRefs)
	liveClaims := collector.liveWorkClaimsV0(subjects)
	candidateClaims := collector.candidateSubjectClaimsV0(candidate, subjects)
	overlap := schedulerLiveWorkOverlapV0{
		liveClaimRefs: liveClaimRefsWithDependencyV0(candidateClaims, liveClaims),
	}
	overlap.hasDependencyWait = len(overlap.liveClaimRefs) > 0

	conflicts := orquestacoreconcurrency.DetectWorksetConflictsV0(
		append(append([]orquestacoreconcurrency.WorksetClaimV0{}, candidateClaims...), liveClaims...),
	)
	for _, conflict := range conflicts {
		if !conflictTouchesSubjectAndLiveV0(conflict, subjects, liveClaims) {
			continue
		}
		overlap.conflictRefs = append(overlap.conflictRefs, conflict.ConflictRef)
		overlap.liveClaimRefs = append(overlap.liveClaimRefs, liveClaimRefsInConflictV0(conflict, liveClaims)...)
	}
	overlap.liveClaimRefs = compactSchedulerStringsV0(overlap.liveClaimRefs)
	if len(overlap.liveClaimRefs) == 0 {
		overlap.liveAgentRefs = nil
	}
	overlap.conflictRefs = compactSchedulerStringsV0(overlap.conflictRefs)
	overlap.liveAgentRefs = liveAgentRefsForClaimRefsV0(liveClaims, overlap.liveClaimRefs)
	return overlap
}

func (collector *schedulerTickCollectorV0) liveWorkClaimsV0(
	subjects map[string]bool,
) []orquestacoreconcurrency.WorksetClaimV0 {
	activeAgents := activeStartedAgentSetV0(collector.input.Snapshot)
	claims := collector.input.WorkClaims
	if len(claims) == 0 {
		claims = collector.claimsFromWorkCandidatesV0()
	}
	live := make([]orquestacoreconcurrency.WorksetClaimV0, 0)
	for _, claim := range claims {
		if subjects[claim.ClaimRef] || !activeAgents[claim.AgentRequestID] {
			continue
		}
		live = append(live, claim)
	}
	return live
}

func (collector *schedulerTickCollectorV0) claimsFromWorkCandidatesV0() []orquestacoreconcurrency.WorksetClaimV0 {
	var claims []orquestacoreconcurrency.WorksetClaimV0
	for _, candidate := range collector.input.WorkCandidates {
		claims = append(claims, candidate.Claims...)
	}
	return claims
}

func (collector *schedulerTickCollectorV0) candidateSubjectClaimsV0(
	candidate SchedulableWorkCandidateV0,
	subjects map[string]bool,
) []orquestacoreconcurrency.WorksetClaimV0 {
	claimByRef := map[string]orquestacoreconcurrency.WorksetClaimV0{}
	for _, claim := range append(append([]orquestacoreconcurrency.WorksetClaimV0{}, collector.input.WorkClaims...), candidate.Claims...) {
		claimByRef[claim.ClaimRef] = claim
	}
	claims := make([]orquestacoreconcurrency.WorksetClaimV0, 0, len(subjects))
	for ref := range subjects {
		if claim, ok := claimByRef[ref]; ok {
			claims = append(claims, claim)
		}
	}
	return claims
}

func newWorkSequenceDecisionV0(
	candidate SchedulableWorkCandidateV0,
	action SchedulerWorkSequenceActionV0,
	overlap schedulerLiveWorkOverlapV0,
) SchedulerWorkSequenceDecisionV0 {
	return SchedulerWorkSequenceDecisionV0{
		CandidateRef:     candidate.CandidateRef,
		Action:           action,
		SubjectClaimRefs: candidate.SubjectClaimRefs,
		LiveClaimRefs:    compactSchedulerStringsV0(overlap.liveClaimRefs),
		LiveAgentRefs:    compactSchedulerStringsV0(overlap.liveAgentRefs),
		ConflictRefs:     compactSchedulerStringsV0(overlap.conflictRefs),
		Summary:          workSequenceSummaryV0(action, overlap),
		EvidenceRefs:     liveWorkPolicyEvidenceRefsV0(candidate),
	}
}

func workSequenceSummaryV0(action SchedulerWorkSequenceActionV0, overlap schedulerLiveWorkOverlapV0) string {
	return "work_sequence action=" + string(action) +
		" live_claims=" + strconv.Itoa(len(compactSchedulerStringsV0(overlap.liveClaimRefs))) +
		" conflicts=" + strconv.Itoa(len(compactSchedulerStringsV0(overlap.conflictRefs)))
}
