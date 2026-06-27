package orquestamcp

import "strings"

const (
	mcpAutoprogrammingActionStaleRunningV0                  = "stale_running"
	mcpAutoprogrammingActionRunningStaleNoProcessV0         = "running_stale_no_process"
	mcpAutoprogrammingActionRunningWithoutRecentStatsV0     = "running_without_recent_stats"
	mcpAutoprogrammingActionStaleRunningReconciledV0        = "stale_running_reconciled"
	mcpAutoprogrammingActionProviderUsageLimitRetryV0       = "provider_usage_limit_retry_after"
	mcpAutoprogrammingActionExternalWorkStoppedNoDeliveryV0 = "external_work_accepted_stopped_without_delivery"
	mcpAutoprogrammingEvidenceRunningStaleReconciledV0      = "evidence-ref-run-queue-running-stale-no-live-process-reconciled"
	mcpAutoprogrammingEvidenceProviderUsageLimitRetryV0     = "evidence-ref-provider-usage-limit-retry-after"
	mcpAutoprogrammingEvidenceExternalWorkRunStartedV0      = "evidence-ref-external-work-run-started"
	mcpAutoprogrammingEvidenceExternalWorkRunQueuedV0       = "evidence-ref-external-work-run-queued"
	mcpAutoprogrammingEvidenceRunCoordinatorExecutedV0      = "evidence-ref-run-coordinator-executed"
)

func buildMCPAutoprogrammingStaleRunningV0(
	queue *MCPRunQueuePriorityToolResultV0,
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
		action, ok := mcpAutoprogrammingStaleRunningActionForCandidateV0(candidate, observedByRunRef)
		if ok {
			out = append(out, action)
		}
	}
	return out
}

func mcpAutoprogrammingStaleRunningActionForCandidateV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) (MCPAutoprogrammingActionableRunV0, bool) {
	status := strings.ToLower(strings.TrimSpace(candidate.Status))
	runRef := strings.TrimSpace(candidate.RunRef)
	if runRef == "" {
		return MCPAutoprogrammingActionableRunV0{}, false
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
	if status == "running" {
		observed := observedByRunRef[runRef]
		if mcpAutoprogrammingRunStatsLiveV0(observed) {
			return MCPAutoprogrammingActionableRunV0{}, false
		}
		if observed != nil && observed.Stats != nil {
			if mcpAutoprogrammingRunStatsHasUnknownProcessSignalV0(*observed.Stats) {
				return mcpAutoprogrammingActionableRunFromCandidateV0(
					candidate,
					mcpAutoprogrammingActionRunningWithoutRecentStatsV0,
					"info",
					"queue_candidate_running_requires_liveness_confirmation",
					"observe_run_ref_with_process_refs_before_reconcile",
				), true
			}
			if !mcpAutoprogrammingRunStatsHasConfirmedNoLiveProcessV0(*observed.Stats) {
				return mcpAutoprogrammingActionableRunFromCandidateV0(
					candidate,
					mcpAutoprogrammingActionRunningWithoutRecentStatsV0,
					"info",
					"queue_candidate_running_requires_liveness_confirmation",
					"observe_run_ref_with_process_refs_before_reconcile",
				), true
			}
			return mcpAutoprogrammingActionableRunFromCandidateV0(
				candidate,
				mcpAutoprogrammingActionRunningStaleNoProcessV0,
				"warning",
				"running_stale_no_live_process_detected",
				"reconcile_if_no_live_process_or_wait_for_late_ack",
			), true
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

func mcpAutoprogrammingStoppedExternalWorkAcceptedWithoutDeliveryV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil || observed.Stats == nil {
		return false
	}
	if !mcpAutoprogrammingCandidateHasEvidenceV0(candidate, mcpAutoprogrammingEvidenceRunCoordinatorExecutedV0) {
		return false
	}
	if !mcpAutoprogrammingCandidateHasEvidenceV0(candidate, mcpAutoprogrammingEvidenceExternalWorkRunStartedV0) &&
		!mcpAutoprogrammingCandidateHasEvidenceV0(candidate, mcpAutoprogrammingEvidenceExternalWorkRunQueuedV0) {
		return false
	}
	counts := observed.Stats.Counts
	return counts.AgentsRequested == 0 &&
		counts.AgentsStarted == 0 &&
		counts.AgentsInFlight == 0 &&
		counts.Deliveries == 0 &&
		counts.AgentsDelivered == 0
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
	if run == nil || run.Stats == nil {
		return false
	}
	return mcpAutoprogrammingRunStatsHasLiveSignalV0(*run.Stats)
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
