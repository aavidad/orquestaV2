package orquestamcp

import (
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

const (
	mcpAutoprogrammingActionStaleRunningV0                    = "stale_running"
	mcpAutoprogrammingActionRunningStaleNoProcessV0           = "running_stale_no_process"
	mcpAutoprogrammingActionRunningWithoutRecentStatsV0       = "running_without_recent_stats"
	mcpAutoprogrammingActionStaleRunningReconciledV0          = "stale_running_reconciled"
	mcpAutoprogrammingActionProviderUsageLimitRetryV0         = "provider_usage_limit_retry_after"
	mcpAutoprogrammingActionExternalWorkStoppedNoDeliveryV0   = "external_work_accepted_stopped_without_delivery"
	mcpAutoprogrammingActionExternalWorkNoAgentMaterializedV0 = "external_work_accepted_no_agent_materialized"
	mcpAutoprogrammingActionGoalFirstStateMissingV0           = "goal_first_state_missing"
	mcpAutoprogrammingActionGoalFirstBlockedV0                = "goal_first_blocked"
	mcpAutoprogrammingEvidenceRunningStaleReconciledV0        = "evidence-ref-run-queue-running-stale-no-live-process-reconciled"
	mcpAutoprogrammingEvidenceProviderUsageLimitRetryV0       = "evidence-ref-provider-usage-limit-retry-after"
	mcpAutoprogrammingEvidenceExternalWorkRunStartedV0        = "evidence-ref-external-work-run-started"
	mcpAutoprogrammingEvidenceExternalWorkRunQueuedV0         = "evidence-ref-external-work-run-queued"
	mcpAutoprogrammingEvidenceRunCoordinatorExecutedV0        = "evidence-ref-run-coordinator-executed"
	mcpAutoprogrammingEvidenceGoalFirstStateMissingV0         = "evidence-ref-autoprogramming-status-goal-first-state-missing"
	mcpAutoprogrammingEvidenceGoalFirstBlockedV0              = "evidence-ref-autoprogramming-status-goal-first-blocked"
)

func buildMCPAutoprogrammingStaleRunningV0(
	queue *MCPRunQueuePriorityToolResultV0,
	goalStatesByRunRef map[string]orquestagoal.GoalWorkStateV0,
	goalRunMarkersByRunRef map[string]orquestagoal.GoalWorkRunMarkerV0,
	run *MCPDirectorStatsToolResultV0,
	observedRuns ...*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingActionableRunV0 {
	if queue == nil {
		return nil
	}
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	observedByRunRef := mcpAutoprogrammingObservedRunsByRefV0(run, observedRuns...)
	for _, candidate := range append(
		append([]MCPRunQueueRankedCandidateCompactV0{}, queue.Ranked...),
		queue.Terminal...,
	) {
		action, ok := mcpAutoprogrammingStaleRunningActionForCandidateV0(candidate, goalStatesByRunRef, goalRunMarkersByRunRef, observedByRunRef)
		if ok {
			out = append(out, action)
		}
	}
	return out
}

func mcpAutoprogrammingGoalFirstBlockedActionsV0(
	states []orquestagoal.GoalWorkStateV0,
) []MCPAutoprogrammingActionableRunV0 {
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	for _, state := range states {
		if !mcpAutoprogrammingGoalStateNeedsAttentionV0(state) ||
			mcpAutoprogrammingGoalStateObserveRequiredV0(state) {
			continue
		}
		metadata := mcpGoalWorkStateDomainOperationalMetadataV0(state)
		recommendedAction := "review_replan_goal_first"
		if metadata.RetryFromPhase != "" {
			recommendedAction = "retry_from_phase"
		}
		if metadata.CloseSupersededByLocalEvidence {
			recommendedAction = "close_superseded_by_local_evidence"
		}
		out = append(out, MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionGoalFirstBlockedV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(state.RunRef),
			Status:            strings.TrimSpace(state.Status),
			Reason:            "goal-first state persisted as blocked or invalid",
			RecommendedAction: recommendedAction,
			CurrentPhase:      metadata.CurrentPhase,
			DomainCounters:    metadata.DomainCounters,
			EvidenceRefs: compactStringsMCPV0(append(
				[]string{mcpAutoprogrammingEvidenceGoalFirstBlockedV0},
				state.EvidenceRefs...,
			)),
		})
	}
	return out
}

func mcpAutoprogrammingStaleRunningActionForCandidateV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
	goalStatesByRunRef map[string]orquestagoal.GoalWorkStateV0,
	goalRunMarkersByRunRef map[string]orquestagoal.GoalWorkRunMarkerV0,
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) (MCPAutoprogrammingActionableRunV0, bool) {
	status := strings.ToLower(strings.TrimSpace(candidate.Status))
	runRef := strings.TrimSpace(candidate.RunRef)
	if runRef == "" {
		return MCPAutoprogrammingActionableRunV0{}, false
	}
	if _, ok := goalStatesByRunRef[runRef]; ok {
		return MCPAutoprogrammingActionableRunV0{}, false
	}
	if marker, ok := goalRunMarkersByRunRef[runRef]; ok {
		return mcpAutoprogrammingGoalFirstStateMissingActionV0(candidate, marker), true
	}
	if mcpAutoprogrammingCandidateHasEvidenceV0(candidate, mcpAutoprogrammingEvidenceProviderUsageLimitRetryV0) {
		return mcpAutoprogrammingActionableRunFromCandidateV0(
			candidate,
			mcpAutoprogrammingActionProviderUsageLimitRetryV0,
			"blocked",
			firstNonEmptyMCPV0(strings.TrimSpace(candidate.RescueReason), mcpAutoprogrammingActionProviderUsageLimitRetryV0),
			"wait_for_quota_and_relaunch_idempotently",
		), true
	}
	if mcpAutoprogrammingCandidateHasEvidenceV0(candidate, mcpAutoprogrammingEvidenceRunningStaleReconciledV0) {
		return mcpAutoprogrammingActionableRunFromCandidateV0(
			candidate,
			mcpAutoprogrammingActionStaleRunningReconciledV0,
			"info",
			firstNonEmptyMCPV0(strings.TrimSpace(candidate.RescueReason), "queued_running_stale_no_live_process_reconciled"),
			"inspect_run_ref_before_relaunch",
		), true
	}
	if status == "stopped" &&
		mcpAutoprogrammingStoppedExternalWorkAcceptedWithoutDeliveryV0(candidate, observedByRunRef[runRef]) {
		return mcpAutoprogrammingActionableRunFromCandidateV0(
			candidate,
			mcpAutoprogrammingActionExternalWorkStoppedNoDeliveryV0,
			"blocked",
			"external_work accepted and stopped without agents or delivery",
			"relaunch_or_replan_external_work_with_causal_error",
		), true
	}
	if mcpAutoprogrammingTerminalDoneNoAgentStatusV0(status) &&
		mcpAutoprogrammingExternalWorkAcceptedNoAgentMaterializedV0(candidate, observedByRunRef[runRef]) {
		return mcpAutoprogrammingActionableRunFromCandidateV0(
			candidate,
			mcpAutoprogrammingActionExternalWorkNoAgentMaterializedV0,
			"blocked",
			"external_work accepted as terminal without agents or delivery",
			"relaunch_or_replan_external_work_with_causal_error",
		), true
	}
	if status == "running" {
		observed := observedByRunRef[runRef]
		if mcpAutoprogrammingGoalStatsSuppressesStaleRunningV0(observed) {
			return MCPAutoprogrammingActionableRunV0{}, false
		}
		if mcpAutoprogrammingRunStatsLiveV0(observed) {
			return MCPAutoprogrammingActionableRunV0{}, false
		}
		if observed != nil && observed.Stats != nil {
			liveness := mcpAutoprogrammingRunLivenessV0(*observed.Stats)
			switch liveness.Class {
			case orquestaruncoordinator.RunLivenessClassRunningWithoutRecentStatsV0:
				return mcpAutoprogrammingActionableRunWithObservedStatsV0(mcpAutoprogrammingActionableRunFromCandidateV0(
					candidate,
					mcpAutoprogrammingActionRunningWithoutRecentStatsV0,
					"info",
					firstNonEmptyMCPV0(liveness.Reason, "queue_candidate_running_requires_liveness_confirmation"),
					firstNonEmptyMCPV0(liveness.RecommendedAction, "observe_run_ref_with_process_refs_before_reconcile"),
				), observed), true
			case orquestaruncoordinator.RunLivenessClassRunningStaleNoProcessV0:
				return mcpAutoprogrammingActionableRunWithObservedStatsV0(mcpAutoprogrammingActionableRunFromCandidateV0(
					candidate,
					mcpAutoprogrammingActionRunningStaleNoProcessV0,
					"warning",
					firstNonEmptyMCPV0(liveness.Reason, "running_stale_no_live_process_detected"),
					firstNonEmptyMCPV0(liveness.RecommendedAction, "reconcile_if_no_live_process_or_wait_for_late_ack"),
				), observed), true
			default:
				return mcpAutoprogrammingActionableRunWithObservedStatsV0(mcpAutoprogrammingActionableRunFromCandidateV0(
					candidate,
					mcpAutoprogrammingActionRunningWithoutRecentStatsV0,
					"info",
					firstNonEmptyMCPV0(liveness.Reason, "queue_candidate_running_requires_liveness_confirmation"),
					firstNonEmptyMCPV0(liveness.RecommendedAction, "observe_run_ref_with_process_refs_before_reconcile"),
				), observed), true
			}
		}
		return mcpAutoprogrammingActionableRunFromCandidateV0(
			candidate,
			mcpAutoprogrammingActionRunningWithoutRecentStatsV0,
			"info",
			"queue_candidate_running_requires_liveness_confirmation",
			"observe_run_ref_with_process_refs_before_reconcile",
		), true
	}
	return MCPAutoprogrammingActionableRunV0{}, false
}

func mcpAutoprogrammingGoalFirstStateMissingActionV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
	marker orquestagoal.GoalWorkRunMarkerV0,
) MCPAutoprogrammingActionableRunV0 {
	return MCPAutoprogrammingActionableRunV0{
		Code:              mcpAutoprogrammingActionGoalFirstStateMissingV0,
		Severity:          "blocked",
		RunRef:            strings.TrimSpace(candidate.RunRef),
		AppRef:            strings.TrimSpace(candidate.AppRef),
		Status:            strings.TrimSpace(candidate.Status),
		Reason:            "goal-first container missing GoalWorkStateV0",
		RecommendedAction: "repair_goal_state_before_legacy_supervision",
		EvidenceRefs: compactStringsMCPV0(append(
			append([]string{mcpAutoprogrammingEvidenceGoalFirstStateMissingV0}, candidate.EvidenceRefs...),
			marker.EvidenceRefs...,
		)),
	}
}

func mcpAutoprogrammingStoppedExternalWorkAcceptedWithoutDeliveryV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil || observed.Stats == nil {
		return false
	}
	if !mcpAutoprogrammingExternalWorkAcceptedCandidateV0(candidate) {
		return false
	}
	counts := observed.Stats.Counts
	return counts.AgentsRequested == 0 &&
		counts.AgentsStarted == 0 &&
		counts.AgentsInFlight == 0 &&
		counts.Deliveries == 0 &&
		counts.AgentsDelivered == 0
}

func mcpAutoprogrammingExternalWorkAcceptedNoAgentMaterializedV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil || observed.Stats == nil {
		return false
	}
	if !mcpAutoprogrammingExternalWorkAcceptedCandidateV0(candidate) {
		return false
	}
	if !mcpAutoprogrammingTerminalDoneNoAgentStatusV0(observed.Stats.Status) &&
		!mcpAutoprogrammingTerminalDoneNoAgentStatusV0(candidate.Status) {
		return false
	}
	counts := observed.Stats.Counts
	return counts.AgentsRequested == 0 &&
		counts.AgentsStarted == 0 &&
		counts.AgentsInFlight == 0 &&
		counts.Deliveries == 0 &&
		counts.AgentsDelivered == 0
}

func mcpAutoprogrammingExternalWorkAcceptedCandidateV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
) bool {
	if !mcpAutoprogrammingCandidateHasEvidenceV0(candidate, mcpAutoprogrammingEvidenceRunCoordinatorExecutedV0) {
		return false
	}
	return mcpAutoprogrammingCandidateHasEvidenceV0(candidate, mcpAutoprogrammingEvidenceExternalWorkRunStartedV0) ||
		mcpAutoprogrammingCandidateHasEvidenceV0(candidate, mcpAutoprogrammingEvidenceExternalWorkRunQueuedV0)
}

func mcpAutoprogrammingTerminalDoneNoAgentStatusV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "done", "completed", "complete", "closed":
		return true
	default:
		return false
	}
}

func mcpAutoprogrammingObservedRunsByRefV0(
	run *MCPDirectorStatsToolResultV0,
	observedRuns ...*MCPDirectorStatsToolResultV0,
) map[string]*MCPDirectorStatsToolResultV0 {
	out := map[string]*MCPDirectorStatsToolResultV0{}
	if run != nil && run.Stats != nil {
		if runRef := strings.TrimSpace(run.Stats.RunRef); runRef != "" {
			out[runRef] = run
		}
	}
	for _, observed := range observedRuns {
		if observed == nil || observed.Stats == nil {
			continue
		}
		if runRef := strings.TrimSpace(observed.Stats.RunRef); runRef != "" {
			out[runRef] = observed
		}
	}
	return out
}

func mcpAutoprogrammingActionableRunFromCandidateV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
	code string,
	severity string,
	reason string,
	recommendedAction string,
) MCPAutoprogrammingActionableRunV0 {
	return MCPAutoprogrammingActionableRunV0{
		Code:              strings.TrimSpace(code),
		Severity:          strings.TrimSpace(severity),
		RunRef:            strings.TrimSpace(candidate.RunRef),
		AppRef:            strings.TrimSpace(candidate.AppRef),
		Status:            strings.TrimSpace(candidate.Status),
		Reason:            strings.TrimSpace(reason),
		RecommendedAction: strings.TrimSpace(recommendedAction),
		EvidenceRefs:      compactStringsMCPV0(candidate.EvidenceRefs),
	}
}

func mcpAutoprogrammingActionableRunWithObservedStatsV0(
	action MCPAutoprogrammingActionableRunV0,
	observed *MCPDirectorStatsToolResultV0,
) MCPAutoprogrammingActionableRunV0 {
	if observed == nil || observed.Stats == nil {
		return action
	}
	stats := *observed.Stats
	liveness := mcpAutoprogrammingRunLivenessV0(stats)
	action.ProcessAliveCount = liveness.AgentsLive
	action.AckDetected, action.LastAckAt = mcpAutoprogrammingStatsAckSummaryV0(stats)
	action.LastOutputAt = mcpAutoprogrammingStatsLastActivityAtV0(stats)
	action.LastArtifactAt = mcpAutoprogrammingStatsLastArtifactAtV0(stats)
	return action
}

func mcpAutoprogrammingStatsAckSummaryV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) (bool, string) {
	lastAckAt := ""
	ackDetected := false
	for _, task := range stats.Progress.Tasks {
		ackDetected = ackDetected || task.SecondsSinceAck > 0 || strings.TrimSpace(task.LastAckAt) != ""
		lastAckAt = maxMCPAutoprogrammingTimestampV0(lastAckAt, task.LastAckAt)
	}
	for _, agent := range stats.Agents {
		if agent.LastProgress == nil {
			continue
		}
		ackDetected = ackDetected ||
			agent.LastProgress.SecondsSinceAck > 0 ||
			strings.TrimSpace(agent.LastProgress.LastAckAt) != ""
		lastAckAt = maxMCPAutoprogrammingTimestampV0(lastAckAt, agent.LastProgress.LastAckAt)
	}
	return ackDetected, lastAckAt
}

func mcpAutoprogrammingStatsLastActivityAtV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) string {
	last := ""
	for _, task := range stats.Progress.Tasks {
		last = maxMCPAutoprogrammingTimestampV0(last, task.LastActivityAt)
	}
	for _, agent := range stats.Agents {
		if agent.LastProgress == nil {
			continue
		}
		last = maxMCPAutoprogrammingTimestampV0(last, agent.LastProgress.LastActivityAt)
	}
	return last
}

func mcpAutoprogrammingStatsLastArtifactAtV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) string {
	last := ""
	for _, task := range stats.Progress.Tasks {
		if strings.TrimSpace(task.DeliveryRef) == "" {
			continue
		}
		last = maxMCPAutoprogrammingTimestampV0(last, firstNonEmptyMCPV0(task.LastActivityAt, task.LastAckAt))
	}
	for _, agent := range stats.Agents {
		if agent.LastProgress == nil || strings.TrimSpace(agent.LastProgress.DeliveryRef) == "" {
			continue
		}
		last = maxMCPAutoprogrammingTimestampV0(
			last,
			firstNonEmptyMCPV0(agent.LastProgress.LastActivityAt, agent.LastProgress.LastAckAt),
		)
	}
	return last
}

func maxMCPAutoprogrammingTimestampV0(
	current string,
	candidate string,
) string {
	current = strings.TrimSpace(current)
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return current
	}
	if current == "" || candidate > current {
		return candidate
	}
	return current
}

func mcpAutoprogrammingCandidateHasEvidenceV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
	evidenceRef string,
) bool {
	evidenceRef = strings.TrimSpace(evidenceRef)
	for _, value := range candidate.EvidenceRefs {
		if strings.TrimSpace(value) == evidenceRef {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingRunStatsLiveV0(run *MCPDirectorStatsToolResultV0) bool {
	if run == nil {
		return false
	}
	if mcpAutoprogrammingGoalStatsSuppressesStaleRunningV0(run) {
		return true
	}
	if run.Stats == nil {
		return false
	}
	return mcpAutoprogrammingRunStatsHasLiveSignalV0(*run.Stats)
}

func mcpAutoprogrammingGoalStatsSuppressesStaleRunningV0(
	run *MCPDirectorStatsToolResultV0,
) bool {
	return run != nil && run.Goal != nil && strings.TrimSpace(run.Goal.GoalRef) != ""
}

func diagnosticsFromStaleRunningMCPAutoprogrammingV0(
	values []MCPAutoprogrammingActionableRunV0,
) []MCPAutoprogrammingDiagnosticV0 {
	out := make([]MCPAutoprogrammingDiagnosticV0, 0, len(values))
	for _, value := range values {
		out = append(out, MCPAutoprogrammingDiagnosticV0{
			Code:         strings.TrimSpace(value.Code),
			Scope:        "run:" + strings.TrimSpace(value.RunRef),
			Message:      strings.TrimSpace(value.Reason),
			EvidenceRefs: compactStringsMCPV0(value.EvidenceRefs),
		})
	}
	return out
}
