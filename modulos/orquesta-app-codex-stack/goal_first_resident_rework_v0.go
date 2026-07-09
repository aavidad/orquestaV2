package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

const (
	goalFirstResidentReworkPreparedEvidenceRefV0      = "evidence-ref-goal-first-resident-rework-prepared"
	goalFirstResidentReworkExistingEvidencePrefixV0   = "evidence-ref-goal-first-resident-rework-goal:"
	goalFirstResidentReworkReasonCheckpointOnlyV0     = "checkpoint_only_high_consumption"
	goalFirstResidentReworkReasonNoCheckpointV0       = "goal_active_no_checkpoint_high_consumption"
	goalFirstResidentReworkReasonActiveTimeoutV0      = "codex_app_server_goal_active_timeout"
	goalFirstResidentReworkReasonQAFailedTextV0       = orquestamcp.MCPGoalFirstQAFailedPublicTextV0
	goalFirstResidentReworkReasonArtifactPathsV0      = orquestamcp.MCPGoalFirstArtifactPathsOmittedMaterializedV0
	goalFirstResidentReworkReasonOutOfScopeV0         = orquestamcp.MCPGoalFirstOutOfScopeMaterializedArtifactsV0
	goalFirstResidentReworkReasonMissingReceiptV0     = orquestamcp.MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0
	goalFirstResidentReworkReasonRequiredTestsV0      = orquestamcp.MCPGoalFirstRequiredTestEvidenceMissingV0
	goalFirstResidentReworkReasonPhase0V0             = orquestamcp.MCPGoalFirstPhase0CompleteNonPublishableV0
	goalFirstResidentReworkReasonPartialArtifactsV0   = orquestamcp.MCPGoalFirstPartialArtifactsWrittenV0
	goalFirstResidentReworkReasonBackendMissingV0     = "goal_backend_missing_after_external_cleanup"
	goalFirstResidentReworkReasonWorkdirV0            = "codex_app_server_write_set_prepare_failed"
	goalFirstResidentReworkReasonAuthV0               = "codex_app_server_provider_unauthorized"
	goalFirstResidentReworkReasonAuthMissingV0        = "codex_app_server_auth_missing"
	goalFirstResidentReworkReasonProviderLimitedV0    = "codex_app_server_goal_provider_limited"
	goalFirstResidentReworkReasonStorageQuotaV0       = "codex_app_server_storage_quota_exceeded"
	goalFirstResidentReworkReasonBackendUnavailableV0 = "codex_app_server_unavailable"
	goalFirstResidentBackendMissingEvidenceRefV0      = "evidence-ref-autoprogramming-goal-backend-missing-after-external-cleanup"
	goalFirstResidentBackendMissingReconciledV0       = "evidence-ref-goal-first-resident-backend-missing-reconciled"
	goalFirstResidentRunControlTerminalEvidenceV0     = "evidence-ref-run-control-terminal-after-goal-reconcile"
)

func (executor CodexStackRunSupervisorExecutorV0) maybePrepareGoalFirstResidentReworkV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	state orquestagoal.GoalWorkStateV0,
	result orquestamcp.MCPRunSupervisorToolResultV0,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	if !input.ResidentMode || executor.Stack == nil {
		return result
	}
	store := executor.Stack.Ports.GoalStateStore
	launcher := executor.Stack.Ports.GoalReworkLauncher
	if store == nil || launcher == nil {
		return result
	}
	reason, evidenceRefs, ok := goalFirstResidentReworkReasonV0(state)
	if !ok {
		return result
	}
	if existing := goalFirstResidentExistingReworkRunRefV0(state); existing != "" {
		return goalFirstResidentReworkResultV0(result, state.RunRef, existing, reason, evidenceRefs, true)
	}
	spec := goalFirstResidentReworkSpecV0(state, reason, evidenceRefs)
	if _, err := store.LoadGoalWorkStateV0(ctx, spec.RunRef); err == nil {
		state.EvidenceRefs = compactStringsV0(append(
			state.EvidenceRefs,
			goalFirstResidentReworkExistingEvidencePrefixV0+spec.RunRef,
			goalFirstResidentReworkPreparedEvidenceRefV0,
		))
		_ = store.SaveGoalWorkStateV0(ctx, state)
		return goalFirstResidentReworkResultV0(result, state.RunRef, spec.RunRef, reason, evidenceRefs, true)
	}
	start, err := orquestagoal.StartGoalWorkV0(ctx, orquestagoal.GoalWorkStartRequestV0{
		RunRef:       spec.RunRef,
		Spec:         spec,
		EvidenceRefs: compactStringsV0(append(evidenceRefs, goalFirstResidentReworkPreparedEvidenceRefV0)),
	}, orquestagoal.GoalWorkLifecyclePortsV0{
		Launcher:   launcher,
		StateStore: store,
	})
	if err != nil {
		result.NextActions = compactStringsV0(append(result.NextActions, "goal_first_rework_launcher_failed", "inspect_goal_rework_launcher"))
		result.Diagnostics = append(result.Diagnostics, orquestamcp.MCPAutoprogrammingDiagnosticV0{
			Code:         "goal_first_resident_rework_launch_failed",
			Scope:        "run:" + strings.TrimSpace(state.RunRef),
			Message:      err.Error(),
			EvidenceRefs: evidenceRefs,
		})
		return result
	}
	reworkRunRef := strings.TrimSpace(start.State.RunRef)
	if reworkRunRef == "" {
		reworkRunRef = strings.TrimSpace(spec.RunRef)
	}
	state.EvidenceRefs = compactStringsV0(append(
		state.EvidenceRefs,
		goalFirstResidentReworkExistingEvidencePrefixV0+reworkRunRef,
		goalFirstResidentReworkPreparedEvidenceRefV0,
	))
	_ = store.SaveGoalWorkStateV0(ctx, state)
	return goalFirstResidentReworkResultV0(result, state.RunRef, reworkRunRef, reason, evidenceRefs, false)
}

func goalFirstResidentReworkReasonV0(state orquestagoal.GoalWorkStateV0) (string, []string, bool) {
	if !goalFirstResidentStateNeedsReworkV0(state) {
		return "", nil, false
	}
	if goalFirstResidentHasIssueOrEvidenceV0(state, goalFirstResidentReworkReasonCheckpointOnlyV0) {
		return goalFirstResidentReworkReasonCheckpointOnlyV0, goalFirstResidentReworkEvidenceRefsV0(state), true
	}
	if goalFirstResidentHasIssueOrEvidenceV0(state, goalFirstResidentReworkReasonNoCheckpointV0) {
		return goalFirstResidentReworkReasonNoCheckpointV0, goalFirstResidentReworkEvidenceRefsV0(state), true
	}
	if goalFirstResidentInitialTimeoutWithoutArtifactsV0(state) {
		return goalFirstResidentReworkReasonActiveTimeoutV0, goalFirstResidentReworkEvidenceRefsV0(state), true
	}
	for _, reason := range []string{
		goalFirstResidentReworkReasonBackendMissingV0,
		goalFirstResidentReworkReasonWorkdirV0,
		goalFirstResidentReworkReasonAuthV0,
		goalFirstResidentReworkReasonAuthMissingV0,
		goalFirstResidentReworkReasonProviderLimitedV0,
		goalFirstResidentReworkReasonStorageQuotaV0,
		goalFirstResidentReworkReasonBackendUnavailableV0,
		goalFirstResidentReworkReasonQAFailedTextV0,
		goalFirstResidentReworkReasonArtifactPathsV0,
		goalFirstResidentReworkReasonOutOfScopeV0,
		goalFirstResidentReworkReasonMissingReceiptV0,
		goalFirstResidentReworkReasonRequiredTestsV0,
		goalFirstResidentReworkReasonPhase0V0,
		goalFirstResidentReworkReasonPartialArtifactsV0,
	} {
		if goalFirstResidentHasIssueOrEvidenceV0(state, reason) {
			return reason, goalFirstResidentReworkEvidenceRefsV0(state), true
		}
	}
	return "", nil, false
}

func goalFirstResidentStateNeedsReworkV0(state orquestagoal.GoalWorkStateV0) bool {
	status := strings.TrimSpace(state.Status)
	if status == orquestagoal.GoalStatusBlockedV0 || status == orquestagoal.GoalStatusInvalidV0 {
		return true
	}
	if state.LastClosure != nil && state.LastClosure.NeedsRework {
		return true
	}
	if state.LastResult != nil {
		resultStatus := strings.TrimSpace(state.LastResult.Status)
		return resultStatus == orquestagoal.GoalStatusBlockedV0 || resultStatus == orquestagoal.GoalStatusInvalidV0
	}
	return false
}

func goalFirstResidentHasIssueOrEvidenceV0(state orquestagoal.GoalWorkStateV0, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for _, issue := range goalFirstResidentIssueCodesV0(state) {
		if strings.TrimSpace(issue) == code {
			return true
		}
	}
	for _, ref := range goalFirstResidentAllEvidenceRefsV0(state) {
		if strings.Contains(strings.TrimSpace(ref), code) ||
			strings.Contains(strings.TrimSpace(ref), strings.ReplaceAll(code, "_", "-")) {
			return true
		}
	}
	if state.LastResult != nil && strings.Contains(strings.TrimSpace(state.LastResult.Summary), code) {
		return true
	}
	return false
}

func goalFirstResidentInitialTimeoutWithoutArtifactsV0(state orquestagoal.GoalWorkStateV0) bool {
	if !goalFirstResidentHasIssueOrEvidenceV0(state, goalFirstResidentReworkReasonActiveTimeoutV0) ||
		state.LastResult == nil {
		return false
	}
	if len(compactStringsV0(state.LastResult.DomainReceiptRefs)) > 0 {
		return false
	}
	return len(compactStringsV0(state.LastResult.ArtifactRefs)) == 0 &&
		len(compactStringsV0(state.LastResult.ArtifactPaths)) == 0
}

func goalFirstResidentIssueCodesV0(state orquestagoal.GoalWorkStateV0) []string {
	var codes []string
	if state.LastResult != nil {
		for _, issue := range state.LastResult.Issues {
			codes = append(codes, issue.Code)
		}
	}
	if state.LastClosure != nil {
		for _, issue := range state.LastClosure.Issues {
			codes = append(codes, issue.Code)
		}
	}
	return compactStringsV0(codes)
}

func goalFirstResidentAllEvidenceRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	var refs []string
	refs = append(refs, state.EvidenceRefs...)
	refs = append(refs, state.Spec.EvidenceRefs...)
	refs = append(refs, state.LaunchReceipt.EvidenceRefs...)
	if state.LastResult != nil {
		refs = append(refs, state.LastResult.EvidenceRefs...)
	}
	if state.LastClosure != nil {
		refs = append(refs, state.LastClosure.EvidenceRefs...)
	}
	return compactStringsV0(refs)
}

func goalFirstResidentReworkEvidenceRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	refs := []string{goalFirstResidentReworkPreparedEvidenceRefV0}
	for _, ref := range goalFirstResidentAllEvidenceRefsV0(state) {
		trimmed := strings.TrimSpace(ref)
		if strings.Contains(trimmed, "high-consumption") ||
			strings.Contains(trimmed, "high_consumption") ||
			strings.Contains(trimmed, "checkpoint-only") ||
			strings.Contains(trimmed, "no-checkpoint") ||
			strings.Contains(trimmed, "active-timeout") ||
			strings.Contains(trimmed, "goal-active-timeout") ||
			strings.Contains(trimmed, "goal-materialized") ||
			strings.Contains(trimmed, "artifact_paths") ||
			strings.Contains(trimmed, "artifact-paths") ||
			strings.Contains(trimmed, "external-cleanup") ||
			strings.Contains(trimmed, "external_cleanup") ||
			strings.Contains(trimmed, "backend-missing") ||
			strings.Contains(trimmed, "backend_missing") ||
			strings.Contains(trimmed, "backend-unavailable") ||
			strings.Contains(trimmed, "backend_unavailable") ||
			strings.Contains(trimmed, "write-set-prepare") ||
			strings.Contains(trimmed, "write_set_prepare") ||
			strings.Contains(trimmed, "workdir") ||
			strings.Contains(trimmed, "provider-unauthorized") ||
			strings.Contains(trimmed, "provider_unauthorized") ||
			strings.Contains(trimmed, "auth-missing") ||
			strings.Contains(trimmed, "auth_missing") ||
			strings.Contains(trimmed, "provider-limited") ||
			strings.Contains(trimmed, "provider_limited") ||
			strings.Contains(trimmed, "storage-quota") ||
			strings.Contains(trimmed, "storage_quota") ||
			strings.Contains(trimmed, "required-test") ||
			strings.Contains(trimmed, "required_test") {
			refs = append(refs, trimmed)
		}
	}
	return compactStringsV0(refs)
}

func (executor CodexStackRunSupervisorExecutorV0) maybeReconcileGoalFirstResidentBackendMissingV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	state orquestagoal.GoalWorkStateV0,
	result orquestamcp.MCPRunSupervisorToolResultV0,
) (orquestagoal.GoalWorkStateV0, orquestamcp.MCPRunSupervisorToolResultV0) {
	if !input.ResidentMode ||
		executor.Stack == nil ||
		executor.Stack.Ports.GoalStateStore == nil ||
		executor.Stack.MCPTransportBindings.DirectorStats == nil ||
		strings.TrimSpace(state.ExternalGoalRef) == "" ||
		!orquestagoal.GoalWorkStatePendingObservationV0(state) {
		return state, result
	}
	observed, err := executor.Stack.MCPTransportBindings.DirectorStats.Execute(
		ctx,
		orquestamcp.MCPDirectorStatsToolInputV0{
			RunRef: strings.TrimSpace(state.RunRef),
		},
	)
	if err != nil ||
		strings.TrimSpace(observed.Estado) != orquestamcp.MCPDirectorStatsEstadoOKV0 ||
		goalFirstResidentObservedGoalActiveV0(observed) ||
		goalFirstResidentObservedGoalTerminalV0(observed) {
		return state, result
	}
	evidenceRefs := compactStringsV0(append(
		[]string{
			goalFirstResidentBackendMissingReconciledV0,
			goalFirstResidentBackendMissingEvidenceRefV0,
		},
		goalFirstResidentObservedGoalEvidenceRefsV0(observed)...,
	))
	goalRef := firstNonEmptyQueuedSourceV0(state.GoalRef, goalFirstResidentObservedGoalRefV0(observed))
	externalGoalRef := firstNonEmptyQueuedSourceV0(state.ExternalGoalRef, goalFirstResidentObservedExternalGoalRefV0(observed))
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Summary:         "resident supervisor reconciled missing backend goal after external cleanup",
		ArtifactRefs:    compactStringsV0(goalFirstResidentObservedArtifactRefsV0(observed)),
		EvidenceRefs:    evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  goalFirstResidentReworkReasonBackendMissingV0,
			Field: "goal_backend",
		}},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusBlockedV0,
		NeedsRework:  true,
		EvidenceRefs: evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  goalFirstResidentReworkReasonBackendMissingV0,
			Field: "goal_backend",
		}},
	}
	state.EvidenceRefs = compactStringsV0(append(state.EvidenceRefs, evidenceRefs...))
	if err := executor.Stack.Ports.GoalStateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		result.Diagnostics = append(result.Diagnostics, orquestamcp.MCPAutoprogrammingDiagnosticV0{
			Code:         "goal_first_resident_backend_missing_reconcile_failed",
			Scope:        "run:" + strings.TrimSpace(state.RunRef),
			Message:      err.Error(),
			EvidenceRefs: evidenceRefs,
		})
		return state, result
	}
	result = executor.completeRunControlAfterResidentBackendMissingV0(ctx, input, state, evidenceRefs, result)
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, evidenceRefs...))
	result.NextActions = compactStringsV0(append(result.NextActions, "goal_first_resident_rework_after_external_cleanup"))
	result.Diagnostics = append(result.Diagnostics, orquestamcp.MCPAutoprogrammingDiagnosticV0{
		Code:         "goal_first_resident_backend_missing_reconciled",
		Scope:        "run:" + strings.TrimSpace(state.RunRef),
		Message:      "resident supervisor marked running goal-first state blocked/rework after observing missing backend goal",
		EvidenceRefs: evidenceRefs,
	})
	return state, result
}

func (executor CodexStackRunSupervisorExecutorV0) completeRunControlAfterResidentBackendMissingV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	state orquestagoal.GoalWorkStateV0,
	evidenceRefs []string,
	result orquestamcp.MCPRunSupervisorToolResultV0,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	if executor.Stack == nil || executor.Stack.Ports.RunControlTerminal == nil {
		return result
	}
	runRef := strings.TrimSpace(state.RunRef)
	if runRef == "" {
		runRef = strings.TrimSpace(input.RunRef)
	}
	completed, err := executor.Stack.Ports.RunControlTerminal.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:         runRef,
		TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
		RequestedBy:    "orquesta-run-supervisor-resident",
		Reason:         "external cleanup goal backend reconcile completed run control",
		IdempotencyKey: "idem-run-supervisor-external-cleanup-reconcile-stop-" + runRef,
		EvidenceRefs: compactStringsV0(append(
			evidenceRefs,
			goalFirstResidentRunControlTerminalEvidenceV0,
		)),
	})
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, orquestamcp.MCPAutoprogrammingDiagnosticV0{
			Code:         "goal_first_resident_run_control_terminal_failed",
			Scope:        "run:" + runRef,
			Message:      err.Error(),
			EvidenceRefs: evidenceRefs,
		})
		return result
	}
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, completed.EvidenceRefs...))
	result.Diagnostics = append(result.Diagnostics, orquestamcp.MCPAutoprogrammingDiagnosticV0{
		Code:    "goal_first_resident_run_control_terminal_completed",
		Scope:   "run:" + runRef,
		Message: "resident supervisor completed RunControl after external cleanup reconcile",
		EvidenceRefs: compactStringsV0(append(
			completed.EvidenceRefs,
			runRef,
		)),
	})
	return result
}

func goalFirstResidentObservedGoalActiveV0(
	observed orquestamcp.MCPDirectorStatsToolResultV0,
) bool {
	if observed.Goal == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(observed.Goal.Status)) {
	case "active", orquestagoal.GoalStatusRunningV0, orquestagoal.GoalStatusAcceptedV0:
		return true
	default:
		return false
	}
}

func goalFirstResidentObservedGoalTerminalV0(
	observed orquestamcp.MCPDirectorStatsToolResultV0,
) bool {
	if observed.Goal == nil {
		return false
	}
	return goalFirstResidentObservedGoalStatusTerminalV0(observed.Goal.Status) ||
		observed.Goal.ClosureAccepted ||
		goalFirstResidentObservedGoalStatusTerminalV0(observed.Goal.ClosureStatus)
}

func goalFirstResidentObservedGoalStatusTerminalV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case orquestagoal.GoalStatusCompleteV0, orquestagoal.GoalStatusAcceptedV0:
		return true
	default:
		return false
	}
}

func goalFirstResidentObservedGoalEvidenceRefsV0(
	observed orquestamcp.MCPDirectorStatsToolResultV0,
) []string {
	if observed.Goal == nil {
		return []string{}
	}
	return compactStringsV0(observed.Goal.EvidenceRefs)
}

func goalFirstResidentObservedArtifactRefsV0(
	observed orquestamcp.MCPDirectorStatsToolResultV0,
) []string {
	if observed.Goal == nil {
		return []string{}
	}
	return compactStringsV0(observed.Goal.ArtifactRefs)
}

func goalFirstResidentObservedGoalRefV0(
	observed orquestamcp.MCPDirectorStatsToolResultV0,
) string {
	if observed.Goal == nil {
		return ""
	}
	return strings.TrimSpace(observed.Goal.GoalRef)
}

func goalFirstResidentObservedExternalGoalRefV0(
	observed orquestamcp.MCPDirectorStatsToolResultV0,
) string {
	if observed.Goal == nil {
		return ""
	}
	return strings.TrimSpace(observed.Goal.ExternalGoalRef)
}

func goalFirstResidentExistingReworkRunRefV0(state orquestagoal.GoalWorkStateV0) string {
	for _, ref := range state.EvidenceRefs {
		ref = strings.TrimSpace(ref)
		if strings.HasPrefix(ref, goalFirstResidentReworkExistingEvidencePrefixV0) {
			return strings.TrimSpace(strings.TrimPrefix(ref, goalFirstResidentReworkExistingEvidencePrefixV0))
		}
	}
	return ""
}

func goalFirstResidentReworkSpecV0(
	state orquestagoal.GoalWorkStateV0,
	reason string,
	evidenceRefs []string,
) orquestagoal.GoalWorkSpecV0 {
	sourceRunRef := strings.TrimSpace(state.RunRef)
	sourceGoalRef := strings.TrimSpace(state.GoalRef)
	suffix := codexStackDeterministicDigestV0(sourceRunRef, sourceGoalRef, reason)[:16]
	spec := orquestagoal.NormalizeGoalWorkSpecV0(state.Spec)
	spec.RunRef = sourceRunRef + "-rework-" + suffix
	spec.GoalRef = sourceGoalRef + "-rework-" + suffix
	if strings.TrimSpace(spec.RequestRef) != "" {
		spec.RequestRef = strings.TrimSpace(spec.RequestRef) + "-rework-" + suffix
	} else {
		spec.RequestRef = spec.RunRef
	}
	spec.Objective = strings.TrimSpace(spec.Objective) + "\n\nRework acotado: continuar desde artefactos y checkpoints existentes, no repetir lecturas amplias, producir el siguiente artefacto o receipt verificable, o cerrar blocked con causa concreta."
	spec.ContextRefs = append(spec.ContextRefs,
		orquestagoal.GoalContextRefV0{Kind: "source_run", Ref: sourceRunRef, Purpose: "goal-first resident rework source", Required: true},
		orquestagoal.GoalContextRefV0{Kind: "source_goal", Ref: sourceGoalRef, Purpose: "goal-first resident rework source", Required: true},
		orquestagoal.GoalContextRefV0{Kind: "rework_reason", Ref: reason, Purpose: "goal-first terminal rework state", Required: true},
	)
	spec.AcceptanceCriteria = compactStringsV0(append(
		spec.AcceptanceCriteria,
		"Debe reutilizar artefactos/checkpoints existentes antes de releer contexto amplio.",
		"Debe materializar un artefacto, receipt o bloqueo terminal verificable.",
	))
	spec.EvidenceRefs = compactStringsV0(append(spec.EvidenceRefs, evidenceRefs...))
	spec.EvidenceRefs = compactStringsV0(append(spec.EvidenceRefs,
		goalFirstResidentReworkPreparedEvidenceRefV0,
		"evidence-ref-goal-first-resident-rework-source:"+sourceRunRef,
	))
	spec.ReworkPolicy.PreserveArtifacts = true
	return spec
}

func goalFirstResidentReworkResultV0(
	result orquestamcp.MCPRunSupervisorToolResultV0,
	sourceRunRef string,
	reworkRunRef string,
	reason string,
	evidenceRefs []string,
	existing bool,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	result.StopReason = "goal_first_resident_rework_prepared"
	result.RepairRunRefs = compactStringsV0(append(result.RepairRunRefs, reworkRunRef))
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, evidenceRefs...))
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, goalFirstResidentReworkPreparedEvidenceRefV0))
	result.Last.EvidenceRefs = compactStringsV0(append(result.Last.EvidenceRefs, result.EvidenceRefs...))
	result.NextActions = compactStringsV0(append(
		[]string{
			"observe_goal_rework_followup",
			"do_not_relaunch_source_goal",
		},
		result.NextActions...,
	))
	code := "goal_first_resident_rework_prepared"
	if existing {
		code = "goal_first_resident_rework_already_prepared"
	}
	result.Diagnostics = append(result.Diagnostics, orquestamcp.MCPAutoprogrammingDiagnosticV0{
		Code:         code,
		Scope:        "run:" + strings.TrimSpace(sourceRunRef),
		Message:      "goal-first residente preparo rework acotado por " + strings.TrimSpace(reason),
		EvidenceRefs: compactStringsV0(append(evidenceRefs, goalFirstResidentReworkPreparedEvidenceRefV0)),
	})
	return result
}
