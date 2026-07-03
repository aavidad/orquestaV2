package orquestaserver

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	goalObserverHighConsumptionCheckpointOnlyReasonV0 = "checkpoint_only_high_consumption"
	goalObserverHighConsumptionNoCheckpointReasonV0   = "goal_active_no_checkpoint_high_consumption"
	goalObserverHighConsumptionRecommendedActionV0    = "replan_narrow_context"

	goalObserverHighConsumptionRequestedByV0       = "orquesta-server-goal-observer"
	goalObserverHighConsumptionStopEvidenceV0      = "evidence-ref-goal-observer-high-consumption-stop-requested"
	goalObserverCheckpointOnlyHighConsumptionRefV0 = "evidence-ref-goal-observer-checkpoint-only-high-consumption"
	goalObserverNoCheckpointHighConsumptionRefV0   = "evidence-ref-goal-observer-no-checkpoint-high-consumption"
)

func (runtime *RuntimeV0) reconcileGoalObserverHighConsumptionV0(
	ctx context.Context,
	result orquestagoal.GoalWorkObserveActiveResultV0,
) orquestagoal.GoalWorkObserveActiveResultV0 {
	if runtime == nil || runtime.goalStateStore == nil || len(result.Observations) == 0 {
		return result
	}
	for index := range result.Observations {
		observation := result.Observations[index]
		reason, evidenceRefs, ok := goalObserverHighConsumptionReasonV0(observation)
		if !ok {
			continue
		}
		blocked := runtime.blockGoalObserverHighConsumptionV0(ctx, observation, reason, evidenceRefs)
		result.Observations[index].Result = blocked
		result.Observations[index].State.Status = blocked.Status
		result.Observations[index].State.LastResult = &blocked
		result.Observations[index].State.EvidenceRefs = compactConfigStringsV0(
			append(result.Observations[index].State.EvidenceRefs, blocked.EvidenceRefs...),
		)
		result.Observations[index].Terminal = true
		result.Observations[index].Accepted = false
		result.Observations[index].NeedsRework = true
		result.Observations[index].EvidenceRefs = compactConfigStringsV0(
			append(result.Observations[index].EvidenceRefs, blocked.EvidenceRefs...),
		)
		result.EvidenceRefs = compactConfigStringsV0(append(result.EvidenceRefs, blocked.EvidenceRefs...))
		result.Issues = append(result.Issues, orquestagoal.GoalWorkObserveActiveIssueV0{
			RunRef:  strings.TrimSpace(result.Observations[index].State.RunRef),
			GoalRef: strings.TrimSpace(blocked.GoalRef),
			Code:    reason,
			Field:   "goal_progress",
			Message: goalObserverHighConsumptionRecommendedActionV0,
		})
	}
	return result
}

func goalObserverHighConsumptionReasonV0(
	observation orquestagoal.GoalWorkObserveResultV0,
) (string, []string, bool) {
	result := orquestagoal.NormalizeGoalWorkResultV0(observation.Result)
	if strings.TrimSpace(result.Status) != orquestagoal.GoalStatusRunningV0 {
		return "", nil, false
	}
	if !goalObserverHighConsumptionObservedV0(observation, result) {
		return "", nil, false
	}
	if goalObserverHasDomainReceiptV0(result) || goalObserverHasNonCheckpointArtifactV0(result) {
		return "", nil, false
	}
	evidenceRefs := goalObserverHighConsumptionEvidenceRefsV0(observation, result)
	if goalObserverHasCheckpointArtifactV0(result) ||
		goalObserverHasIssueOrEvidenceV0(observation, result, goalObserverHighConsumptionCheckpointOnlyReasonV0) {
		return goalObserverHighConsumptionCheckpointOnlyReasonV0,
			compactConfigStringsV0(append(evidenceRefs, goalObserverCheckpointOnlyHighConsumptionRefV0)),
			true
	}
	if goalObserverHasIssueOrEvidenceV0(observation, result, goalObserverHighConsumptionNoCheckpointReasonV0) ||
		!goalObserverHasAnyArtifactV0(result) {
		return goalObserverHighConsumptionNoCheckpointReasonV0,
			compactConfigStringsV0(append(evidenceRefs, goalObserverNoCheckpointHighConsumptionRefV0)),
			true
	}
	return "", nil, false
}

func goalObserverHighConsumptionObservedV0(
	observation orquestagoal.GoalWorkObserveResultV0,
	result orquestagoal.GoalWorkResultV0,
) bool {
	if goalObserverHasIssueOrEvidenceV0(observation, result, goalObserverHighConsumptionCheckpointOnlyReasonV0) ||
		goalObserverHasIssueOrEvidenceV0(observation, result, goalObserverHighConsumptionNoCheckpointReasonV0) {
		return true
	}
	for _, value := range append(append([]string{}, result.EvidenceRefs...), observation.EvidenceRefs...) {
		if strings.TrimSpace(value) == "evidence-ref-codex-app-server-goal-high-token-usage" {
			return true
		}
	}
	return strings.Contains(
		strings.TrimSpace(result.Summary),
		"codex_app_server_goal_status_active_high_token_usage",
	)
}

func goalObserverHasIssueOrEvidenceV0(
	observation orquestagoal.GoalWorkObserveResultV0,
	result orquestagoal.GoalWorkResultV0,
	code string,
) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for _, issue := range result.Issues {
		if strings.TrimSpace(issue.Code) == code {
			return true
		}
	}
	if observation.Closure.Status != "" {
		for _, issue := range observation.Closure.Issues {
			if strings.TrimSpace(issue.Code) == code {
				return true
			}
		}
	}
	for _, ref := range goalObserverHighConsumptionEvidenceRefsV0(observation, result) {
		trimmed := strings.TrimSpace(ref)
		if strings.Contains(trimmed, code) ||
			strings.Contains(trimmed, strings.ReplaceAll(code, "_", "-")) {
			return true
		}
	}
	return strings.Contains(strings.TrimSpace(result.Summary), code)
}

func goalObserverHighConsumptionEvidenceRefsV0(
	observation orquestagoal.GoalWorkObserveResultV0,
	result orquestagoal.GoalWorkResultV0,
) []string {
	refs := []string{}
	refs = append(refs, observation.State.EvidenceRefs...)
	refs = append(refs, observation.EvidenceRefs...)
	refs = append(refs, result.EvidenceRefs...)
	refs = append(refs, result.Checklist.EvidenceRefs...)
	if observation.Closure.Status != "" {
		refs = append(refs, observation.Closure.EvidenceRefs...)
	}
	for _, artifact := range result.MaterializedArtifacts {
		refs = append(refs, artifact.EvidenceRefs...)
	}
	return compactConfigStringsV0(refs)
}

func goalObserverHasDomainReceiptV0(result orquestagoal.GoalWorkResultV0) bool {
	return len(compactConfigStringsV0(result.DomainReceiptRefs)) > 0
}

func goalObserverHasAnyArtifactV0(result orquestagoal.GoalWorkResultV0) bool {
	return len(compactConfigStringsV0(result.ArtifactRefs)) > 0 ||
		len(compactConfigStringsV0(result.ArtifactPaths)) > 0 ||
		len(result.MaterializedArtifacts) > 0
}

func goalObserverHasNonCheckpointArtifactV0(result orquestagoal.GoalWorkResultV0) bool {
	for _, ref := range result.ArtifactRefs {
		if !goalObserverArtifactLooksLikeCheckpointV0(ref) {
			return true
		}
	}
	for _, path := range result.ArtifactPaths {
		if !goalObserverArtifactLooksLikeCheckpointV0(path) {
			return true
		}
	}
	for _, artifact := range result.MaterializedArtifacts {
		if !goalObserverMaterializedArtifactLooksLikeCheckpointV0(artifact) {
			return true
		}
	}
	return false
}

func goalObserverHasCheckpointArtifactV0(result orquestagoal.GoalWorkResultV0) bool {
	for _, ref := range result.ArtifactRefs {
		if goalObserverArtifactLooksLikeCheckpointV0(ref) {
			return true
		}
	}
	for _, path := range result.ArtifactPaths {
		if goalObserverArtifactLooksLikeCheckpointV0(path) {
			return true
		}
	}
	for _, artifact := range result.MaterializedArtifacts {
		if goalObserverMaterializedArtifactLooksLikeCheckpointV0(artifact) {
			return true
		}
	}
	return false
}

func goalObserverMaterializedArtifactLooksLikeCheckpointV0(
	artifact orquestagoal.GoalMaterializedArtifactV0,
) bool {
	return goalObserverArtifactLooksLikeCheckpointV0(strings.Join([]string{
		artifact.ArtifactRef,
		artifact.Path,
		artifact.ArtifactType,
		artifact.Scope,
	}, "|"))
}

func goalObserverArtifactLooksLikeCheckpointV0(value string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(value)), "checkpoint")
}

func (runtime *RuntimeV0) blockGoalObserverHighConsumptionV0(
	ctx context.Context,
	observation orquestagoal.GoalWorkObserveResultV0,
	reason string,
	evidenceRefs []string,
) orquestagoal.GoalWorkResultV0 {
	result := orquestagoal.NormalizeGoalWorkResultV0(observation.Result)
	if strings.TrimSpace(result.GoalRef) == "" {
		result.GoalRef = strings.TrimSpace(observation.State.GoalRef)
	}
	if strings.TrimSpace(result.ExternalGoalRef) == "" {
		result.ExternalGoalRef = strings.TrimSpace(observation.State.ExternalGoalRef)
	}
	blocked := result
	blocked.Status = orquestagoal.GoalStatusBlockedV0
	blocked.Summary = reason
	blocked.ReworkPlanRefs = compactConfigStringsV0(append(
		blocked.ReworkPlanRefs,
		"rework-plan-ref-"+strings.ReplaceAll(reason, "_", "-"),
	))
	blocked.EvidenceRefs = compactConfigStringsV0(append(blocked.EvidenceRefs, evidenceRefs...))
	blocked.Issues = append(blocked.Issues, orquestagoal.GoalWorkIssueV0{
		Code:   reason,
		Field:  "goal_progress",
		Detail: "recommended_action=" + goalObserverHighConsumptionRecommendedActionV0,
	})
	stopEvidence, stopIssues := runtime.requestGoalObserverHighConsumptionStopV0(ctx, observation, blocked, reason)
	blocked.EvidenceRefs = compactConfigStringsV0(append(blocked.EvidenceRefs, stopEvidence...))
	blocked.Issues = append(blocked.Issues, stopIssues...)
	blocked = orquestagoal.NormalizeGoalWorkResultV0(blocked)
	runtime.persistGoalObserverHighConsumptionStateV0(ctx, observation.State, blocked, reason)
	return blocked
}

func (runtime *RuntimeV0) requestGoalObserverHighConsumptionStopV0(
	ctx context.Context,
	observation orquestagoal.GoalWorkObserveResultV0,
	result orquestagoal.GoalWorkResultV0,
	reason string,
) ([]string, []orquestagoal.GoalWorkIssueV0) {
	runRef := strings.TrimSpace(observation.State.RunRef)
	if runRef == "" {
		return []string{"evidence-ref-goal-observer-high-consumption-run-ref-missing"}, []orquestagoal.GoalWorkIssueV0{{
			Code:  "goal_observer_high_consumption_run_ref_missing",
			Field: "run_ref",
		}}
	}
	if runtime == nil || runtime.goalStopper == nil {
		return []string{"evidence-ref-goal-observer-high-consumption-stop-port-unavailable"}, []orquestagoal.GoalWorkIssueV0{{
			Code:  "goal_observer_high_consumption_stop_port_unavailable",
			Field: "goal_stopper",
		}}
	}
	stopResult, err := runtime.goalStopper.RequestGoalCooperativeStopV0(ctx, GoalCooperativeStopRequestV0{
		RunRef:            runRef,
		GoalRef:           strings.TrimSpace(result.GoalRef),
		ExternalGoalRef:   strings.TrimSpace(result.ExternalGoalRef),
		Reason:            reason,
		RecommendedAction: goalObserverHighConsumptionRecommendedActionV0,
		RequestedBy:       goalObserverHighConsumptionRequestedByV0,
		IdempotencyKey:    "idem-goal-observer-high-consumption-" + serverGoalProgressSafeRefPartV0(runRef),
		EvidenceRefs: compactConfigStringsV0(append(
			result.EvidenceRefs,
			goalObserverHighConsumptionStopEvidenceV0,
		)),
	})
	if err != nil {
		return []string{"evidence-ref-goal-observer-high-consumption-stop-request-failed"}, []orquestagoal.GoalWorkIssueV0{{
			Code:   "goal_observer_high_consumption_stop_request_failed",
			Field:  "goal_stopper",
			Detail: err.Error(),
		}}
	}
	refs := append([]string(nil), stopResult.EvidenceRefs...)
	if stopResult.Requested {
		refs = append(refs, goalObserverHighConsumptionStopEvidenceV0)
	}
	return compactConfigStringsV0(refs), nil
}

func (runtime *RuntimeV0) persistGoalObserverHighConsumptionStateV0(
	ctx context.Context,
	observedState orquestagoal.GoalWorkStateV0,
	blocked orquestagoal.GoalWorkResultV0,
	reason string,
) {
	if runtime == nil || runtime.goalStateStore == nil {
		return
	}
	runRef := strings.TrimSpace(observedState.RunRef)
	if runRef == "" {
		return
	}
	state := observedState
	if loaded, err := runtime.goalStateStore.LoadGoalWorkStateV0(ctx, runRef); err == nil {
		state = loaded
	}
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &blocked
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusBlockedV0,
		NeedsRework:  true,
		EvidenceRefs: append([]string(nil), blocked.EvidenceRefs...),
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  reason,
			Field: "goal_progress",
		}},
	}
	state.EvidenceRefs = compactConfigStringsV0(append(state.EvidenceRefs, blocked.EvidenceRefs...))
	_ = runtime.goalStateStore.SaveGoalWorkStateV0(ctx, state)
}
