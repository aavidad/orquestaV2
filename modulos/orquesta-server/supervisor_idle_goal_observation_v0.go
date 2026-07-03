package orquestaserver

import (
	"context"
	"errors"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	idleSelfImprovementGoalObserverUnavailableReasonV0 = "goal_observer_unavailable"
	idleSelfImprovementGoalRunningReasonV0             = "goal_running"
	idleSelfImprovementGoalCompletePendingClosureV0    = "goal_complete_pending_closure_validation"
	idleSelfImprovementGoalClosureAcceptedReasonV0     = "goal_closure_accepted"
	idleSelfImprovementGoalBlockedReasonV0             = "goal_blocked"
	idleSelfImprovementGoalBackendGoneWithoutResultV0  = IdleSelfImprovementGoalBackendGoneWithoutResultReasonV0
	idleSelfImprovementGoalInvalidReasonV0             = "goal_invalid"
	idleSelfImprovementGoalObservationErrorReasonV0    = "goal_observation_error"
)

func (runtime *RuntimeV0) observePendingIdleSelfImprovementGoalV0(ctx context.Context, now time.Time) bool {
	// Compatibility fallback for idle_self_improvement goal-first. When the
	// generic active-goal observer is available, it owns observation and tracker
	// projection; this path only keeps older/self-contained compositions working.
	if runtime == nil || !runtime.config.IdleSelfImprovementGoalFirst || runtime.tracker == nil {
		return false
	}
	if runtime.goalObservationAvailableV0() {
		return false
	}
	request, ok := idleSelfImprovementGoalObservationRequestFromStateV0(runtime.tracker.SnapshotV0())
	if !ok {
		return false
	}
	if result, ok := runtime.materializedIdleSelfImprovementGoalResultV0(ctx, request); ok {
		result = runtime.idleSelfImprovementResultWithFrozenTestGuardV0(result)
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkIdleSelfImprovementGoalObservedV0(result, nil, now),
			"idle_self_improvement_goal_materialized_result_observed",
		)
		return true
	}
	observer, ok := runtime.supervisor.(IdleSelfImprovementGoalObserverPortV0)
	if !ok || observer == nil {
		result := orquestagoal.GoalWorkResultV0{
			Status:          orquestagoal.GoalStatusInvalidV0,
			GoalRef:         request.GoalRef,
			ExternalGoalRef: request.ExternalGoalRef,
			Summary:         idleSelfImprovementGoalObserverUnavailableReasonV0,
		}
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkIdleSelfImprovementGoalObservedV0(
				result,
				errors.New(idleSelfImprovementGoalObserverUnavailableReasonV0),
				now,
			),
			"idle_self_improvement_goal_observer_unavailable",
		)
		return true
	}
	result, err := observer.ObserveGoalWorkV0(ctx, request)
	if strings.TrimSpace(result.GoalRef) == "" {
		result.GoalRef = request.GoalRef
	}
	if strings.TrimSpace(result.ExternalGoalRef) == "" {
		result.ExternalGoalRef = request.ExternalGoalRef
	}
	result = runtime.idleSelfImprovementResultWithFrozenTestGuardV0(result)
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkIdleSelfImprovementGoalObservedV0(result, err, now),
		"idle_self_improvement_goal_observed",
	)
	return true
}

func (runtime *RuntimeV0) materializedIdleSelfImprovementGoalResultV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, bool) {
	if runtime == nil || runtime.supervisor == nil {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	source, ok := runtime.supervisor.(IdleSelfImprovementMaterializedGoalResultPortV0)
	if !ok || source == nil {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	state := runtime.idleSelfImprovementGoalStateForObservationV0(ctx, request)
	loaded, err := source.LoadIdleSelfImprovementMaterializedGoalResultV0(
		ctx,
		IdleSelfImprovementMaterializedGoalResultRequestV0{
			GoalState:       state,
			GoalRef:         request.GoalRef,
			ExternalGoalRef: request.ExternalGoalRef,
		},
	)
	if err != nil || !loaded.Found {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	result := orquestagoal.NormalizeGoalWorkResultV0(loaded.Result)
	if !orquestagoal.GoalWorkResultTerminalV0(result.Status) ||
		len(orquestagoal.ValidateGoalWorkResultV0(result)) > 0 ||
		!idleSelfImprovementGoalResultMatchesObservationV0(result, request) {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	result.EvidenceRefs = compactConfigStringsV0(append(result.EvidenceRefs, loaded.EvidenceRefs...))
	return result, true
}

func (runtime *RuntimeV0) idleSelfImprovementGoalStateForObservationV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) orquestagoal.GoalWorkStateV0 {
	if runtime == nil || runtime.goalStateStore == nil {
		return orquestagoal.GoalWorkStateV0{}
	}
	snapshot := runtime.tracker.SnapshotV0()
	runRef := ""
	if snapshot.IdleSelfImprovementGoalSpec != nil {
		runRef = strings.TrimSpace(snapshot.IdleSelfImprovementGoalSpec.RunRef)
	}
	if runRef != "" {
		if state, err := runtime.goalStateStore.LoadGoalWorkStateV0(ctx, runRef); err == nil {
			return state
		}
	}
	lister, ok := runtime.goalStateStore.(orquestagoal.GoalWorkStateListPortV0)
	if !ok || lister == nil {
		return orquestagoal.GoalWorkStateV0{}
	}
	states, err := lister.ListGoalWorkStatesV0(ctx, orquestagoal.GoalWorkStateListRequestV0{
		Statuses: []string{
			orquestagoal.GoalStatusRunningV0,
			orquestagoal.GoalStatusCompleteV0,
			orquestagoal.GoalStatusBlockedV0,
			orquestagoal.GoalStatusInvalidV0,
		},
		MaxItems: 100,
	})
	if err != nil {
		return orquestagoal.GoalWorkStateV0{}
	}
	for _, state := range states {
		if idleSelfImprovementGoalStateMatchesObservationV0(state, request) {
			return state
		}
	}
	return orquestagoal.GoalWorkStateV0{}
}

func idleSelfImprovementGoalStateMatchesObservationV0(
	state orquestagoal.GoalWorkStateV0,
	request orquestagoal.GoalObservationRequestV0,
) bool {
	goalRef := strings.TrimSpace(request.GoalRef)
	externalGoalRef := strings.TrimSpace(request.ExternalGoalRef)
	return goalRef != "" && strings.TrimSpace(state.GoalRef) == goalRef ||
		externalGoalRef != "" && strings.TrimSpace(state.ExternalGoalRef) == externalGoalRef
}

func idleSelfImprovementGoalResultMatchesObservationV0(
	result orquestagoal.GoalWorkResultV0,
	request orquestagoal.GoalObservationRequestV0,
) bool {
	goalRef := strings.TrimSpace(request.GoalRef)
	if goalRef == "" {
		return false
	}
	if strings.TrimSpace(result.GoalRef) != goalRef {
		return false
	}
	externalGoalRef := strings.TrimSpace(request.ExternalGoalRef)
	return externalGoalRef == "" || strings.TrimSpace(result.ExternalGoalRef) == "" ||
		strings.TrimSpace(result.ExternalGoalRef) == externalGoalRef
}

func idleSelfImprovementGoalObservationRequestFromStateV0(
	state StateV0,
) (orquestagoal.GoalObservationRequestV0, bool) {
	message := state.IdleSelfImprovementOperationalMessage
	if message == nil || !idleSelfImprovementGoalObservationReasonV0(state.IdleSelfImprovementReason, message.ReasonCode) {
		return orquestagoal.GoalObservationRequestV0{}, false
	}
	refs := compactConfigStringsV0(message.GoalRefs)
	if len(refs) == 0 {
		return orquestagoal.GoalObservationRequestV0{}, false
	}
	request := orquestagoal.GoalObservationRequestV0{GoalRef: refs[0]}
	if len(refs) > 1 && refs[1] != refs[0] {
		request.ExternalGoalRef = refs[1]
	}
	if issues := orquestagoal.ValidateGoalObservationRequestV0(request); len(issues) > 0 {
		return orquestagoal.GoalObservationRequestV0{}, false
	}
	return request, true
}

func idleSelfImprovementGoalObservationReasonV0(reason string, reasonCode string) bool {
	reason = strings.TrimSpace(reason)
	reasonCode = strings.TrimSpace(reasonCode)
	if strings.HasPrefix(reason, "prepared") ||
		reasonCode == "prepared" ||
		reasonCode == idleSelfImprovementGoalRunningReasonV0 ||
		reasonCode == idleSelfImprovementGoalCompletePendingClosureV0 ||
		reasonCode == idleSelfImprovementGoalBlockedReasonV0 ||
		reasonCode == idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 ||
		reasonCode == idleSelfImprovementGoalInvalidReasonV0 ||
		reasonCode == idleSelfImprovementGoalObservationErrorReasonV0 ||
		reasonCode == idleSelfImprovementGoalObserverUnavailableReasonV0 {
		return true
	}
	return false
}
