package orquestaserver

import (
	"context"
	"sync/atomic"
	"time"
)

func (runtime *RuntimeV0) runSupervisorLoopV0(ctx context.Context) {
	if runtime.supervisor == nil {
		return
	}
	runtime.runSupervisorTickAsyncV0(ctx)
	ticker := time.NewTicker(runtime.config.TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runtime.runSupervisorTickAsyncV0(ctx)
		}
	}
}

func (runtime *RuntimeV0) runSupervisorTickAsyncV0(ctx context.Context) bool {
	if runtime.supervisorFrozenForShutdownV0() {
		runtime.markSupervisorFrozenForShutdownV0(ctx, "shutdown_in_progress")
		return false
	}
	if !atomic.CompareAndSwapInt32(&runtime.supervisorTickActive, 0, 1) {
		atomic.StoreInt32(&runtime.supervisorTickPending, 1)
		return false
	}
	atomic.StoreInt32(&runtime.supervisorTickPending, 0)
	runtime.runAsyncWorkV0("supervisor_tick", func() {
		runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkSupervisorTickActiveV0(true, runtime.clock.Now()), "supervisor_tick_active")
		defer func() {
			atomic.StoreInt32(&runtime.supervisorTickPending, 0)
			atomic.StoreInt32(&runtime.supervisorTickActive, 0)
			runtime.withAsyncReceiptContextV0(func(receiptCtx context.Context) {
				runtime.persistStateTransitionV0(receiptCtx, runtime.tracker.MarkSupervisorTickActiveV0(false, runtime.clock.Now()), "supervisor_tick_inactive")
			})
		}()
		runtime.runSupervisorTickV0(ctx)
		if atomic.SwapInt32(&runtime.supervisorTickPending, 0) == 1 &&
			ctx.Err() == nil &&
			!runtime.supervisorFrozenForShutdownV0() {
			runtime.runSupervisorTickV0(ctx)
		}
	})
	return true
}

func (runtime *RuntimeV0) runSupervisorTickV0(ctx context.Context) {
	defer runtime.recoverSupervisorTickPanicV0(ctx)
	if runtime.supervisorFrozenForShutdownV0() {
		runtime.markSupervisorFrozenForShutdownV0(ctx, "shutdown_in_progress")
		return
	}
	if err := ctx.Err(); err != nil {
		return
	}
	command := runtime.config.SupervisorCommand
	if command.MaxTicks <= 0 {
		command.MaxTicks = DefaultSupervisorMaxTicksV0
	}
	runtime.auditEventV0(ctx, "supervisor_tick_start", "running", "", map[string]interface{}{
		"command_summary": supervisorCommandAuditSummaryV0(command),
	})
	result, err := runtime.supervisor.RunGlobalSupervisorV0(ctx, command)
	now := runtime.clock.Now()
	if err != nil {
		runtime.auditEventV0(ctx, "supervisor_tick_error", "error", err.Error(), map[string]interface{}{
			"command_summary": supervisorCommandAuditSummaryV0(command),
			"result_summary":  supervisorResultAuditSummaryV0(result),
		})
		runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkSupervisorErrorV0(
			command,
			result,
			err.Error(),
			now,
		), "supervisor_error")
		return
	}
	runtime.auditEventV0(ctx, "supervisor_tick_result", "ok", "", map[string]interface{}{
		"command_summary": supervisorCommandAuditSummaryV0(command),
		"result_summary":  supervisorResultAuditSummaryV0(result),
	})
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkSupervisorV0(command, result, now), "supervisor_tick")
	runtime.maybeScheduleIdleSelfImprovementV0(ctx, result, now)
}
