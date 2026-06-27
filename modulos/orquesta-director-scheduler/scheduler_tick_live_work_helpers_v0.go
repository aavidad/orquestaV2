package orquestadirectorscheduler

import orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"

func activeStartedAgentSetV0(snapshot RunSchedulingSnapshotV0) map[string]bool {
	failed := schedulerStringSetV0(snapshot.FailedAgents)
	stopped := schedulerStringSetV0(snapshot.StoppedAgents)
	active := map[string]bool{}
	for _, agentRef := range compactSchedulerStringsV0(snapshot.StartedAgents) {
		if failed[agentRef] || stopped[agentRef] {
			continue
		}
		active[agentRef] = true
	}
	return active
}

func liveClaimRefsWithDependencyV0(
	candidateClaims []orquestacoreconcurrency.WorksetClaimV0,
	liveClaims []orquestacoreconcurrency.WorksetClaimV0,
) []string {
	liveByRef := liveClaimSetV0(liveClaims)
	var refs []string
	for _, claim := range candidateClaims {
		for _, dependencyRef := range claim.DependsOn {
			if liveByRef[dependencyRef] {
				refs = append(refs, dependencyRef)
			}
		}
	}
	return compactSchedulerStringsV0(refs)
}

func conflictTouchesSubjectAndLiveV0(
	conflict orquestacoreconcurrency.WorksetConflictV0,
	subjects map[string]bool,
	liveClaims []orquestacoreconcurrency.WorksetClaimV0,
) bool {
	liveByRef := liveClaimSetV0(liveClaims)
	hasSubject := false
	hasLive := false
	for _, claimRef := range conflict.ClaimRefs {
		if subjects[claimRef] {
			hasSubject = true
		}
		if liveByRef[claimRef] {
			hasLive = true
		}
	}
	return hasSubject && hasLive
}

func liveClaimRefsInConflictV0(
	conflict orquestacoreconcurrency.WorksetConflictV0,
	liveClaims []orquestacoreconcurrency.WorksetClaimV0,
) []string {
	liveByRef := liveClaimSetV0(liveClaims)
	var refs []string
	for _, claimRef := range conflict.ClaimRefs {
		if liveByRef[claimRef] {
			refs = append(refs, claimRef)
		}
	}
	return compactSchedulerStringsV0(refs)
}

func liveAgentRefsForClaimRefsV0(
	claims []orquestacoreconcurrency.WorksetClaimV0,
	claimRefs []string,
) []string {
	wanted := schedulerStringSetV0(claimRefs)
	var refs []string
	for _, claim := range claims {
		if wanted[claim.ClaimRef] {
			refs = append(refs, claim.AgentRequestID)
		}
	}
	return compactSchedulerStringsV0(refs)
}

func liveClaimSetV0(claims []orquestacoreconcurrency.WorksetClaimV0) map[string]bool {
	set := map[string]bool{}
	for _, claim := range claims {
		set[claim.ClaimRef] = true
	}
	return set
}

func liveWorkPolicyEvidenceRefsV0(candidate SchedulableWorkCandidateV0) []string {
	refs := append([]string{}, candidate.EvidenceRefs...)
	if candidate.LiveWorkPolicy != nil {
		refs = append(refs, candidate.LiveWorkPolicy.EvidenceRefs...)
	}
	return compactSchedulerStringsV0(refs)
}

func compactSchedulerWorkSequenceDecisionsV0(
	decisions []SchedulerWorkSequenceDecisionV0,
) []SchedulerWorkSequenceDecisionV0 {
	if len(decisions) == 0 {
		return nil
	}
	compact := make([]SchedulerWorkSequenceDecisionV0, 0, len(decisions))
	seen := map[string]bool{}
	for _, decision := range decisions {
		key := decision.CandidateRef + "|" + string(decision.Action)
		if seen[key] {
			continue
		}
		seen[key] = true
		decision.SubjectClaimRefs = compactSchedulerStringsV0(decision.SubjectClaimRefs)
		decision.LiveClaimRefs = compactSchedulerStringsV0(decision.LiveClaimRefs)
		decision.LiveAgentRefs = compactSchedulerStringsV0(decision.LiveAgentRefs)
		decision.ConflictRefs = compactSchedulerStringsV0(decision.ConflictRefs)
		decision.EvidenceRefs = compactSchedulerStringsV0(decision.EvidenceRefs)
		compact = append(compact, decision)
	}
	return compact
}
