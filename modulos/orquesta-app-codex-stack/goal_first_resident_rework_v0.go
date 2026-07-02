package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

const (
	goalFirstResidentReworkPreparedEvidenceRefV0    = "evidence-ref-goal-first-resident-rework-prepared"
	goalFirstResidentReworkExistingEvidencePrefixV0 = "evidence-ref-goal-first-resident-rework-goal:"
	goalFirstResidentReworkReasonCheckpointOnlyV0   = "checkpoint_only_high_consumption"
	goalFirstResidentReworkReasonNoCheckpointV0     = "goal_active_no_checkpoint_high_consumption"
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
	evidenceRefs := goalFirstResidentHighConsumptionEvidenceRefsV0(state)
	if goalFirstResidentHasIssueOrEvidenceV0(state, goalFirstResidentReworkReasonCheckpointOnlyV0) {
		return goalFirstResidentReworkReasonCheckpointOnlyV0, evidenceRefs, true
	}
	if goalFirstResidentHasIssueOrEvidenceV0(state, goalFirstResidentReworkReasonNoCheckpointV0) {
		return goalFirstResidentReworkReasonNoCheckpointV0, evidenceRefs, true
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
	return false
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

func goalFirstResidentHighConsumptionEvidenceRefsV0(state orquestagoal.GoalWorkStateV0) []string {
	refs := []string{goalFirstResidentReworkPreparedEvidenceRefV0}
	for _, ref := range goalFirstResidentAllEvidenceRefsV0(state) {
		trimmed := strings.TrimSpace(ref)
		if strings.Contains(trimmed, "high-consumption") ||
			strings.Contains(trimmed, "high_consumption") ||
			strings.Contains(trimmed, "checkpoint-only") ||
			strings.Contains(trimmed, "no-checkpoint") {
			refs = append(refs, trimmed)
		}
	}
	return compactStringsV0(refs)
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
		orquestagoal.GoalContextRefV0{Kind: "rework_reason", Ref: reason, Purpose: "high consumption terminal state", Required: true},
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
