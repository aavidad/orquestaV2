package orquestamcp

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

type MCPRunControlToolExecutorV0 struct {
	Port               orquestaruncontrol.RunControlWriterPortV0
	ExternalJobSource  MCPDirectorExternalJobStatsSourcePortV0
	GoalBackendState   MCPTransportDirectorStatsExecutorV0
	GoalStateStore     orquestagoal.GoalWorkStateStorePortV0
	GoalProgressPolicy MCPAutoprogrammingGoalProgressPolicyV0
}

func NewMCPRunControlToolExecutorV0(
	port orquestaruncontrol.RunControlWriterPortV0,
) MCPRunControlToolExecutorV0 {
	return MCPRunControlToolExecutorV0{Port: port}
}

func (executor MCPRunControlToolExecutorV0) Execute(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var issues []MCPValidationIssueV0
	input, issues = normalizeMCPRunControlIdentityV0(input, "", "")
	if len(issues) > 0 {
		return newMCPRunControlErrorV0(input, issues[0].Code, issues[0].Field), nil
	}
	resolved, ok, err := executor.resolveRunControlInputV0(ctx, input)
	if err != nil {
		return newMCPRunControlErrorV0(input, "external_job_run_ref_error", "external_job_ref"), nil
	}
	if !ok || strings.TrimSpace(resolved.RunRef) == "" {
		return newMCPRunControlErrorV0(input, "run_ref_requerido", "run_ref"), nil
	}
	if executor.Port == nil {
		return newMCPRunControlErrorV0(resolved, "run_control_port_no_disponible", "port"), nil
	}
	if !isMCPRunControlActionSupportedV0(input.Action) {
		return newMCPRunControlErrorV0(resolved, "action_no_soportada", "action"), nil
	}
	beforeLocal := executor.readRunControlStateIfAvailableV0(ctx, resolved.RunRef)
	beforeGoal := executor.observeRunControlGoalBackendV0(ctx, resolved)
	state, err := executor.executeActionV0(ctx, resolved)
	if err != nil {
		return MCPRunControlToolResultV0{}, err
	}
	afterGoal := executor.observeRunControlGoalBackendV0(ctx, resolved)
	result := newMCPRunControlResultV0(resolved, state)
	result = executor.enrichRunControlGoalBackendResultV0(result, resolved, beforeLocal, beforeGoal, afterGoal)
	result = executor.reconcileGoalStateAfterForcedControlV0(ctx, result, resolved, beforeGoal, afterGoal)
	return result, nil
}

func (executor MCPRunControlToolExecutorV0) resolveRunControlInputV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolInputV0, bool, error) {
	input.RunRef = strings.TrimSpace(input.RunRef)
	if input.RunRef != "" {
		return input, true, nil
	}
	if strings.TrimSpace(input.ExternalJobRef) == "" {
		return input, false, nil
	}
	if executor.ExternalJobSource == nil {
		return input, false, nil
	}
	stats, ok, err := executor.ExternalJobSource.ResolveDirectorExternalJobStatsV0(
		ctx,
		MCPDirectorExternalJobStatsRequestV0{
			AppRef:         strings.TrimSpace(input.AppRef),
			ExternalJobRef: strings.TrimSpace(input.ExternalJobRef),
			CorrelationID:  firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		},
	)
	if err != nil || !ok || strings.TrimSpace(stats.RunRef) == "" {
		return input, ok, err
	}
	input.RunRef = strings.TrimSpace(stats.RunRef)
	input.EvidenceRefs = compactStringsMCPV0(append(input.EvidenceRefs, stats.JobRef, stats.TaskRef, stats.AgentRef))
	return input, true, nil
}

func (executor MCPRunControlToolExecutorV0) readRunControlStateIfAvailableV0(
	ctx context.Context,
	runRef string,
) *orquestaruncontrol.RunControlStateV0 {
	reader, ok := executor.Port.(orquestaruncontrol.RunControlReaderPortV0)
	if !ok || reader == nil {
		return nil
	}
	state, err := reader.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: strings.TrimSpace(runRef)},
	)
	if err != nil {
		return nil
	}
	normalized := state
	normalized.Status = orquestaruncontrol.NormalizeRunControlStatusV0(state.Status)
	return &normalized
}

func (executor MCPRunControlToolExecutorV0) observeRunControlGoalBackendV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) *MCPDirectorStatsToolResultV0 {
	if executor.GoalBackendState == nil || strings.TrimSpace(input.RunRef) == "" {
		return nil
	}
	stats, err := executor.GoalBackendState.Execute(ctx, MCPDirectorStatsToolInputV0{
		RequestID:            input.RequestID,
		CorrelationID:        firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:               strings.TrimSpace(input.RunRef),
		AppRef:               strings.TrimSpace(input.AppRef),
		OccurredAt:           "",
		IncludeProcessRefs:   true,
		IncludeAgentProgress: true,
		IncludeAgentUsage:    true,
	})
	if err != nil || stats.Estado != MCPDirectorStatsEstadoOKV0 || stats.Goal == nil {
		return nil
	}
	return &stats
}

func (executor MCPRunControlToolExecutorV0) enrichRunControlGoalBackendResultV0(
	result MCPRunControlToolResultV0,
	input MCPRunControlToolInputV0,
	beforeLocal *orquestaruncontrol.RunControlStateV0,
	beforeGoal *MCPDirectorStatsToolResultV0,
	afterGoal *MCPDirectorStatsToolResultV0,
) MCPRunControlToolResultV0 {
	action := normalizeMCPRunControlActionV0(input.Action)
	if action != "stop" && action != "cancel" {
		return result
	}
	if beforeLocal != nil {
		result.PreviousStatus = string(orquestaruncontrol.NormalizeRunControlStatusV0(beforeLocal.Status))
	}
	result.GoalStatusBefore = mcpRunControlGoalStatusFromStatsV0(beforeGoal)
	result.GoalStatusAfter = mcpRunControlGoalStatusFromStatsV0(afterGoal)
	result.GoalRef = firstNonEmptyMCPV0(
		mcpRunControlGoalRefFromStatsV0(afterGoal),
		mcpRunControlGoalRefFromStatsV0(beforeGoal),
	)
	result.ExternalGoalRef = firstNonEmptyMCPV0(
		mcpRunControlExternalGoalRefFromStatsV0(afterGoal),
		mcpRunControlExternalGoalRefFromStatsV0(beforeGoal),
	)
	result.GoalControlSignalConfirmed = mcpRunControlGoalBackendTerminalV0(afterGoal)
	if !mcpRunControlGoalBackendActiveV0(afterGoal) {
		return result
	}
	backendEvidenceRefs := compactStringsMCPV0([]string{
		"evidence-ref-run-control-goal-backend-active",
		result.GoalRef,
		result.ExternalGoalRef,
	})
	backendEvidenceRefs = compactStringsMCPV0(append(
		backendEvidenceRefs,
		mcpRunControlGoalEvidenceRefsV0(afterGoal)...,
	))
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, backendEvidenceRefs...))
	result.GoalControlSignalSent = false
	result.GoalControlSignalConfirmed = false
	result.RecommendedAction = "observe_goal_backend_before_declaring_stopped"
	result.Diagnostics = append(result.Diagnostics, MCPRunControlDiagnosticV0{
		Code:         "control_not_propagated_to_goal_backend",
		Scope:        "run:" + strings.TrimSpace(result.RunRef),
		Message:      "run control local no confirma stop/cancel del backend goal-first; no publicar stopped como terminal",
		EvidenceRefs: backendEvidenceRefs,
	})
	result.Errores = append(result.Errores, MCPValidationIssueV0{
		Code:    "control_not_propagated_to_goal_backend",
		Field:   "goal_backend",
		Message: "goal backend sigue activo tras control local",
	})
	result.Estado = MCPRunControlEstadoErrorV0
	result.Status = mcpRunControlRequestedStatusForActionV0(action)
	result.FinalStatus = result.Status
	return result
}

func (executor MCPRunControlToolExecutorV0) reconcileGoalStateAfterForcedControlV0(
	ctx context.Context,
	result MCPRunControlToolResultV0,
	input MCPRunControlToolInputV0,
	beforeGoal *MCPDirectorStatsToolResultV0,
	afterGoal *MCPDirectorStatsToolResultV0,
) MCPRunControlToolResultV0 {
	action := normalizeMCPRunControlActionV0(input.Action)
	if executor.GoalStateStore == nil ||
		result.Estado != MCPRunControlEstadoOKV0 ||
		(action != "stop" && action != "cancel") ||
		mcpRunControlGoalBackendActiveV0(afterGoal) {
		return result
	}
	allowForcedReconcile := input.Forced
	allowExternalCleanupReconcile := mcpRunControlInputRequestsExternalCleanupReconcileV0(input)
	if !allowForcedReconcile && !allowExternalCleanupReconcile {
		return result
	}
	runRef := strings.TrimSpace(input.RunRef)
	state, err := executor.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return result
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil || orquestagoal.GoalWorkResultTerminalV0(state.Status) {
		return result
	}
	reasonCode, evidenceRef, ok := "", "", false
	if allowForcedReconcile {
		reasonCode, evidenceRef, ok = mcpRunControlForcedTerminalReasonV0(beforeGoal, executor.GoalProgressPolicy)
	}
	if !ok && (allowForcedReconcile || allowExternalCleanupReconcile) {
		reasonCode, evidenceRef, ok = mcpRunControlForcedExternalCleanupReasonV0(state, beforeGoal, afterGoal)
	}
	if !ok {
		return result
	}
	reconcileCode := "goal_state_terminal_reconciled_after_forced_stop"
	reconcileEvidenceRef := "evidence-ref-run-control-goal-forced-terminal-reconciled"
	reconcileSummary := "forced " + action + " reconciled active goal into terminal rework state"
	reconcileMessage := "goal-first state marked blocked/rework after forced " + action + " of high-consumption active backend"
	if allowExternalCleanupReconcile && !allowForcedReconcile {
		reconcileCode = "goal_state_terminal_reconciled_after_external_cleanup"
		reconcileEvidenceRef = "evidence-ref-run-control-goal-external-cleanup-reconciled"
		reconcileSummary = "external cleanup reconciled missing goal backend into terminal rework state"
		reconcileMessage = "goal-first state marked blocked/rework after governed reconciliation of external backend cleanup"
	}
	evidenceRefs := compactStringsMCPV0([]string{
		reconcileEvidenceRef,
		evidenceRef,
	})
	goalRef := firstNonEmptyMCPV0(
		state.GoalRef,
		mcpRunControlGoalRefFromStatsV0(beforeGoal),
		mcpRunControlGoalRefFromStatsV0(afterGoal),
	)
	externalGoalRef := firstNonEmptyMCPV0(
		state.ExternalGoalRef,
		mcpRunControlExternalGoalRefFromStatsV0(beforeGoal),
		mcpRunControlExternalGoalRefFromStatsV0(afterGoal),
	)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Summary:         reconcileSummary,
		ArtifactRefs:    compactStringsMCPV0(mcpRunControlGoalArtifactRefsV0(beforeGoal)),
		EvidenceRefs:    evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  reasonCode,
			Field: "goal_backend",
		}},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusBlockedV0,
		NeedsRework:  true,
		EvidenceRefs: evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  reasonCode,
			Field: "goal_backend",
		}},
	}
	state.EvidenceRefs = compactStringsMCPV0(append(state.EvidenceRefs, evidenceRefs...))
	if err := executor.GoalStateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		return result
	}
	result = executor.completeRunControlAfterForcedGoalReconcileV0(ctx, result, input, evidenceRefs)
	result.RecommendedAction = "replan_narrow_context"
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, evidenceRefs...))
	result.Diagnostics = append(result.Diagnostics, MCPRunControlDiagnosticV0{
		Code:    reconcileCode,
		Scope:   "run:" + runRef,
		Message: reconcileMessage,
		EvidenceRefs: compactStringsMCPV0(append(
			evidenceRefs,
			goalRef,
			externalGoalRef,
		)),
	})
	return result
}

func mcpRunControlInputRequestsExternalCleanupReconcileV0(input MCPRunControlToolInputV0) bool {
	for _, ref := range compactStringsMCPV0(input.EvidenceRefs) {
		if ref == mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0 {
			return true
		}
	}
	return false
}

func (executor MCPRunControlToolExecutorV0) completeRunControlAfterForcedGoalReconcileV0(
	ctx context.Context,
	result MCPRunControlToolResultV0,
	input MCPRunControlToolInputV0,
	evidenceRefs []string,
) MCPRunControlToolResultV0 {
	terminal, ok := executor.Port.(orquestaruncontrol.RunControlTerminalWriterPortV0)
	if !ok || terminal == nil {
		return result
	}
	action := normalizeMCPRunControlActionV0(input.Action)
	target := orquestaruncontrol.RunControlStatusStoppedV0
	if action == "cancel" {
		target = orquestaruncontrol.RunControlStatusCanceledV0
	}
	completed, err := terminal.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         strings.TrimSpace(input.RunRef),
		TargetStatus:   target,
		RequestedBy:    firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-run-control"),
		Reason:         firstNonEmptyMCPV0(input.Reason, "forced goal backend reconcile completed run control"),
		IdempotencyKey: firstNonEmptyMCPV0(input.IdempotencyKey, "idem-mcp-run-control-forced-reconcile-"+action+"-"+strings.TrimSpace(input.RunRef)),
		EvidenceRefs:   compactStringsMCPV0(append(evidenceRefs, "evidence-ref-run-control-terminal-after-goal-reconcile")),
	})
	if err != nil {
		return result
	}
	result.Status = string(orquestaruncontrol.NormalizeRunControlStatusV0(completed.Status))
	result.FinalStatus = result.Status
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, completed.EvidenceRefs...))
	return result
}

func mcpRunControlForcedExternalCleanupReasonV0(
	state orquestagoal.GoalWorkStateV0,
	beforeGoal *MCPDirectorStatsToolResultV0,
	afterGoal *MCPDirectorStatsToolResultV0,
) (string, string, bool) {
	if mcpRunControlGoalBackendActiveV0(beforeGoal) ||
		mcpRunControlGoalBackendActiveV0(afterGoal) ||
		strings.TrimSpace(state.ExternalGoalRef) == "" ||
		!orquestagoal.GoalWorkStatePendingObservationV0(state) {
		return "", "", false
	}
	return mcpAutoprogrammingActionGoalBackendMissingAfterExternalCleanupV0,
		mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
		true
}

func mcpRunControlForcedTerminalReasonV0(
	stats *MCPDirectorStatsToolResultV0,
	policy MCPAutoprogrammingGoalProgressPolicyV0,
) (string, string, bool) {
	policy = NormalizeMCPAutoprogrammingGoalProgressPolicyV0(policy)
	if stats == nil ||
		!mcpRunControlGoalBackendActiveV0(stats) ||
		mcpRunControlGoalTokensV0(stats) < policy.CheckpointOnlyHighConsumptionTokens ||
		len(mcpRunControlGoalDomainReceiptRefsV0(stats)) > 0 {
		return "", "", false
	}
	artifactRefs := mcpRunControlGoalArtifactRefsV0(stats)
	if len(artifactRefs) == 0 {
		return mcpAutoprogrammingActionNoCheckpointHighConsumptionV0,
			mcpAutoprogrammingEvidenceNoCheckpointHighConsumptionV0,
			true
	}
	for _, ref := range artifactRefs {
		if !mcpAutoprogrammingArtifactRefLooksCheckpointV0(ref) {
			return "", "", false
		}
	}
	return mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0,
		mcpAutoprogrammingEvidenceCheckpointOnlyHighConsumptionV0,
		true
}

func mcpRunControlGoalTokensV0(stats *MCPDirectorStatsToolResultV0) int64 {
	if stats == nil || stats.Stats == nil || stats.Stats.UsageSummary == nil {
		return 0
	}
	return stats.Stats.UsageSummary.TotalTokens
}

func mcpRunControlGoalArtifactRefsV0(stats *MCPDirectorStatsToolResultV0) []string {
	if stats == nil || stats.Goal == nil {
		return []string{}
	}
	return compactStringsMCPV0(stats.Goal.ArtifactRefs)
}

func mcpRunControlGoalDomainReceiptRefsV0(stats *MCPDirectorStatsToolResultV0) []string {
	if stats == nil || stats.Goal == nil {
		return []string{}
	}
	return compactStringsMCPV0(stats.Goal.DomainReceiptRefs)
}

func mcpRunControlGoalEvidenceRefsV0(stats *MCPDirectorStatsToolResultV0) []string {
	if stats == nil || stats.Goal == nil {
		return []string{}
	}
	return compactStringsMCPV0(stats.Goal.EvidenceRefs)
}

func mcpRunControlGoalStatusFromStatsV0(stats *MCPDirectorStatsToolResultV0) string {
	if stats == nil || stats.Goal == nil {
		return ""
	}
	return strings.TrimSpace(stats.Goal.Status)
}

func mcpRunControlGoalRefFromStatsV0(stats *MCPDirectorStatsToolResultV0) string {
	if stats == nil || stats.Goal == nil {
		return ""
	}
	return strings.TrimSpace(stats.Goal.GoalRef)
}

func mcpRunControlExternalGoalRefFromStatsV0(stats *MCPDirectorStatsToolResultV0) string {
	if stats == nil || stats.Goal == nil {
		return ""
	}
	return strings.TrimSpace(stats.Goal.ExternalGoalRef)
}

func mcpRunControlGoalBackendActiveV0(stats *MCPDirectorStatsToolResultV0) bool {
	status := strings.ToLower(mcpRunControlGoalStatusFromStatsV0(stats))
	switch status {
	case "active", "running":
		return true
	default:
		return false
	}
}

func mcpRunControlGoalBackendTerminalV0(stats *MCPDirectorStatsToolResultV0) bool {
	status := strings.ToLower(mcpRunControlGoalStatusFromStatsV0(stats))
	switch status {
	case "complete", "completed", "accepted", "canceled", "cancelled", "stopped", "failed":
		return true
	default:
		return false
	}
}

func mcpRunControlRequestedStatusForActionV0(action string) string {
	switch action {
	case "cancel":
		return string(orquestaruncontrol.RunControlStatusCancelRequestedV0)
	default:
		return string(orquestaruncontrol.RunControlStatusStopRequestedV0)
	}
}

func (executor MCPRunControlToolExecutorV0) executeActionV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	switch normalizeMCPRunControlActionV0(input.Action) {
	case "pause":
		return executor.Port.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
	case "resume":
		return executor.Port.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
	case "stop":
		state, err := executor.Port.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			Forced:         input.Forced,
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
		if err != nil {
			return state, err
		}
		return executor.recordStopCheckpointIfAvailableV0(ctx, input, state)
	case "cancel":
		state, err := executor.Port.CancelRunV0(ctx, orquestaruncontrol.CancelRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			Forced:         input.Forced,
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
		if err != nil {
			return state, err
		}
		return executor.recordStopCheckpointIfAvailableV0(ctx, input, state)
	default:
		return orquestaruncontrol.RunControlStateV0{}, nil
	}
}

func (executor MCPRunControlToolExecutorV0) recordStopCheckpointIfAvailableV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
	state orquestaruncontrol.RunControlStateV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if state.CheckpointRecorded {
		return state, nil
	}
	checkpointWriter, ok := executor.Port.(orquestaruncontrol.RunControlCheckpointWriterPortV0)
	if !ok {
		return state, nil
	}
	action := normalizeMCPRunControlActionV0(input.Action)
	recorded, err := checkpointWriter.RecordRunCheckpointV0(ctx, orquestaruncontrol.RecordRunCheckpointCommandV0{
		RunRef:         strings.TrimSpace(input.RunRef),
		RequestedBy:    firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-run-control"),
		Reason:         firstNonEmptyMCPV0(input.Reason, "checkpoint registrado por control MCP de run"),
		IdempotencyKey: firstNonEmptyMCPV0(input.IdempotencyKey, "idem-mcp-run-control-checkpoint-"+action+"-"+strings.TrimSpace(input.RunRef)),
		EvidenceRefs:   compactStringsMCPV0(append(input.EvidenceRefs, "evidence-ref-mcp-run-control-checkpoint-recorded")),
	})
	if err != nil {
		return state, err
	}
	return recorded, nil
}

func isMCPRunControlActionSupportedV0(action string) bool {
	switch normalizeMCPRunControlActionV0(action) {
	case "pause", "resume", "stop", "cancel":
		return true
	default:
		return false
	}
}
