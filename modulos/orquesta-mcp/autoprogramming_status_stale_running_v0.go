package orquestamcp

import "strings"

const (
	mcpAutoprogrammingActionStaleRunningV0              = "stale_running"
	mcpAutoprogrammingActionStaleRunningReconciledV0    = "stale_running_reconciled"
	mcpAutoprogrammingActionProviderUsageLimitRetryV0   = "provider_usage_limit_retry_after"
	mcpAutoprogrammingEvidenceRunningStaleReconciledV0  = "evidence-ref-run-queue-running-stale-no-live-process-reconciled"
	mcpAutoprogrammingEvidenceProviderUsageLimitRetryV0 = "evidence-ref-provider-usage-limit-retry-after"
)

func buildMCPAutoprogrammingStaleRunningV0(
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingActionableRunV0 {
	if queue == nil {
		return nil
	}
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	liveRunRef := ""
	if mcpAutoprogrammingRunStatsLiveV0(run) {
		liveRunRef = strings.TrimSpace(run.Stats.RunRef)
	}
	for _, candidate := range append(
		append([]MCPRunQueueRankedCandidateCompactV0{}, queue.Ranked...),
		queue.Terminal...,
	) {
		action, ok := mcpAutoprogrammingStaleRunningActionForCandidateV0(candidate, liveRunRef)
		if ok {
			out = append(out, action)
		}
	}
	return out
}

func mcpAutoprogrammingStaleRunningActionForCandidateV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
	liveRunRef string,
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
	if status == "running" && runRef != strings.TrimSpace(liveRunRef) {
		return mcpAutoprogrammingActionableRunFromCandidateV0(
			candidate,
			mcpAutoprogrammingActionStaleRunningV0,
			"warning",
			"queue_candidate_running_requires_liveness_confirmation",
			"supervise_run_ref_or_reconcile_if_no_live_process",
		), true
	}
	return MCPAutoprogrammingActionableRunV0{}, false
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
	if run == nil || run.Stats == nil || run.Stats.Counts.AgentsInFlight <= 0 {
		return false
	}
	return run.Stats.Progress.ProgressingAgents > 0 ||
		hasMCPAutoprogrammingLiveAgentSignalV0(run.Stats.Agents)
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
