package orquestaserver

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"
)

func (runtime *RuntimeV0) runResidentDirectorLoopV0(ctx context.Context) {
	if runtime.residentDirector == nil || !runtime.config.ResidentDirectorEnabled {
		return
	}
	runtime.runResidentDirectorTickAsyncV0(ctx)
	ticker := time.NewTicker(runtime.config.TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-runtime.residentDirectorWakeups:
			runtime.runResidentDirectorTickAsyncV0(ctx)
		case <-ticker.C:
			runtime.runResidentDirectorTickAsyncV0(ctx)
		}
	}
}

func (runtime *RuntimeV0) runResidentDirectorTickAsyncV0(ctx context.Context) bool {
	if runtime.residentDirector == nil || !runtime.config.ResidentDirectorEnabled {
		return false
	}
	if runtime.supervisorFrozenForShutdownV0() {
		runtime.markResidentDirectorSkippedV0(ctx, "shutdown_in_progress")
		return false
	}
	if !atomic.CompareAndSwapInt32(&runtime.residentDirectorTickActive, 0, 1) {
		atomic.StoreInt32(&runtime.residentDirectorTickPending, 1)
		return false
	}
	atomic.StoreInt32(&runtime.residentDirectorTickPending, 0)
	runtime.runAsyncWorkV0("resident_director_tick", func() {
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkResidentDirectorTickActiveV0(true, runtime.clock.Now()),
			"resident_director_tick_active",
		)
		defer func() {
			atomic.StoreInt32(&runtime.residentDirectorTickPending, 0)
			atomic.StoreInt32(&runtime.residentDirectorTickActive, 0)
			runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
				runtime.persistStateTransitionV0(
					receiptCtx,
					runtime.tracker.MarkResidentDirectorTickActiveV0(false, runtime.clock.Now()),
					"resident_director_tick_inactive",
				)
			})
		}()
		runtime.runResidentDirectorTickV0(ctx)
		if atomic.SwapInt32(&runtime.residentDirectorTickPending, 0) == 1 &&
			ctx.Err() == nil &&
			!runtime.supervisorFrozenForShutdownV0() {
			runtime.runResidentDirectorTickV0(ctx)
		}
	})
	return true
}

func (runtime *RuntimeV0) runResidentDirectorTickV0(ctx context.Context) {
	defer runtime.recoverResidentDirectorTickPanicV0(ctx)
	if runtime.residentDirector == nil || !runtime.config.ResidentDirectorEnabled {
		return
	}
	if runtime.supervisorFrozenForShutdownV0() {
		runtime.markResidentDirectorSkippedV0(ctx, "shutdown_in_progress")
		return
	}
	if err := ctx.Err(); err != nil {
		return
	}
	command := ResidentDirectorCommandV0{
		MaxActions:    runtime.config.ResidentDirectorMaxActions,
		OccurredAt:    runtime.clock.Now().Format(time.RFC3339),
		CorrelationID: "resident-director-" + strconv.Itoa(runtime.tracker.SnapshotV0().ResidentDirectorTicks+1),
		EvidenceRefs:  []string{"evidence-ref-resident-director-loop"},
	}
	runtime.auditEventV0(ctx, "resident_director_tick_start", "running", "", map[string]interface{}{
		"max_actions": command.MaxActions,
	})
	result, err := runtime.residentDirector.RunResidentDirectorV0(ctx, command)
	now := runtime.clock.Now()
	if err != nil {
		runtime.auditEventV0(ctx, "resident_director_tick_error", "error", err.Error(), map[string]interface{}{
			"result_summary": residentDirectorResultAuditSummaryV0(result),
		})
		runtime.persistStateTransitionV0(
			ctx,
			runtime.tracker.MarkResidentDirectorErrorV0(result, err.Error(), now),
			"resident_director_error",
		)
		return
	}
	runtime.auditEventV0(ctx, "resident_director_tick_result", "ok", "", map[string]interface{}{
		"result_summary": residentDirectorResultAuditSummaryV0(result),
	})
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkResidentDirectorV0(result, now),
		"resident_director_tick",
	)
}

func (runtime *RuntimeV0) markResidentDirectorSkippedV0(ctx context.Context, reason string) {
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkResidentDirectorSkippedV0(reason, runtime.clock.Now()),
		"resident_director_skipped",
	)
}

func residentDirectorResultAuditSummaryV0(result ResidentDirectorResultV0) map[string]interface{} {
	return map[string]interface{}{
		"status":           result.Status,
		"run_ref":          result.RunRef,
		"executed_actions": result.ExecutedActions,
		"evidence_refs":    compactServerDiagnosticStringsV0(result.EvidenceRefs),
	}
}
