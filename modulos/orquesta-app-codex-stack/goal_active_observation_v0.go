package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func (stack *StackV0) ObserveActiveGoalWorksV0(
	ctx context.Context,
	request orquestagoal.GoalWorkObserveActiveRequestV0,
) (orquestagoal.GoalWorkObserveActiveResultV0, error) {
	if stack == nil {
		return orquestagoal.GoalWorkObserveActiveResultV0{}, fmt.Errorf("stack requerido")
	}
	if stack.Ports.GoalStateStore == nil {
		return orquestagoal.GoalWorkObserveActiveResultV0{}, orquestagoal.GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_store"}
	}
	lister, ok := stack.Ports.GoalStateStore.(orquestagoal.GoalWorkStateListPortV0)
	if !ok {
		return orquestagoal.GoalWorkObserveActiveResultV0{}, orquestagoal.GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_lister"}
	}
	listRequest := stackActiveGoalWorkStateListRequestV0(request.List)
	states, err := lister.ListGoalWorkStatesV0(ctx, listRequest)
	if err != nil {
		return orquestagoal.GoalWorkObserveActiveResultV0{}, err
	}
	out := orquestagoal.GoalWorkObserveActiveResultV0{}
	for _, state := range states {
		if !orquestagoal.GoalWorkStatePendingObservationV0(state) {
			if !orquestagoal.GoalWorkStateShouldReturnActiveSnapshotV0(state, listRequest) {
				continue
			}
			observed, err := orquestagoal.GoalWorkObservationSnapshotFromStateV0(state)
			if err != nil {
				out.Issues = append(out.Issues, orquestagoal.GoalWorkObserveActiveIssueV0{
					RunRef:  strings.TrimSpace(state.RunRef),
					GoalRef: strings.TrimSpace(state.GoalRef),
					Code:    "goal_state_snapshot_invalid",
					Field:   "goal_state",
					Message: err.Error(),
				})
				continue
			}
			out.Observations = append(out.Observations, observed)
			out.EvidenceRefs = compactStringsV0(append(out.EvidenceRefs, observed.EvidenceRefs...))
			continue
		}
		runRef := strings.TrimSpace(state.RunRef)
		if runRef == "" {
			out.Issues = append(out.Issues, orquestagoal.GoalWorkObserveActiveIssueV0{
				GoalRef: strings.TrimSpace(state.GoalRef),
				Code:    "goal_state_run_ref_missing",
				Field:   "run_ref",
			})
			continue
		}
		observed, err := stack.ObserveAppDirectorGoalV0(
			ctx,
			orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
				RunRef:      runRef,
				RequestedBy: "orquesta-app-codex-stack-active-goal-observer",
			},
		)
		if err != nil {
			out.Issues = append(out.Issues, orquestagoal.GoalWorkObserveActiveIssueV0{
				RunRef:  runRef,
				GoalRef: strings.TrimSpace(state.GoalRef),
				Code:    "observe_goal_failed",
				Field:   "run_ref",
				Message: err.Error(),
			})
			continue
		}
		persisted, loadErr := stack.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
		if loadErr != nil {
			out.Issues = append(out.Issues, orquestagoal.GoalWorkObserveActiveIssueV0{
				RunRef:  runRef,
				GoalRef: strings.TrimSpace(observed.GoalRef),
				Code:    "goal_state_load_after_observe_failed",
				Field:   "run_ref",
				Message: loadErr.Error(),
			})
			continue
		}
		observation := stackGoalActiveObservationFromAppDirectorV0(persisted, observed)
		out.Observations = append(out.Observations, observation)
		out.EvidenceRefs = compactStringsV0(append(out.EvidenceRefs, observation.EvidenceRefs...))
	}
	return out, nil
}

func stackActiveGoalWorkStateListRequestV0(
	request orquestagoal.GoalWorkStateListRequestV0,
) orquestagoal.GoalWorkStateListRequestV0 {
	request = orquestagoal.NormalizeGoalWorkStateListRequestV0(request)
	if len(request.Statuses) == 0 {
		request.ActiveOnly = false
		request.Statuses = []string{
			orquestagoal.GoalStatusRunningV0,
			orquestagoal.GoalStatusCompleteV0,
			orquestagoal.GoalStatusBlockedV0,
			orquestagoal.GoalStatusInvalidV0,
		}
	}
	return orquestagoal.NormalizeGoalWorkStateListRequestV0(request)
}

func stackGoalActiveObservationFromAppDirectorV0(
	state orquestagoal.GoalWorkStateV0,
	result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0,
) orquestagoal.GoalWorkObserveResultV0 {
	goalResult := orquestagoal.NormalizeGoalWorkResultV0(result.GoalResult)
	if strings.TrimSpace(goalResult.Status) == "" {
		goalResult.Status = strings.TrimSpace(state.Status)
	}
	if strings.TrimSpace(goalResult.GoalRef) == "" {
		goalResult.GoalRef = strings.TrimSpace(state.GoalRef)
	}
	if strings.TrimSpace(goalResult.ExternalGoalRef) == "" {
		goalResult.ExternalGoalRef = strings.TrimSpace(state.ExternalGoalRef)
	}
	terminal := orquestagoal.GoalWorkResultTerminalV0(goalResult.Status)
	return orquestagoal.GoalWorkObserveResultV0{
		State:            state,
		Result:           goalResult,
		Closure:          result.Closure,
		Terminal:         terminal,
		ClosureEvaluated: terminal,
		Accepted:         result.Closure.Accepted,
		NeedsRework:      result.Closure.NeedsRework,
		EvidenceRefs:     compactStringsV0(append(append([]string(nil), state.EvidenceRefs...), result.EvidenceRefs...)),
	}
}
