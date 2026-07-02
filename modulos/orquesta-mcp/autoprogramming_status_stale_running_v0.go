package orquestamcp

import (
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

const (
	mcpAutoprogrammingActionStaleRunningV0                         = "stale_running"
	mcpAutoprogrammingActionRunningStaleNoProcessV0                = "running_stale_no_process"
	mcpAutoprogrammingActionRunningWithoutRecentStatsV0            = "running_without_recent_stats"
	mcpAutoprogrammingActionActiveNoCheckpointYetV0                = "active_no_checkpoint_yet"
	mcpAutoprogrammingActionActiveTimeoutCheckpointRecentV0        = "active_timeout_checkpoint_recent"
	mcpAutoprogrammingActionNoCheckpointConsumptionWarningV0       = "goal_active_no_checkpoint_consumption_warning"
	mcpAutoprogrammingActionNoCheckpointHighConsumptionV0          = "goal_active_no_checkpoint_high_consumption"
	mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0        = "checkpoint_only_high_consumption"
	mcpAutoprogrammingActionStaleRunningReconciledV0               = "stale_running_reconciled"
	mcpAutoprogrammingActionProviderUsageLimitRetryV0              = "provider_usage_limit_retry_after"
	mcpAutoprogrammingActionExternalWorkStoppedNoDeliveryV0        = "external_work_accepted_stopped_without_delivery"
	mcpAutoprogrammingActionExternalWorkNoAgentMaterializedV0      = "external_work_accepted_no_agent_materialized"
	mcpAutoprogrammingActionGoalFirstStateMissingV0                = "goal_first_state_missing"
	mcpAutoprogrammingActionGoalFirstBlockedV0                     = "goal_first_blocked"
	mcpAutoprogrammingActionMissingTerminalReceiptV0               = MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0
	mcpAutoprogrammingActionArtifactPathsOmittedV0                 = MCPGoalFirstArtifactPathsOmittedMaterializedV0
	mcpAutoprogrammingActionQAFailedPublicTextV0                   = MCPGoalFirstQAFailedPublicTextV0
	mcpAutoprogrammingActionPartialArtifactsWrittenV0              = MCPGoalFirstPartialArtifactsWrittenV0
	mcpAutoprogrammingActionPhase0CompleteNonPublishableV0         = MCPGoalFirstPhase0CompleteNonPublishableV0
	mcpAutoprogrammingActionRequiredTestEvidenceMissingV0          = MCPGoalFirstRequiredTestEvidenceMissingV0
	mcpAutoprogrammingActionGoalActiveTimeoutBackendActiveV0       = "goal_active_timeout_backend_active"
	mcpAutoprogrammingEvidenceRunningStaleReconciledV0             = "evidence-ref-run-queue-running-stale-no-live-process-reconciled"
	mcpAutoprogrammingEvidenceProviderUsageLimitRetryV0            = "evidence-ref-provider-usage-limit-retry-after"
	mcpAutoprogrammingEvidenceExternalWorkRunStartedV0             = "evidence-ref-external-work-run-started"
	mcpAutoprogrammingEvidenceExternalWorkRunQueuedV0              = "evidence-ref-external-work-run-queued"
	mcpAutoprogrammingEvidenceRunCoordinatorExecutedV0             = "evidence-ref-run-coordinator-executed"
	mcpAutoprogrammingEvidenceGoalFirstStateMissingV0              = "evidence-ref-autoprogramming-status-goal-first-state-missing"
	mcpAutoprogrammingEvidenceGoalFirstBlockedV0                   = "evidence-ref-autoprogramming-status-goal-first-blocked"
	mcpAutoprogrammingEvidenceActiveTimeoutCheckpointRecentV0      = "evidence-ref-autoprogramming-active-timeout-checkpoint-recent"
	mcpAutoprogrammingEvidenceNoCheckpointConsumptionWarningV0     = "evidence-ref-autoprogramming-no-checkpoint-consumption-warning"
	mcpAutoprogrammingEvidenceMissingTerminalReceiptV0             = "evidence-ref-autoprogramming-status-missing-terminal-receipt-after-artifacts-pass"
	mcpAutoprogrammingEvidenceArtifactPathsOmittedV0               = "evidence-ref-autoprogramming-status-artifact-paths-omitted-materialized"
	mcpAutoprogrammingEvidenceQAFailedPublicTextV0                 = "evidence-ref-autoprogramming-status-qa-failed-public-text"
	mcpAutoprogrammingEvidencePartialArtifactsWrittenV0            = "evidence-ref-autoprogramming-status-partial-artifacts-written"
	mcpAutoprogrammingEvidencePhase0CompleteNonPublishableV0       = "evidence-ref-autoprogramming-status-phase0-complete-non-publishable"
	mcpAutoprogrammingEvidenceRequiredTestEvidenceMissingV0        = "evidence-ref-autoprogramming-status-required-test-evidence-missing"
	mcpAutoprogrammingEvidenceGoalActiveTimeoutBackendV0           = "evidence-ref-autoprogramming-goal-active-timeout-backend-active"
	mcpAutoprogrammingEvidenceNoCheckpointHighConsumptionV0        = "evidence-ref-autoprogramming-no-checkpoint-high-consumption"
	mcpAutoprogrammingEvidenceCheckpointOnlyHighConsumptionV0      = "evidence-ref-autoprogramming-checkpoint-only-high-consumption"
	mcpAutoprogrammingCheckpointOnlyHighConsumptionTokensDefaultV0 = int64(100000)
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

func mcpAutoprogrammingMissingTerminalReceiptActionsV0(
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingActionableRunV0 {
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	for runRef, observed := range observedByRunRef {
		if !mcpAutoprogrammingObservedMissingTerminalReceiptV0(observed) {
			continue
		}
		action := MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionMissingTerminalReceiptV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(runRef),
			RunStatus:         mcpAutoprogrammingObservedRunStatusV0(observedByRunRef, runRef),
			Reason:            "missing_terminal_receipt_after_artifacts_pass: validated artifacts and QA pass evidence exist, but terminal goal/domain receipt is missing",
			RecommendedAction: MCPGoalFirstRepairReceiptActionV0,
			EvidenceRefs: []string{
				mcpAutoprogrammingEvidenceMissingTerminalReceiptV0,
			},
		}
		action = mcpAutoprogrammingActionableRunWithObservedStatsV0(action, observed)
		action = mcpAutoprogrammingActionableRunWithObservedGoalV0(action, observed)
		out = append(out, action)
	}
	return out
}

func mcpAutoprogrammingQAFailedPublicTextActionsV0(
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingActionableRunV0 {
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	for runRef, observed := range observedByRunRef {
		if !mcpAutoprogrammingObservedQAFailedPublicTextV0(observed) {
			continue
		}
		action := MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionQAFailedPublicTextV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(runRef),
			RunStatus:         mcpAutoprogrammingObservedRunStatusV0(observedByRunRef, runRef),
			Reason:            "qa_failed_public_text: materialized artifacts are recoverable, but structured QA failed public/editorial text",
			RecommendedAction: MCPGoalFirstReworkPublicTextActionV0,
			EvidenceRefs: []string{
				mcpAutoprogrammingEvidenceQAFailedPublicTextV0,
			},
		}
		action = mcpAutoprogrammingActionableRunWithObservedStatsV0(action, observed)
		action = mcpAutoprogrammingActionableRunWithObservedGoalV0(action, observed)
		out = append(out, action)
	}
	return out
}

func mcpAutoprogrammingArtifactPathsOmittedActionsV0(
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingActionableRunV0 {
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	for runRef, observed := range observedByRunRef {
		if !mcpAutoprogrammingObservedArtifactPathsOmittedV0(observed) {
			continue
		}
		action := MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionArtifactPathsOmittedV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(runRef),
			RunStatus:         mcpAutoprogrammingObservedRunStatusV0(observedByRunRef, runRef),
			Reason:            "artifact_paths_omitted_materialized: materialized artifacts exist in the write_set but the terminal receipt omitted their paths",
			RecommendedAction: MCPGoalFirstRepairReceiptActionV0,
			EvidenceRefs: []string{
				mcpAutoprogrammingEvidenceArtifactPathsOmittedV0,
			},
		}
		action = mcpAutoprogrammingActionableRunWithObservedStatsV0(action, observed)
		action = mcpAutoprogrammingActionableRunWithObservedGoalV0(action, observed)
		out = append(out, action)
	}
	return out
}

func mcpAutoprogrammingPartialArtifactsWrittenActionsV0(
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingActionableRunV0 {
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	for runRef, observed := range observedByRunRef {
		if !mcpAutoprogrammingObservedPartialArtifactsWrittenV0(observed) {
			continue
		}
		action := MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionPartialArtifactsWrittenV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(runRef),
			RunStatus:         mcpAutoprogrammingObservedRunStatusV0(observedByRunRef, runRef),
			Reason:            "partial_artifacts_written: materialized artifacts exist in the write_set without a terminal receipt",
			RecommendedAction: MCPGoalFirstReviewPartialArtifactsActionV0,
			EvidenceRefs: []string{
				mcpAutoprogrammingEvidencePartialArtifactsWrittenV0,
			},
		}
		action = mcpAutoprogrammingActionableRunWithObservedStatsV0(action, observed)
		action = mcpAutoprogrammingActionableRunWithObservedGoalV0(action, observed)
		out = append(out, action)
	}
	return out
}

func mcpAutoprogrammingPhase0CompleteNonPublishableActionsV0(
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingActionableRunV0 {
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	for runRef, observed := range observedByRunRef {
		if !mcpAutoprogrammingObservedPhase0CompleteNonPublishableV0(observed) {
			continue
		}
		action := MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionPhase0CompleteNonPublishableV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(runRef),
			RunStatus:         mcpAutoprogrammingObservedRunStatusV0(observedByRunRef, runRef),
			Reason:            "phase0_complete_non_publishable: phase 0 checkpoint delivery exists and is recoverable, but it is not a publishable final closure",
			RecommendedAction: MCPGoalFirstContinueFromPhase0ActionV0,
			EvidenceRefs: []string{
				mcpAutoprogrammingEvidencePhase0CompleteNonPublishableV0,
			},
		}
		action = mcpAutoprogrammingActionableRunWithObservedStatsV0(action, observed)
		action = mcpAutoprogrammingActionableRunWithObservedGoalV0(action, observed)
		out = append(out, action)
	}
	return out
}

func mcpAutoprogrammingRequiredTestEvidenceMissingActionsV0(
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingActionableRunV0 {
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	for runRef, observed := range observedByRunRef {
		if !mcpAutoprogrammingObservedRequiredTestEvidenceMissingV0(observed) {
			continue
		}
		action := MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionRequiredTestEvidenceMissingV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(runRef),
			RunStatus:         mcpAutoprogrammingObservedRunStatusV0(observedByRunRef, runRef),
			Reason:            "required_test_evidence_missing: terminal receipt declares passed required tests without evidence refs",
			RecommendedAction: MCPGoalFirstRepairReceiptActionV0,
			EvidenceRefs: []string{
				mcpAutoprogrammingEvidenceRequiredTestEvidenceMissingV0,
			},
		}
		action = mcpAutoprogrammingActionableRunWithObservedStatsV0(action, observed)
		action = mcpAutoprogrammingActionableRunWithObservedGoalV0(action, observed)
		out = append(out, action)
	}
	return out
}

func mcpAutoprogrammingObservedMissingTerminalReceiptV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil {
		return false
	}
	if observed.Goal != nil && containsStringMCPV0(observed.Goal.IssueCodes, MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0) {
		return true
	}
	if observed.Stats == nil {
		return false
	}
	for _, issue := range observed.Stats.Progress.Issues {
		if strings.TrimSpace(issue.Code) == MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0 {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingObservedArtifactPathsOmittedV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil {
		return false
	}
	if observed.Goal != nil && containsStringMCPV0(observed.Goal.IssueCodes, MCPGoalFirstArtifactPathsOmittedMaterializedV0) {
		return true
	}
	if observed.Stats == nil {
		return false
	}
	for _, issue := range observed.Stats.Progress.Issues {
		if strings.TrimSpace(issue.Code) == MCPGoalFirstArtifactPathsOmittedMaterializedV0 {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingObservedPhase0CompleteNonPublishableV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil {
		return false
	}
	if observed.Goal != nil && containsStringMCPV0(observed.Goal.IssueCodes, MCPGoalFirstPhase0CompleteNonPublishableV0) {
		return true
	}
	if observed.Stats == nil {
		return false
	}
	for _, issue := range observed.Stats.Progress.Issues {
		if strings.TrimSpace(issue.Code) == MCPGoalFirstPhase0CompleteNonPublishableV0 {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingObservedRequiredTestEvidenceMissingV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil {
		return false
	}
	if observed.Goal != nil && containsStringMCPV0(observed.Goal.IssueCodes, MCPGoalFirstRequiredTestEvidenceMissingV0) {
		return true
	}
	if observed.Stats == nil {
		return false
	}
	for _, issue := range observed.Stats.Progress.Issues {
		if strings.TrimSpace(issue.Code) == MCPGoalFirstRequiredTestEvidenceMissingV0 {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingObservedPartialArtifactsWrittenV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil {
		return false
	}
	if observed.Goal != nil && containsStringMCPV0(observed.Goal.IssueCodes, MCPGoalFirstPartialArtifactsWrittenV0) {
		return true
	}
	if observed.Stats == nil {
		return false
	}
	for _, issue := range observed.Stats.Progress.Issues {
		if strings.TrimSpace(issue.Code) == MCPGoalFirstPartialArtifactsWrittenV0 {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingObservedQAFailedPublicTextV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil {
		return false
	}
	if observed.Goal != nil && containsStringMCPV0(observed.Goal.IssueCodes, MCPGoalFirstQAFailedPublicTextV0) {
		return true
	}
	if observed.Stats == nil {
		return false
	}
	for _, issue := range observed.Stats.Progress.Issues {
		if strings.TrimSpace(issue.Code) == MCPGoalFirstQAFailedPublicTextV0 {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingGoalFirstBlockedActionsV0(
	states []orquestagoal.GoalWorkStateV0,
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
	policy MCPAutoprogrammingGoalProgressPolicyV0,
) ([]MCPAutoprogrammingActionableRunV0, []MCPAutoprogrammingActionableRunV0) {
	policy = NormalizeMCPAutoprogrammingGoalProgressPolicyV0(policy)
	blocked := make([]MCPAutoprogrammingActionableRunV0, 0)
	resolved := make([]MCPAutoprogrammingActionableRunV0, 0)
	for _, state := range states {
		if !mcpAutoprogrammingGoalStateNeedsAttentionV0(state) ||
			mcpAutoprogrammingGoalStateObserveRequiredV0(state) {
			continue
		}
		observed := observedByRunRef[strings.TrimSpace(state.RunRef)]
		metadata := mcpGoalWorkStateDomainOperationalMetadataV0(state)
		recommendedAction := "review_replan_goal_first"
		if metadata.RetryFromPhase != "" {
			recommendedAction = "retry_from_phase"
		}
		if metadata.CloseSupersededByLocalEvidence {
			recommendedAction = "close_superseded_by_local_evidence"
		}
		reason := "goal_backend_state_unreconciled: local goal-first state blocked or invalid without reconciled backend snapshot"
		if state.LastClosure != nil && state.LastClosure.Accepted {
			reason = "goal_terminal_result_requires_state_reconcile"
			recommendedAction = "reconcile_goal_terminal"
		}
		if mcpAutoprogrammingGoalResultTerminalV0(state) &&
			!mcpAutoprogrammingObservedGoalTerminalV0(observed) &&
			!(state.LastClosure != nil && state.LastClosure.Accepted) {
			reason = "goal_terminal_reconcile_pending: durable terminal result present but backend/local closure is not reconciled"
			recommendedAction = "reconcile_goal_terminal"
		}
		activeTimeoutBackendActive := mcpAutoprogrammingGoalStateActiveTimeoutV0(state) &&
			mcpAutoprogrammingObservedGoalActiveV0(observed)
		if activeTimeoutBackendActive {
			reason = "goal_active_timeout_backend_active: local goal-first timed out while backend goal remains active; capture safe snapshot and replan before launching more parents"
			recommendedAction = "replan_goal_after_active_timeout"
		}
		activeNoCheckpoint := !mcpAutoprogrammingGoalResultTerminalV0(state) &&
			!activeTimeoutBackendActive &&
			mcpAutoprogrammingObservedGoalActiveWithRecentActivityV0(observed)
		if activeNoCheckpoint {
			reason = "goal_backend_active_no_checkpoint_yet: backend goal is active with recent activity; wait for checkpoint/artifact before declaring stale"
			recommendedAction = "observe_goal_backend_wait_for_checkpoint"
		}
		action := MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionGoalFirstBlockedV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(state.RunRef),
			Status:            strings.TrimSpace(state.Status),
			RunStatus:         mcpAutoprogrammingObservedRunStatusV0(observedByRunRef, state.RunRef),
			GoalRef:           strings.TrimSpace(state.GoalRef),
			ExternalGoalRef:   strings.TrimSpace(state.ExternalGoalRef),
			GoalStatus:        strings.TrimSpace(state.Status),
			Reason:            reason,
			RecommendedAction: recommendedAction,
			CurrentPhase:      metadata.CurrentPhase,
			DomainCounters:    metadata.DomainCounters,
			EvidenceRefs: compactStringsMCPV0(append(
				[]string{mcpAutoprogrammingEvidenceGoalFirstBlockedV0},
				state.EvidenceRefs...,
			)),
		}
		action = mcpAutoprogrammingActionableRunWithGoalStateSnapshotV0(action, state)
		action = mcpAutoprogrammingActionableRunWithObservedStatsV0(
			action,
			observed,
		)
		action = mcpAutoprogrammingActionableRunWithObservedGoalV0(action, observed)
		activeTimeoutCheckpointRecent := activeTimeoutBackendActive &&
			mcpAutoprogrammingActiveTimeoutCheckpointRecentV0(action, observed, policy)
		noCheckpointConsumptionWarning := !activeTimeoutBackendActive &&
			mcpAutoprogrammingNoCheckpointConsumptionWarningV0(action, observed, policy)
		checkpointOnlyHighConsumption := mcpAutoprogrammingCheckpointOnlyHighConsumptionV0(action, observed, policy)
		noCheckpointHighConsumption := !activeTimeoutBackendActive &&
			mcpAutoprogrammingNoCheckpointHighConsumptionV0(action, observed, policy)
		if activeNoCheckpoint {
			action.Code = mcpAutoprogrammingActionActiveNoCheckpointYetV0
			action.Severity = "info"
		}
		if noCheckpointConsumptionWarning {
			action.Code = mcpAutoprogrammingActionNoCheckpointConsumptionWarningV0
			action.Severity = "warning"
			action.Reason = "goal_active_no_checkpoint_consumption_warning: backend goal is active with rising token usage but no checkpoint, artifacts or domain receipt yet; require a checkpoint soon before high-consumption replan"
			action.RecommendedAction = "observe_goal_backend_require_checkpoint"
			action.EvidenceRefs = compactStringsMCPV0(append(
				action.EvidenceRefs,
				mcpAutoprogrammingEvidenceNoCheckpointConsumptionWarningV0,
			))
		}
		if noCheckpointHighConsumption {
			action.Code = mcpAutoprogrammingActionNoCheckpointHighConsumptionV0
			action.Severity = "blocked"
			action.Reason = "goal_active_no_checkpoint_high_consumption: backend goal remains active after high token usage without checkpoint, artifacts or domain receipt; stop safely or replan with narrower context before relaunch"
			action.RecommendedAction = "replan_narrow_context"
			action.EvidenceRefs = compactStringsMCPV0(append(
				action.EvidenceRefs,
				mcpAutoprogrammingEvidenceNoCheckpointHighConsumptionV0,
			))
		}
		if checkpointOnlyHighConsumption {
			action.Code = mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0
			action.Severity = "blocked"
			action.Reason = "checkpoint_only_high_consumption: backend goal remains active after high token usage with only checkpoint artifacts and no domain receipt; narrow context or stop safely before relaunch"
			action.RecommendedAction = "replan_narrow_context"
			action.EvidenceRefs = compactStringsMCPV0(append(
				action.EvidenceRefs,
				mcpAutoprogrammingEvidenceCheckpointOnlyHighConsumptionV0,
			))
		}
		if activeTimeoutCheckpointRecent && !checkpointOnlyHighConsumption {
			action.Code = mcpAutoprogrammingActionActiveTimeoutCheckpointRecentV0
			action.Severity = "info"
			action.Reason = "active_timeout_checkpoint_recent: local timeout fired after a checkpoint while backend goal remains active below high-consumption threshold; observe for next artifact before replan"
			action.RecommendedAction = "observe_goal_backend_wait_for_checkpoint"
			action.EvidenceRefs = compactStringsMCPV0(append(
				action.EvidenceRefs,
				mcpAutoprogrammingEvidenceActiveTimeoutCheckpointRecentV0,
			))
		}
		if activeTimeoutBackendActive && !activeTimeoutCheckpointRecent && !checkpointOnlyHighConsumption {
			action.Code = mcpAutoprogrammingActionGoalActiveTimeoutBackendActiveV0
			action.EvidenceRefs = compactStringsMCPV0(append(
				action.EvidenceRefs,
				mcpAutoprogrammingEvidenceGoalActiveTimeoutBackendV0,
			))
		}
		if mcpAutoprogrammingObservedGoalTerminalV0(observed) {
			action.Code = "goal_first_terminal_reconciled"
			action.Severity = "info"
			action.Reason = "goal_backend_terminal_reconciled: backend goal is terminal while local queue/state was blocked"
			action.RecommendedAction = "reconcile_goal_terminal"
			resolved = append(resolved, action)
			continue
		}
		blocked = append(blocked, action)
	}
	return blocked, resolved
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
	action.RunStatus = firstNonEmptyMCPV0(action.RunStatus, strings.TrimSpace(stats.Status))
	action.ProcessAliveCount = liveness.AgentsLive
	action.ProcessRefs = compactStringsMCPV0(append(action.ProcessRefs, mcpAutoprogrammingStatsProcessRefsV0(stats)...))
	if stats.UsageSummary != nil && stats.UsageSummary.TotalTokens > 0 {
		action.TokensUsed = stats.UsageSummary.TotalTokens
	}
	action.AckDetected, action.LastAckAt = mcpAutoprogrammingStatsAckSummaryV0(stats)
	action.LastOutputAt = mcpAutoprogrammingStatsLastActivityAtV0(stats)
	action.LastArtifactAt = mcpAutoprogrammingStatsLastArtifactAtV0(stats)
	return action
}

func mcpAutoprogrammingActionableRunWithGoalStateSnapshotV0(
	action MCPAutoprogrammingActionableRunV0,
	state orquestagoal.GoalWorkStateV0,
) MCPAutoprogrammingActionableRunV0 {
	if state.LastResult != nil {
		action.ResultRef = mcpObserveAppDirectorGoalResultRefV0(*state.LastResult)
		action.ArtifactRefs = compactStringsMCPV0(append(action.ArtifactRefs, state.LastResult.ArtifactRefs...))
		action.DomainReceiptRefs = compactStringsMCPV0(append(action.DomainReceiptRefs, state.LastResult.DomainReceiptRefs...))
		action.EvidenceRefs = compactStringsMCPV0(append(action.EvidenceRefs, state.LastResult.EvidenceRefs...))
	}
	if state.LastClosure != nil {
		action.ClosureStatus = strings.TrimSpace(state.LastClosure.Status)
		action.ClosureAccepted = state.LastClosure.Accepted
		action.ClosureNeedsRework = state.LastClosure.NeedsRework
		action.EvidenceRefs = compactStringsMCPV0(append(action.EvidenceRefs, state.LastClosure.EvidenceRefs...))
	}
	if action.ExternalGoalRef == "" {
		action.ExternalGoalRef = strings.TrimSpace(state.LaunchReceipt.ExternalGoalRef)
	}
	return action
}

func mcpAutoprogrammingActionableRunWithObservedGoalV0(
	action MCPAutoprogrammingActionableRunV0,
	observed *MCPDirectorStatsToolResultV0,
) MCPAutoprogrammingActionableRunV0 {
	if observed == nil || observed.Goal == nil {
		return action
	}
	goal := observed.Goal
	action.GoalRef = firstNonEmptyMCPV0(action.GoalRef, goal.GoalRef)
	action.ExternalGoalRef = firstNonEmptyMCPV0(action.ExternalGoalRef, goal.ExternalGoalRef)
	action.GoalStatus = firstNonEmptyMCPV0(strings.TrimSpace(goal.Status), action.GoalStatus)
	action.ClosureStatus = firstNonEmptyMCPV0(action.ClosureStatus, goal.ClosureStatus)
	action.ClosureAccepted = action.ClosureAccepted || goal.ClosureAccepted
	action.ClosureNeedsRework = action.ClosureNeedsRework || goal.ClosureNeedsRework
	action.ArtifactRefs = compactStringsMCPV0(append(action.ArtifactRefs, goal.ArtifactRefs...))
	action.DomainReceiptRefs = compactStringsMCPV0(append(action.DomainReceiptRefs, goal.DomainReceiptRefs...))
	action.ExpectedReceiptRefs = compactStringsMCPV0(append(action.ExpectedReceiptRefs, goal.ExpectedReceiptRefs...))
	action.EvidenceRefs = compactStringsMCPV0(append(action.EvidenceRefs, goal.EvidenceRefs...))
	return action
}

func mcpAutoprogrammingCheckpointOnlyHighConsumptionV0(
	action MCPAutoprogrammingActionableRunV0,
	observed *MCPDirectorStatsToolResultV0,
	policy MCPAutoprogrammingGoalProgressPolicyV0,
) bool {
	policy = NormalizeMCPAutoprogrammingGoalProgressPolicyV0(policy)
	if !mcpAutoprogrammingObservedGoalActiveV0(observed) ||
		action.TokensUsed < policy.CheckpointOnlyHighConsumptionTokens ||
		len(action.DomainReceiptRefs) > 0 {
		return false
	}
	artifactRefs := compactStringsMCPV0(action.ArtifactRefs)
	if len(artifactRefs) == 0 {
		return false
	}
	for _, ref := range artifactRefs {
		if !mcpAutoprogrammingArtifactRefLooksCheckpointV0(ref) {
			return false
		}
	}
	return true
}

func mcpAutoprogrammingActiveTimeoutCheckpointRecentV0(
	action MCPAutoprogrammingActionableRunV0,
	observed *MCPDirectorStatsToolResultV0,
	policy MCPAutoprogrammingGoalProgressPolicyV0,
) bool {
	policy = NormalizeMCPAutoprogrammingGoalProgressPolicyV0(policy)
	if !mcpAutoprogrammingObservedGoalActiveV0(observed) ||
		action.TokensUsed >= policy.CheckpointOnlyHighConsumptionTokens ||
		len(action.DomainReceiptRefs) > 0 {
		return false
	}
	artifactRefs := compactStringsMCPV0(action.ArtifactRefs)
	if len(artifactRefs) == 0 {
		return false
	}
	for _, ref := range artifactRefs {
		if !mcpAutoprogrammingArtifactRefLooksCheckpointV0(ref) {
			return false
		}
	}
	return true
}

func mcpAutoprogrammingNoCheckpointHighConsumptionV0(
	action MCPAutoprogrammingActionableRunV0,
	observed *MCPDirectorStatsToolResultV0,
	policy MCPAutoprogrammingGoalProgressPolicyV0,
) bool {
	policy = NormalizeMCPAutoprogrammingGoalProgressPolicyV0(policy)
	return mcpAutoprogrammingObservedGoalActiveV0(observed) &&
		action.TokensUsed >= policy.CheckpointOnlyHighConsumptionTokens &&
		len(compactStringsMCPV0(action.ArtifactRefs)) == 0 &&
		len(compactStringsMCPV0(action.DomainReceiptRefs)) == 0
}

func mcpAutoprogrammingNoCheckpointConsumptionWarningV0(
	action MCPAutoprogrammingActionableRunV0,
	observed *MCPDirectorStatsToolResultV0,
	policy MCPAutoprogrammingGoalProgressPolicyV0,
) bool {
	policy = NormalizeMCPAutoprogrammingGoalProgressPolicyV0(policy)
	warningTokens := policy.CheckpointOnlyHighConsumptionTokens / 2
	if warningTokens <= 0 {
		warningTokens = mcpAutoprogrammingCheckpointOnlyHighConsumptionTokensDefaultV0 / 2
	}
	return mcpAutoprogrammingObservedGoalActiveV0(observed) &&
		action.TokensUsed >= warningTokens &&
		action.TokensUsed < policy.CheckpointOnlyHighConsumptionTokens &&
		len(compactStringsMCPV0(action.ArtifactRefs)) == 0 &&
		len(compactStringsMCPV0(action.DomainReceiptRefs)) == 0
}

func mcpAutoprogrammingArtifactRefLooksCheckpointV0(ref string) bool {
	ref = strings.ToLower(strings.TrimSpace(ref))
	return strings.Contains(ref, "checkpoint")
}

func mcpAutoprogrammingGoalStateActiveTimeoutV0(
	state orquestagoal.GoalWorkStateV0,
) bool {
	if state.LastResult != nil {
		if strings.Contains(strings.TrimSpace(state.LastResult.Summary), "codex_app_server_goal_active_timeout") {
			return true
		}
		for _, issue := range state.LastResult.Issues {
			if strings.Contains(strings.TrimSpace(issue.Code), "codex_app_server_goal_active_timeout") {
				return true
			}
		}
		for _, ref := range state.LastResult.EvidenceRefs {
			if mcpAutoprogrammingEvidenceLooksGoalActiveTimeoutV0(ref) {
				return true
			}
		}
	}
	for _, ref := range state.EvidenceRefs {
		if mcpAutoprogrammingEvidenceLooksGoalActiveTimeoutV0(ref) {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingEvidenceLooksGoalActiveTimeoutV0(ref string) bool {
	ref = strings.TrimSpace(ref)
	return strings.Contains(ref, "codex-app-server-goal-active-timeout") ||
		strings.Contains(ref, "codex_app_server_goal_active_timeout")
}

func mcpAutoprogrammingObservedGoalActiveV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil || observed.Goal == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(observed.Goal.Status)) {
	case "active", orquestagoal.GoalStatusRunningV0, orquestagoal.GoalStatusAcceptedV0:
		return true
	default:
		return false
	}
}

func mcpAutoprogrammingObservedGoalTerminalV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if observed == nil || observed.Goal == nil {
		return false
	}
	return mcpAutoprogrammingGoalStatusTerminalV0(observed.Goal.Status) ||
		observed.Goal.ClosureAccepted ||
		mcpAutoprogrammingGoalStatusTerminalV0(observed.Goal.ClosureStatus)
}

func mcpAutoprogrammingObservedGoalActiveWithRecentActivityV0(
	observed *MCPDirectorStatsToolResultV0,
) bool {
	if !mcpAutoprogrammingObservedGoalActiveV0(observed) {
		return false
	}
	if observed.Stats == nil {
		return false
	}
	stats := *observed.Stats
	if mcpAutoprogrammingStatsLastActivityAtV0(stats) != "" ||
		mcpAutoprogrammingStatsLastArtifactAtV0(stats) != "" {
		return true
	}
	return mcpAutoprogrammingRunStatsHasLiveSignalV0(stats)
}

func mcpAutoprogrammingGoalResultTerminalV0(
	state orquestagoal.GoalWorkStateV0,
) bool {
	return state.LastResult != nil && mcpAutoprogrammingGoalStatusTerminalV0(state.LastResult.Status)
}

func mcpAutoprogrammingGoalStatusTerminalV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case orquestagoal.GoalStatusCompleteV0, orquestagoal.GoalStatusAcceptedV0:
		return true
	default:
		return false
	}
}

func mcpAutoprogrammingObservedRunStatusV0(
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
	runRef string,
) string {
	observed := observedByRunRef[strings.TrimSpace(runRef)]
	if observed == nil || observed.Stats == nil {
		return ""
	}
	return strings.TrimSpace(observed.Stats.Status)
}

func mcpAutoprogrammingStatsProcessRefsV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) []string {
	out := make([]string, 0)
	for _, agent := range stats.Agents {
		if agent.Process == nil {
			continue
		}
		out = append(out, agent.Process.ProcessRef, agent.Process.SessionRef, agent.Process.LaunchRef)
	}
	return compactStringsMCPV0(out)
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
