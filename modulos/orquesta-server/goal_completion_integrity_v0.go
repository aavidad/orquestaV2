package orquestaserver

import (
	"context"
	"regexp"
	"sort"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	idleSelfImprovementGoalCompletedWithoutResultReasonV0   = "goal_completed_without_materialized_result"
	idleSelfImprovementGoalSelfReportTaskMismatchReasonV0   = "goal_self_report_task_mismatch"
	idleSelfImprovementGoalMaterializeResultActionV0        = "materialize_result_or_review_closure"
	idleSelfImprovementGoalVerifyDiffAgainstBacklogActionV0 = "verify_diff_against_backlog_task"
	idleSelfImprovementGoalCompletedWithoutResultEvidenceV0 = "evidence-ref-goal-completed-without-materialized-result"
	idleSelfImprovementGoalSelfReportTaskMismatchEvidenceV0 = "evidence-ref-goal-self-report-task-mismatch"
)

var serverGoalTaskTokenPatternV0 = regexp.MustCompile(`(?i)\b(?:mej-task-\d+|t\d{3})\b`)

// reconcileIdleSelfImprovementGoalCompletionIntegrityV0 audita goals de
// automejora observados como complete: un cierre no aceptado y sin result
// durable deja una ventana ciega para cualquier observador por fichero, y un
// autoinforme que nombra otra tarea invalida el resumen como senal de que se
// hizo. Publica issues y evidencias sin alterar el status del goal.
func (runtime *RuntimeV0) reconcileIdleSelfImprovementGoalCompletionIntegrityV0(
	ctx context.Context,
	result orquestagoal.GoalWorkObserveActiveResultV0,
) orquestagoal.GoalWorkObserveActiveResultV0 {
	if runtime == nil || runtime.tracker == nil || len(result.Observations) == 0 {
		return result
	}
	state := runtime.tracker.SnapshotV0()
	index, observed, ok := idleSelfImprovementGoalObservationIndexV0(state, result)
	if !ok || observed.Status != orquestagoal.GoalStatusCompleteV0 {
		return result
	}
	issues := []orquestagoal.GoalWorkIssueV0{}
	evidenceRefs := []string{}
	if !result.Observations[index].Accepted && !idleSelfImprovementGoalHasMaterializedResultV0(observed) {
		issues = append(issues, orquestagoal.GoalWorkIssueV0{
			Code:   idleSelfImprovementGoalCompletedWithoutResultReasonV0,
			Field:  "goal_result",
			Detail: "recommended_action=" + idleSelfImprovementGoalMaterializeResultActionV0,
		})
		evidenceRefs = append(evidenceRefs, idleSelfImprovementGoalCompletedWithoutResultEvidenceV0)
	}
	if mismatch, detail := idleSelfImprovementGoalSelfReportTaskMismatchV0(state, observed); mismatch {
		issues = append(issues, orquestagoal.GoalWorkIssueV0{
			Code:   idleSelfImprovementGoalSelfReportTaskMismatchReasonV0,
			Field:  "goal_summary",
			Detail: detail + " recommended_action=" + idleSelfImprovementGoalVerifyDiffAgainstBacklogActionV0,
		})
		evidenceRefs = append(evidenceRefs, idleSelfImprovementGoalSelfReportTaskMismatchEvidenceV0)
	}
	if len(issues) == 0 {
		return result
	}
	audited := observed
	audited.Issues = append(audited.Issues, issues...)
	audited.EvidenceRefs = compactConfigStringsV0(append(audited.EvidenceRefs, evidenceRefs...))
	audited = orquestagoal.NormalizeGoalWorkResultV0(audited)
	result.Observations[index].Result = audited
	result.Observations[index].State.LastResult = &audited
	result.Observations[index].State.EvidenceRefs = compactConfigStringsV0(
		append(result.Observations[index].State.EvidenceRefs, evidenceRefs...),
	)
	result.Observations[index].EvidenceRefs = compactConfigStringsV0(
		append(result.Observations[index].EvidenceRefs, evidenceRefs...),
	)
	result.EvidenceRefs = compactConfigStringsV0(append(result.EvidenceRefs, evidenceRefs...))
	runRef := strings.TrimSpace(result.Observations[index].State.RunRef)
	for _, issue := range issues {
		result.Issues = append(result.Issues, orquestagoal.GoalWorkObserveActiveIssueV0{
			RunRef:  runRef,
			GoalRef: strings.TrimSpace(audited.GoalRef),
			Code:    issue.Code,
			Field:   issue.Field,
			Message: issue.Detail,
		})
	}
	runtime.persistIdleSelfImprovementGoalIntegrityStateV0(ctx, result.Observations[index].State, audited)
	return result
}

func (runtime *RuntimeV0) persistIdleSelfImprovementGoalIntegrityStateV0(
	ctx context.Context,
	observedState orquestagoal.GoalWorkStateV0,
	audited orquestagoal.GoalWorkResultV0,
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
	state.LastResult = &audited
	state.EvidenceRefs = compactConfigStringsV0(append(state.EvidenceRefs, audited.EvidenceRefs...))
	_ = runtime.goalStateStore.SaveGoalWorkStateV0(ctx, state)
}

func idleSelfImprovementGoalHasMaterializedResultV0(result orquestagoal.GoalWorkResultV0) bool {
	candidates := append([]string(nil), result.ArtifactPaths...)
	for _, artifact := range result.MaterializedArtifacts {
		candidates = append(candidates, artifact.Path, artifact.ArtifactRef, artifact.ArtifactType)
	}
	for _, candidate := range candidates {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, "orquesta_goal_result_") || strings.Contains(candidate, "goal-result") {
			return true
		}
	}
	return false
}

func idleSelfImprovementGoalSelfReportTaskMismatchV0(
	state StateV0,
	result orquestagoal.GoalWorkResultV0,
) (bool, string) {
	claimed := serverGoalTaskTokensV0(result.Summary)
	if len(claimed) == 0 {
		return false, ""
	}
	expectedSources := []string{result.GoalRef, result.ExternalGoalRef}
	if state.IdleSelfImprovementGoalSpec != nil {
		spec := orquestagoal.NormalizeGoalWorkSpecV0(*state.IdleSelfImprovementGoalSpec)
		expectedSources = append(expectedSources, spec.GoalRef, spec.RunRef, spec.Objective)
	}
	if message := state.IdleSelfImprovementOperationalMessage; message != nil {
		expectedSources = append(expectedSources, message.GoalRefs...)
	}
	expected := serverGoalTaskTokensV0(strings.Join(expectedSources, " "))
	if len(expected) == 0 {
		return false, ""
	}
	for token := range claimed {
		if _, ok := expected[token]; ok {
			return false, ""
		}
	}
	return true, "summary_tasks=" + serverGoalTaskTokensJoinedV0(claimed) +
		" expected_tasks=" + serverGoalTaskTokensJoinedV0(expected)
}

func serverGoalTaskTokensV0(text string) map[string]struct{} {
	tokens := map[string]struct{}{}
	for _, match := range serverGoalTaskTokenPatternV0.FindAllString(text, -1) {
		tokens[strings.ToLower(strings.TrimSpace(match))] = struct{}{}
	}
	return tokens
}

func serverGoalTaskTokensJoinedV0(tokens map[string]struct{}) string {
	values := make([]string, 0, len(tokens))
	for token := range tokens {
		values = append(values, token)
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}
