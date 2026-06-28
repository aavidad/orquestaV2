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
	idleSelfImprovementGoalInvalidReasonV0             = "goal_invalid"
	idleSelfImprovementGoalObservationErrorReasonV0    = "goal_observation_error"
)

func (runtime *RuntimeV0) observePendingIdleSelfImprovementGoalV0(ctx context.Context, now time.Time) bool {
	// Compatibility bridge for idle_self_improvement goal-first. The generic
	// active-goal observer closes app/external-work goals, but this helper still
	// updates the idle tracker and gates later self-improvement attempts.
	if runtime == nil || !runtime.config.IdleSelfImprovementGoalFirst || runtime.tracker == nil {
		return false
	}
	request, ok := idleSelfImprovementGoalObservationRequestFromStateV0(runtime.tracker.SnapshotV0())
	if !ok {
		return false
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
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkIdleSelfImprovementGoalObservedV0(result, err, now),
		"idle_self_improvement_goal_observed",
	)
	return true
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
		reasonCode == idleSelfImprovementGoalInvalidReasonV0 ||
		reasonCode == idleSelfImprovementGoalObservationErrorReasonV0 ||
		reasonCode == idleSelfImprovementGoalObserverUnavailableReasonV0 {
		return true
	}
	return false
}
