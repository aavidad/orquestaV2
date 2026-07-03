package orquestaserver

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func (runtime *RuntimeV0) runGoalObservationLoopV0(ctx context.Context) {
	if !runtime.goalObservationAvailableV0() {
		return
	}
	runtime.runGoalObservationTickAsyncV0(ctx)
	ticker := time.NewTicker(runtime.config.GoalObserverInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-runtime.goalObservationWakeups:
			runtime.runGoalObservationTickAsyncV0(ctx)
		case <-ticker.C:
			runtime.runGoalObservationTickAsyncV0(ctx)
		}
	}
}

func (runtime *RuntimeV0) runGoalObservationTickAsyncV0(ctx context.Context) bool {
	if !runtime.goalObservationAvailableV0() {
		return false
	}
	if runtime.supervisorFrozenForShutdownV0() {
		runtime.markGoalObservationSkippedV0(ctx, "shutdown_in_progress")
		return false
	}
	if !atomic.CompareAndSwapInt32(&runtime.goalObservationTickActive, 0, 1) {
		atomic.StoreInt32(&runtime.goalObservationTickPending, 1)
		return false
	}
	atomic.StoreInt32(&runtime.goalObservationTickPending, 0)
	runtime.runAsyncWorkV0("goal_observation_tick", func() {
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkGoalObserverTickActiveV0(true, runtime.clock.Now()),
			"goal_observer_tick_active",
		)
		defer func() {
			atomic.StoreInt32(&runtime.goalObservationTickPending, 0)
			atomic.StoreInt32(&runtime.goalObservationTickActive, 0)
			runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
				runtime.persistStateTransitionV0(
					receiptCtx,
					runtime.tracker.MarkGoalObserverTickActiveV0(false, runtime.clock.Now()),
					"goal_observer_tick_inactive",
				)
			})
		}()
		runtime.runGoalObservationTickV0(ctx)
		if atomic.SwapInt32(&runtime.goalObservationTickPending, 0) == 1 &&
			ctx.Err() == nil &&
			!runtime.supervisorFrozenForShutdownV0() {
			runtime.runGoalObservationTickV0(ctx)
		}
	})
	return true
}

func (runtime *RuntimeV0) runGoalObservationTickV0(ctx context.Context) {
	defer runtime.recoverGoalObservationTickPanicV0(ctx)
	if !runtime.goalObservationAvailableV0() {
		return
	}
	observer, ok := runtime.goalActiveObserverV0()
	if !ok {
		return
	}
	if runtime.supervisorFrozenForShutdownV0() {
		runtime.markGoalObservationSkippedV0(ctx, "shutdown_in_progress")
		return
	}
	if err := ctx.Err(); err != nil {
		return
	}
	request := orquestagoal.GoalWorkObserveActiveRequestV0{
		List: orquestagoal.GoalWorkStateListRequestV0{
			ActiveOnly: true,
			MaxItems:   runtime.config.GoalObserverMaxItems,
		},
	}
	fingerprintPlan := runtime.goalObservationFingerprintPlanV0(ctx, request)
	request = fingerprintPlan.Request
	if fingerprintPlan.FilterActive && len(request.List.RunRefs) == 0 {
		runtime.auditEventV0(ctx, "goal_observer_tick_skipped", "skipped", "unchanged_fingerprint", map[string]interface{}{
			"skipped": fingerprintPlan.Skipped,
		})
		runtime.markGoalObservationSkippedV0(ctx, "unchanged_fingerprint")
		return
	}
	correlationID := "goal-observer-" + strconv.Itoa(runtime.tracker.SnapshotV0().GoalObserverTicks+1)
	runtime.auditEventV0(ctx, "goal_observer_tick_start", "running", "", map[string]interface{}{
		"correlation_id": correlationID,
		"max_items":      request.List.MaxItems,
		"run_refs":       compactServerDiagnosticStringsV0(request.List.RunRefs),
		"skipped":        fingerprintPlan.Skipped,
	})
	result, err := observer.ObserveActiveGoalWorksV0(ctx, request)
	now := runtime.clock.Now()
	if err != nil {
		runtime.auditEventV0(ctx, "goal_observer_tick_error", "error", err.Error(), map[string]interface{}{
			"result_summary": goalObservationResultAuditSummaryV0(result),
			"correlation_id": correlationID,
		})
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkGoalObserverErrorV0(err.Error(), now),
			"goal_observer_error",
		)
		return
	}
	runtime.rememberGoalObservationFingerprintsV0(ctx, result, fingerprintPlan.Pending)
	runtime.auditEventV0(ctx, "goal_observer_tick_result", "ok", "", map[string]interface{}{
		"result_summary": goalObservationResultAuditSummaryV0(result),
		"correlation_id": correlationID,
	})
	terminal := goalObserverTerminalCountV0(result)
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkGoalObserverV0(result, now),
		"goal_observer_tick",
	)
	if terminal > 0 {
		runtime.RequestSupervisorWakeupV0("goal_observer_terminal")
	}
}

func (runtime *RuntimeV0) goalObservationAvailableV0() bool {
	if runtime == nil ||
		!runtime.config.GoalObserverEnabled ||
		runtime.goalStateStore == nil {
		return false
	}
	_, ok := runtime.goalActiveObserverV0()
	return ok
}

func (runtime *RuntimeV0) goalActiveObserverV0() (GoalActiveObservationPortV0, bool) {
	if runtime == nil || runtime.supervisor == nil {
		return nil, false
	}
	observer, ok := runtime.supervisor.(GoalActiveObservationPortV0)
	return observer, ok && observer != nil
}

func (runtime *RuntimeV0) markGoalObservationSkippedV0(ctx context.Context, reason string) {
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkGoalObserverSkippedV0(reason, runtime.clock.Now()),
		"goal_observer_skipped",
	)
}

func goalObservationResultAuditSummaryV0(
	result orquestagoal.GoalWorkObserveActiveResultV0,
) map[string]interface{} {
	return map[string]interface{}{
		"observations":  len(result.Observations),
		"terminal":      goalObserverTerminalCountV0(result),
		"issues":        len(result.Issues),
		"evidence_refs": compactServerDiagnosticStringsV0(result.EvidenceRefs),
	}
}
