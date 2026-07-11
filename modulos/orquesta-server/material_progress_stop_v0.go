package orquestaserver

import (
	"context"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func (runtime *RuntimeV0) stopForMaterialProgressV0(
	ctx context.Context,
	observation orquestagoal.GoalWorkObserveResultV0,
	progress orquestaautoprogramming.MaterialProgressStateV0,
	code string,
	replan bool,
) orquestagoal.GoalWorkObserveResultV0 {
	if runtime.goalStopper == nil {
		observation.Result.Issues = append(observation.Result.Issues, orquestagoal.GoalWorkIssueV0{Code: code + "_stop_unavailable", Field: "goal_stopper"})
		return observation
	}
	stop, err := runtime.goalStopper.RequestGoalCooperativeStopV0(ctx, GoalCooperativeStopRequestV0{
		RunRef: observation.State.RunRef, GoalRef: observation.State.GoalRef,
		ExternalGoalRef: observation.State.ExternalGoalRef, RequireConfirmedBackendStop: true,
		Reason: code, RecommendedAction: code, RequestedBy: "orquesta-server-material-progress",
		IdempotencyKey: progress.LastActionIdempotencyKey,
		EvidenceRefs:   compactConfigStringsV0(append(progress.EvidenceRefs, progress.LastCheckpointRef)),
	})
	if err != nil || !stop.Requested {
		observation.Result.Issues = append(observation.Result.Issues, orquestagoal.GoalWorkIssueV0{Code: code + "_stop_unconfirmed", Field: "goal_stopper"})
		return observation
	}
	blocked := orquestagoal.NormalizeGoalWorkResultV0(observation.Result)
	blocked.Status = orquestagoal.GoalStatusBlockedV0
	blocked.Summary = code
	blocked.EvidenceRefs = compactConfigStringsV0(append(blocked.EvidenceRefs, stop.EvidenceRefs...))
	blocked.Issues = append(blocked.Issues, orquestagoal.GoalWorkIssueV0{Code: code, Field: "goal_progress"})
	if replan {
		blocked.ReworkPlanRefs = compactConfigStringsV0(append(blocked.ReworkPlanRefs, "rework-plan-ref-material-progress"))
	}
	state := observation.State
	if loaded, loadErr := runtime.goalStateStore.LoadGoalWorkStateV0(ctx, state.RunRef); loadErr == nil {
		state = loaded
	}
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &blocked
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status: orquestagoal.GoalStatusBlockedV0, NeedsRework: replan,
		EvidenceRefs: append([]string(nil), blocked.EvidenceRefs...),
		Issues:       []orquestagoal.GoalWorkIssueV0{{Code: code, Field: "goal_progress"}},
	}
	state.EvidenceRefs = compactConfigStringsV0(append(state.EvidenceRefs, blocked.EvidenceRefs...))
	if err := runtime.goalStateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		observation.Result.Issues = append(observation.Result.Issues, orquestagoal.GoalWorkIssueV0{Code: code + "_state_save_failed", Field: "goal_state"})
		observation.EvidenceRefs = compactConfigStringsV0(append(observation.EvidenceRefs, stop.EvidenceRefs...))
		return observation
	}
	observation.State = state
	observation.Result = blocked
	observation.Closure = *state.LastClosure
	observation.Terminal = true
	observation.ClosureEvaluated = true
	observation.Accepted = false
	observation.NeedsRework = replan
	observation.EvidenceRefs = compactConfigStringsV0(append(observation.EvidenceRefs, blocked.EvidenceRefs...))
	return observation
}
