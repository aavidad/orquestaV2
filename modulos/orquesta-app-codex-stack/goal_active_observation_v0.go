package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	activeGoalObservationFanoutV0           = 4
	activeGoalObservationDeadlineIssueV0    = "observe_goal_deadline_exceeded"
	activeGoalObservationRunInFlightIssueV0 = "goal_observation_run_in_flight"
)

type activeGoalObservationOutcomeV0 struct {
	observation *orquestagoal.GoalWorkObserveResultV0
	issues      []orquestagoal.GoalWorkObserveActiveIssueV0
}

type activeGoalObservationIndexedOutcomeV0 struct {
	index   int
	outcome activeGoalObservationOutcomeV0
}

func (stack *StackV0) ObserveActiveGoalWorksV0(ctx context.Context, request orquestagoal.GoalWorkObserveActiveRequestV0) (orquestagoal.GoalWorkObserveActiveResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
	if len(states) == 0 {
		return orquestagoal.GoalWorkObserveActiveResultV0{}, nil
	}

	outcomes := make([]activeGoalObservationOutcomeV0, len(states))
	received := make([]bool, len(states))
	jobs := make(chan int)
	results := make(chan activeGoalObservationIndexedOutcomeV0, len(states))
	workers := activeGoalObservationFanoutV0
	if workers > len(states) {
		workers = len(states)
	}
	for worker := 0; worker < workers; worker++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					if ctx.Err() != nil {
						return
					}
					results <- activeGoalObservationIndexedOutcomeV0{
						index:   index,
						outcome: stack.observeActiveGoalWorkStateV0(ctx, states[index], listRequest),
					}
				}
			}
		}()
	}
	launched := 0
scheduling:
	for launched < len(states) {
		select {
		case <-ctx.Done():
			break scheduling
		case jobs <- launched:
			launched++
		}
	}
	close(jobs)
	completed := 0
	for completed < launched {
		select {
		case result := <-results:
			if !received[result.index] {
				outcomes[result.index] = result.outcome
				received[result.index] = true
				completed++
			}
		case <-ctx.Done():
			for {
				select {
				case result := <-results:
					if !received[result.index] {
						outcomes[result.index] = result.outcome
						received[result.index] = true
						completed++
					}
				default:
					for index := range states {
						if !received[index] {
							outcomes[index] = activeGoalObservationDeadlineOutcomeV0(states[index])
						}
					}
					return foldActiveGoalObservationOutcomesV0(outcomes), ctx.Err()
				}
			}
		}
	}
	return foldActiveGoalObservationOutcomesV0(outcomes), nil
}

func activeGoalObservationDeadlineOutcomeV0(state orquestagoal.GoalWorkStateV0) activeGoalObservationOutcomeV0 {
	return activeGoalObservationOutcomeV0{issues: []orquestagoal.GoalWorkObserveActiveIssueV0{{
		RunRef:  strings.TrimSpace(state.RunRef),
		GoalRef: strings.TrimSpace(state.GoalRef),
		Code:    activeGoalObservationDeadlineIssueV0,
		Field:   "run_ref",
	}}}
}

func foldActiveGoalObservationOutcomesV0(outcomes []activeGoalObservationOutcomeV0) orquestagoal.GoalWorkObserveActiveResultV0 {
	out := orquestagoal.GoalWorkObserveActiveResultV0{}
	for _, outcome := range outcomes {
		out.Issues = append(out.Issues, outcome.issues...)
		if outcome.observation == nil {
			continue
		}
		out.Observations = append(out.Observations, *outcome.observation)
		out.EvidenceRefs = compactStringsV0(append(out.EvidenceRefs, outcome.observation.EvidenceRefs...))
	}
	return out
}

func (stack *StackV0) observeActiveGoalWorkStateV0(ctx context.Context, state orquestagoal.GoalWorkStateV0, listRequest orquestagoal.GoalWorkStateListRequestV0) activeGoalObservationOutcomeV0 {
	out := activeGoalObservationOutcomeV0{}
	issue := func(code, field, message string) activeGoalObservationOutcomeV0 {
		out.issues = append(out.issues, orquestagoal.GoalWorkObserveActiveIssueV0{RunRef: strings.TrimSpace(state.RunRef), GoalRef: strings.TrimSpace(state.GoalRef), Code: code, Field: field, Message: message})
		return out
	}
	if repaired, ok, err := orquestaappdirectorservice.ReconcileAppDirectorGoalReworkStateFromMarkerV0(ctx, state, stack.Ports); err != nil {
		return issue("goal_rework_state_reconcile_failed", "goal_state", err.Error())
	} else if ok {
		state = repaired
	}
	if stack.goalFirstAutoprogrammingPromotionRecoveryPendingV0(state) {
		complete, err := stack.recoverGoalFirstAutoprogrammingPromotionV0(ctx, state)
		if err != nil {
			return issue("goal_first_promotion_recovery_failed", "autoprogramming_promotion", err.Error())
		}
		if !complete {
			out.issues = append(out.issues, orquestagoal.GoalWorkObserveActiveIssueV0{RunRef: strings.TrimSpace(state.RunRef), GoalRef: strings.TrimSpace(state.GoalRef), Code: "goal_first_promotion_recovery_pending", Field: "autoprogramming_promotion"})
		}
		if refreshed, err := stack.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, state.RunRef); err == nil {
			state = refreshed
		}
	}
	if !orquestagoal.GoalWorkStatePendingObservationV0(state) {
		if !orquestagoal.GoalWorkStateShouldReturnActiveSnapshotV0(state, listRequest) {
			return out
		}
		observed, err := orquestagoal.GoalWorkObservationSnapshotFromStateV0(state)
		if err != nil {
			return issue("goal_state_snapshot_invalid", "goal_state", err.Error())
		}
		out.observation = &observed
		return out
	}
	runRef := strings.TrimSpace(state.RunRef)
	if runRef == "" {
		return issue("goal_state_run_ref_missing", "run_ref", "")
	}
	coordinator := stack.goalFirstObservationCoordinatorV0()
	if coordinator == nil {
		return issue("goal_first_observation_coordinator_unavailable", "run_ref", "")
	}
	release, acquired := coordinator.tryAcquireV0(runRef)
	if !acquired {
		return issue(activeGoalObservationRunInFlightIssueV0, "run_ref", "")
	}
	defer release()
	observed, err := stack.observeAppDirectorGoalSerializedV0(ctx, orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{RunRef: runRef, RequestedBy: "orquesta-app-codex-stack-active-goal-observer"})
	if err != nil {
		return issue("observe_goal_failed", "run_ref", err.Error())
	}
	persisted, err := stack.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return issue("goal_state_load_after_observe_failed", "run_ref", err.Error())
	}
	observation := stackGoalActiveObservationFromAppDirectorV0(persisted, observed)
	out.observation = &observation
	return out
}

func (stack *StackV0) goalFirstAutoprogrammingPromotionRecoveryPendingV0(
	state orquestagoal.GoalWorkStateV0,
) bool {
	return strings.TrimSpace(state.Status) == orquestagoal.GoalStatusCompleteV0 &&
		state.LastClosure != nil && state.LastClosure.Accepted &&
		strings.TrimSpace(state.Spec.WorkKind) == "autoprogramming" &&
		!autoprogrammingGoalFirstPromotionCompletionVerifiedV0(state, state.RunRef)
}

func (stack *StackV0) recoverGoalFirstAutoprogrammingPromotionV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (bool, error) {
	if stack.Ports.RunStore == nil {
		return false, fmt.Errorf("run_store requerido para recuperar promocion goal-first")
	}
	run, err := stack.Ports.RunStore.LoadRunV0(ctx, state.RunRef)
	if err != nil {
		return false, err
	}
	complete, refs, err := stack.maybePromoteClosedAutoprogrammingRunV0(ctx, run)
	if err != nil || !complete {
		return false, err
	}
	result := orquestaappdirectorservice.ObserveAppDirectorGoalResultV0{
		SchemaVersion:         orquestaappdirectorservice.ObserveAppDirectorGoalResultSchemaV0,
		Status:                state.Status,
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
		RunRef:                state.RunRef,
		GoalRef:               state.GoalRef,
		ExternalGoalRef:       state.ExternalGoalRef,
		Run:                   run,
		EvidenceRefs:          compactStringsV0(append(state.EvidenceRefs, refs...)),
	}
	if state.LastResult != nil {
		result.GoalResult = *state.LastResult
	}
	if state.LastClosure != nil {
		result.Closure = *state.LastClosure
	}
	if err := stack.syncGoalFirstQueueAfterObservationV0(
		ctx,
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{RunRef: state.RunRef},
		result,
	); err != nil {
		return false, err
	}
	return true, nil
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
