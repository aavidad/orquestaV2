package orquestaserver

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
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
		case <-runtime.supervisorWakeups:
			runtime.runSupervisorTickAsyncV0(ctx)
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
		"result_summary":  supervisorResultAuditSummaryWithStateV0(result, runtime.tracker.SnapshotV0()),
	})
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkSupervisorV0(command, result, now), "supervisor_tick")
	if supervisorResultHasUnhandledOutboxWaitV0(result) && !supervisorResultHasLaunchFailedV0(result) && !supervisorResultHasUnverifiedRunningV0(result) {
		runtime.markIdleSelfImprovementCheckedV0(ctx, SupervisorPublicStatusWaitingOutboxV0, now)
		return
	}
	if supervisorResultQueueIdleButGoalBackendActiveForStateV0(result, runtime.tracker.SnapshotV0()) {
		runtime.markIdleSelfImprovementCheckedV0(ctx, SupervisorPublicStopQueueIdleButGoalBackendActiveV0, now)
		return
	}
	runtime.maybeScheduleIdleSelfImprovementCausalV0(ctx, result, now)
}

func supervisorResultHasUnhandledOutboxWaitV0(result orquestarunsupervisor.RunSupervisorResultV0) bool {
	if supervisorWaitOutboxValueV0(result.StopProjection.PublicReason, result.StopProjection.Category, result.StopReason) ||
		supervisorDiagnosticsHavePendingOutboxV0(result.Diagnostics) {
		return true
	}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			if supervisorWaitOutboxValueV0(execution.Outcome, execution.QueueStatus) ||
				supervisorDiagnosticsHavePendingOutboxV0(execution.Diagnostics) {
				return true
			}
		}
		for _, skip := range tick.Result.Skips {
			if supervisorWaitOutboxValueV0(skip.Reason, skip.Status) {
				return true
			}
		}
	}
	return false
}

func supervisorResultHasExternalWaitV0(result orquestarunsupervisor.RunSupervisorResultV0) bool {
	if supervisorExternalWaitValueV0(result.StopProjection.PublicReason, result.StopProjection.Category, result.StopReason) {
		return true
	}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			if supervisorExternalWaitValueV0(execution.Outcome, execution.QueueStatus) {
				return true
			}
		}
		for _, skip := range tick.Result.Skips {
			if supervisorExternalWaitValueV0(skip.Reason, skip.Status) {
				return true
			}
		}
	}
	return false
}

func supervisorResultHasLaunchFailedV0(result orquestarunsupervisor.RunSupervisorResultV0) bool {
	if supervisorLaunchFailedValueV0(result.StopProjection.PublicReason, result.StopProjection.Category, result.StopReason) {
		return true
	}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			if supervisorLaunchFailedValueV0(execution.Outcome, execution.QueueStatus) ||
				supervisorDiagnosticsHaveLaunchFailedV0(execution.Diagnostics) {
				return true
			}
		}
		for _, skip := range tick.Result.Skips {
			if supervisorLaunchFailedValueV0(skip.Reason, skip.Status) {
				return true
			}
		}
	}
	return supervisorDiagnosticsHaveLaunchFailedV0(result.Diagnostics)
}

func supervisorResultHasUnverifiedRunningV0(result orquestarunsupervisor.RunSupervisorResultV0) bool {
	if supervisorRunningValueV0(result.StopProjection.PublicReason, result.StopProjection.Category, result.StopReason) &&
		!supervisorLiveEvidenceV0(result.StopProjection.PublicReason, result.StopProjection.Category, result.StopProjection.EvidenceRefs) &&
		!supervisorDiagnosticsHaveLiveProcessV0(result.Diagnostics) {
		return true
	}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			if supervisorExternalWaitValueV0(execution.Outcome, execution.QueueStatus) ||
				supervisorExecutionIsOnlyWaitingOutboxV0(execution) {
				continue
			}
			if supervisorRunningValueV0(execution.Outcome, execution.QueueStatus) &&
				!supervisorLiveEvidenceV0(execution.Outcome, execution.QueueStatus, execution.EvidenceRefs) &&
				!supervisorDiagnosticsHaveLiveProcessV0(execution.Diagnostics) {
				return true
			}
			if supervisorDiagnosticsHaveUnverifiedRunningV0(execution.Diagnostics) {
				return true
			}
		}
		for _, skip := range tick.Result.Skips {
			if supervisorWaitOutboxValueV0(skip.Reason) {
				continue
			}
			if supervisorRunningValueV0(skip.Reason) &&
				!supervisorLiveEvidenceV0(skip.Reason, skip.Status, nil) {
				return true
			}
		}
	}
	return supervisorDiagnosticsHaveUnverifiedRunningV0(result.Diagnostics)
}

func supervisorResultHasRunningLiveV0(result orquestarunsupervisor.RunSupervisorResultV0) bool {
	if supervisorLiveEvidenceV0(result.StopProjection.PublicReason, result.StopProjection.Category, result.StopProjection.EvidenceRefs) ||
		supervisorDiagnosticsHaveLiveProcessV0(result.Diagnostics) {
		return true
	}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			if supervisorLiveEvidenceV0(execution.Outcome, execution.QueueStatus, execution.EvidenceRefs) ||
				supervisorDiagnosticsHaveLiveProcessV0(execution.Diagnostics) {
				return true
			}
		}
	}
	return false
}

func supervisorPublicCountersV0(result orquestarunsupervisor.RunSupervisorResultV0) map[string]int {
	counters := map[string]int{
		"registered":              0,
		"waiting_outbox":          0,
		"waiting_external":        0,
		"running_live":            0,
		"stalled":                 0,
		"launch_failed":           0,
		"external_work_empty_run": 0,
	}
	for _, tick := range result.Ticks {
		counters["registered"] += len(tick.Result.Ranked)
		for _, execution := range tick.Result.Executions {
			switch supervisorExecutionPublicStatusV0(execution.Outcome, execution.QueueStatus, execution.EvidenceRefs, execution.Diagnostics) {
			case SupervisorPublicStatusExternalEmptyRunV0:
				counters["external_work_empty_run"]++
			case SupervisorPublicStatusLaunchFailedV0:
				counters["launch_failed"]++
			case SupervisorPublicStatusRunningLiveV0:
				counters["running_live"]++
			case SupervisorPublicStatusStalledV0:
				counters["stalled"]++
			case SupervisorPublicStatusWaitingOutboxV0:
				counters["waiting_outbox"]++
			case SupervisorPublicStatusWaitingExternalV0:
				counters["waiting_external"]++
			}
		}
		for _, skip := range tick.Result.Skips {
			if supervisorWaitOutboxValueV0(skip.Reason, skip.Status) {
				counters["waiting_outbox"]++
			}
			if supervisorExternalWaitValueV0(skip.Reason, skip.Status) {
				counters["waiting_external"]++
			}
			if supervisorRunningValueV0(skip.Reason, skip.Status) &&
				!supervisorLiveEvidenceV0(skip.Reason, skip.Status, nil) {
				counters["stalled"]++
			}
		}
	}
	if counters["external_work_empty_run"] == 0 && supervisorResultHasExternalEmptyRunV0(result) {
		counters["external_work_empty_run"] = 1
	}
	return counters
}

func supervisorExecutionPublicStatusV0(
	outcome string,
	queueStatus string,
	evidenceRefs []string,
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
) string {
	if supervisorTerminalEmptyRunEvidenceV0(outcome, queueStatus, "", evidenceRefs, diagnostics) {
		return SupervisorPublicStatusExternalEmptyRunV0
	}
	if supervisorLaunchFailedValueV0(outcome, queueStatus) || supervisorDiagnosticsHaveLaunchFailedV0(diagnostics) {
		return SupervisorPublicStatusLaunchFailedV0
	}
	if supervisorLiveEvidenceV0(outcome, queueStatus, evidenceRefs) || supervisorDiagnosticsHaveLiveProcessV0(diagnostics) {
		return SupervisorPublicStatusRunningLiveV0
	}
	if supervisorExternalWaitValueV0(outcome, queueStatus) {
		return SupervisorPublicStatusWaitingExternalV0
	}
	if supervisorWaitOutboxValueV0(outcome, queueStatus) ||
		(supervisorDiagnosticsHavePendingOutboxV0(diagnostics) && !supervisorRunningValueV0(outcome)) {
		return SupervisorPublicStatusWaitingOutboxV0
	}
	if supervisorRunningValueV0(outcome, queueStatus) || supervisorDiagnosticsHaveUnverifiedRunningV0(diagnostics) {
		return SupervisorPublicStatusStalledV0
	}
	return SupervisorPublicStatusOKV0
}

func supervisorExecutionIsOnlyWaitingOutboxV0(execution orquestaruncoordinator.RunExecutionSummaryV0) bool {
	if supervisorWaitOutboxValueV0(execution.Outcome, execution.QueueStatus) {
		return true
	}
	return supervisorDiagnosticsHavePendingOutboxV0(execution.Diagnostics) &&
		!supervisorRunningValueV0(execution.Outcome)
}

func supervisorWaitOutboxValueV0(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "wait_unhandled_outbox", "wait_outbox", SupervisorPublicStatusWaitingOutboxV0:
			return true
		}
	}
	return false
}

func supervisorExternalWaitValueV0(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "wait_external", "candidate_pending", SupervisorPublicStatusWaitingExternalV0:
			return true
		}
	}
	return false
}

func supervisorLaunchFailedValueV0(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case SupervisorPublicStatusLaunchFailedV0, "launch_error", "provider_launch_failed":
			return true
		}
	}
	return false
}

func supervisorRunningValueV0(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "running", "process_ref_registered", "process_registered":
			return true
		}
	}
	return false
}

func supervisorLiveEvidenceV0(values ...interface{}) bool {
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			if supervisorLiveEvidenceStringV0(typed) {
				return true
			}
		case []string:
			for _, item := range typed {
				if supervisorLiveEvidenceStringV0(item) {
					return true
				}
			}
		}
	}
	return false
}

func supervisorLiveEvidenceStringV0(value string) bool {
	value = strings.TrimSpace(value)
	switch value {
	case SupervisorPublicStatusRunningLiveV0, "process_live", "heartbeat_recent", "ack_recent", "lease_active",
		"evidence-ref-codex-supervisor-process-live":
		return true
	default:
		return false
	}
}

func supervisorDiagnosticHasLiveProcessV0(diagnostic orquestaruncoordinator.RunDrainDiagnosticV0) bool {
	if supervisorLiveEvidenceV0(diagnostic.Kind, diagnostic.Status, diagnostic.EvidenceRefs) {
		return true
	}
	switch strings.TrimSpace(diagnostic.Kind) {
	case "process_snapshot", "process_runtime_snapshot", "runtime_process_snapshot":
		switch strings.TrimSpace(diagnostic.Status) {
		case "running", "stopping", SupervisorPublicStatusRunningLiveV0:
			return true
		}
		return false
	default:
		return false
	}
}

func supervisorDiagnosticsHaveLiveProcessV0(diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0) bool {
	for _, diagnostic := range diagnostics {
		if supervisorDiagnosticHasLiveProcessV0(diagnostic) {
			return true
		}
	}
	return false
}

func supervisorDiagnosticsHavePendingOutboxV0(diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0) bool {
	for _, diagnostic := range diagnostics {
		if supervisorDiagnosticHasPendingOutboxV0(diagnostic) {
			return true
		}
	}
	return false
}

func supervisorDiagnosticHasPendingOutboxV0(diagnostic orquestaruncoordinator.RunDrainDiagnosticV0) bool {
	return diagnostic.PendingOutboxCount > 0 || len(diagnostic.PendingOutboxRefs) > 0
}

func supervisorDiagnosticsHaveLaunchFailedV0(diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0) bool {
	for _, diagnostic := range diagnostics {
		if supervisorLaunchFailedValueV0(diagnostic.Kind, diagnostic.Status, diagnostic.Error) {
			return true
		}
	}
	return false
}

func supervisorDiagnosticsHaveUnverifiedRunningV0(diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0) bool {
	for _, diagnostic := range diagnostics {
		if supervisorDiagnosticHasLiveProcessV0(diagnostic) {
			continue
		}
		if supervisorRunningValueV0(diagnostic.Kind, diagnostic.Status) &&
			!supervisorLiveEvidenceV0(diagnostic.Kind, diagnostic.Status, diagnostic.EvidenceRefs) {
			return true
		}
	}
	return false
}
