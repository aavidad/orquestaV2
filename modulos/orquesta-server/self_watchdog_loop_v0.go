package orquestaserver

import (
	"context"
	"sync/atomic"
	"time"
)

func (runtime *RuntimeV0) runSelfWatchdogLoopV0(
	ctx context.Context,
	stop chan<- SelfWatchdogDecisionV0,
) {
	if runtime.selfWatchdog == nil {
		return
	}
	if runtime.runSelfWatchdogTickV0(ctx, stop) {
		return
	}
	interval := runtime.config.TickInterval
	if interval <= 0 {
		interval = DefaultTickIntervalV0
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if runtime.runSelfWatchdogTickV0(ctx, stop) {
				return
			}
		}
	}
}

func (runtime *RuntimeV0) runSelfWatchdogTickV0(
	ctx context.Context,
	stop chan<- SelfWatchdogDecisionV0,
) bool {
	if err := ctx.Err(); err != nil || runtime.selfWatchdog == nil {
		return false
	}
	now := runtime.clock.Now()
	observation, err := runtime.selfWatchdog.ObserveSelfWatchdogV0(ctx, SelfWatchdogObservationRequestV0{
		State:      runtime.tracker.SnapshotV0(),
		Config:     runtime.config.SelfWatchdog,
		ObservedAt: now,
	})
	if err != nil {
		runtime.auditEventV0(ctx, "self_watchdog_observation_error", "warning", err.Error(), nil)
		return false
	}
	observation.SupervisorTickActive = observation.SupervisorTickActive || atomic.LoadInt32(&runtime.supervisorTickActive) == 1
	observation.ResidentDirectorTickActive = observation.ResidentDirectorTickActive || atomic.LoadInt32(&runtime.residentDirectorTickActive) == 1
	observation.ShutdownInProgress = observation.ShutdownInProgress || atomic.LoadInt32(&runtime.shutdownInProgress) == 1
	decision := EvaluateSelfWatchdogV0(runtime.config.SelfWatchdog, observation)
	runtime.persistStateTransitionV0(ctx, runtime.tracker.MarkSelfWatchdogV0(decision, now), "self_watchdog")
	if !decision.ShouldRequestShutdown {
		return false
	}
	runtime.auditEventV0(ctx, "self_watchdog_shutdown_requested", "warning", decision.ReasonCode, map[string]interface{}{
		"cpu_percent": decision.CPUPercent,
		"reason_code": decision.ReasonCode,
	})
	select {
	case stop <- decision:
	default:
	}
	return true
}

func selfWatchdogShutdownCauseV0(decision SelfWatchdogDecisionV0) ShutdownSignalCauseV0 {
	return ShutdownSignalCauseV0{
		SignalName: "self_watchdog",
		Count:      1,
		Escalated:  false,
	}
}
