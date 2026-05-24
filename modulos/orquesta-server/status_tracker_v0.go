package orquestaserver

import (
	"os"
	"strings"
	"sync"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type StatusTrackerV0 struct {
	mu                          sync.RWMutex
	state                       StateV0
	supervisorNoExecutionSince  time.Time
	lastIdleSelfImprovementAt   time.Time
	idleSelfImprovementInFlight bool
	idleSelfImprovementAccepted bool
	idleSelfImprovementAttempts int
	idleSelfImprovementPrepared int
}

func NewStatusTrackerV0(config ConfigV0, now time.Time) *StatusTrackerV0 {
	config = NormalizeConfigV0(config)
	return &StatusTrackerV0{state: StateV0{
		SchemaVersion:             StateSchemaVersionV0,
		Status:                    "starting",
		PID:                       os.Getpid(),
		Addr:                      config.Addr,
		ProjectWorkDir:            config.ProjectWorkDir,
		RuntimeWorkDir:            config.RuntimeWorkDir,
		StartedAt:                 formatTimeV0(now),
		LastHeartbeatAt:           formatTimeV0(now),
		IdleSelfImprovementAfter:  config.IdleSelfImprovementAfter.String(),
		IdleSelfImprovementTarget: config.IdleSelfImprovementTargetQueue,
	}}
}

func (tracker *StatusTrackerV0) SnapshotV0() StateV0 {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	return tracker.state
}

func (tracker *StatusTrackerV0) MarkServingV0(addr string, now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "running"
		state.Addr = strings.TrimSpace(addr)
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastError = ""
	})
}

func (tracker *StatusTrackerV0) MarkStartupCheckingV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "starting"
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastStartupCheckAt = formatTimeV0(now)
		state.StartupStatus = StartupCheckStatusCheckingV0
		state.StartupReady = false
		state.StartupMessage = ""
		state.StartupEvidenceRefs = nil
		state.LastError = ""
	})
}

func (tracker *StatusTrackerV0) MarkStartupReadyV0(
	result StartupCheckResultV0,
	now time.Time,
) StateV0 {
	result = normalizeStartupCheckResultV0(result)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastStartupCheckAt = formatTimeV0(now)
		state.StartupStatus = result.Status
		state.StartupReady = true
		state.StartupMessage = result.Message
		state.StartupEvidenceRefs = append([]string(nil), result.EvidenceRefs...)
		state.LastError = ""
	})
}

func (tracker *StatusTrackerV0) MarkStartupBlockedV0(
	result StartupCheckResultV0,
	now time.Time,
) StateV0 {
	result = normalizeStartupCheckResultV0(result)
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "startup_blocked"
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastStartupCheckAt = formatTimeV0(now)
		state.StartupStatus = result.Status
		state.StartupReady = false
		state.StartupMessage = result.Message
		state.StartupEvidenceRefs = append([]string(nil), result.EvidenceRefs...)
		state.LastError = result.Message
	})
}

func (tracker *StatusTrackerV0) MarkSupervisorV0(
	command orquestarunsupervisor.RunSupervisorCommandV0,
	result orquestarunsupervisor.RunSupervisorResultV0,
	now time.Time,
) StateV0 {
	metrics := collectSupervisorResultMetricsV0(result)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastSupervisorAt = formatTimeV0(now)
		state.LastSupervisorStatus = "ok"
		state.LastSupervisorStop = strings.TrimSpace(metrics.StopReason)
		state.LastSupervisorError = ""
		state.LastSupervisorQueueRef = strings.TrimSpace(command.QueueRef)
		state.LastSupervisorQueueSize = metrics.QueueSize
		state.LastSupervisorTickNumber = metrics.LastTick
		state.LastSupervisorResultTicks = metrics.ResultTicks
		state.LastSupervisorExecutions = metrics.Executions
		state.LastSupervisorSkips = metrics.Skips
		tracker.markSupervisorIdleWindowV0(result, now)
		state.SupervisorTicks++
		state.SupervisorExecutions += metrics.Executions
		state.SupervisorSkips += metrics.Skips
		state.LastError = ""
	})
}

func (tracker *StatusTrackerV0) MarkSupervisorErrorV0(
	command orquestarunsupervisor.RunSupervisorCommandV0,
	result orquestarunsupervisor.RunSupervisorResultV0,
	message string,
	now time.Time,
) StateV0 {
	message = strings.TrimSpace(message)
	metrics := collectSupervisorResultMetricsV0(result)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastSupervisorAt = formatTimeV0(now)
		state.LastSupervisorStatus = "error"
		state.LastSupervisorStop = strings.TrimSpace(metrics.StopReason)
		state.LastSupervisorError = message
		state.SupervisorLastErrorAt = formatTimeV0(now)
		state.SupervisorLastError = message
		state.LastSupervisorQueueRef = strings.TrimSpace(command.QueueRef)
		state.LastSupervisorQueueSize = metrics.QueueSize
		state.LastSupervisorTickNumber = metrics.LastTick
		state.LastSupervisorResultTicks = metrics.ResultTicks
		state.LastSupervisorExecutions = metrics.Executions
		state.LastSupervisorSkips = metrics.Skips
		tracker.markSupervisorIdleWindowV0(result, now)
		state.SupervisorTicks++
		state.SupervisorErrorTicks++
		state.SupervisorExecutions += metrics.Executions
		state.SupervisorSkips += metrics.Skips
		state.LastError = message
	})
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementPreparedV0(
	result IdleSelfImprovementResultV0,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		reason := []string{"prepared"}
		if strings.TrimSpace(result.RunRef) != "" {
			reason = append(reason, "run_ref="+strings.TrimSpace(result.RunRef))
		}
		if strings.TrimSpace(result.RequestRef) != "" {
			reason = append(reason, "request_ref="+strings.TrimSpace(result.RequestRef))
		}
		if strings.TrimSpace(result.Status) != "" {
			reason = append(reason, "status="+strings.TrimSpace(result.Status))
		}
		if strings.TrimSpace(result.Message) != "" {
			reason = append(reason, "message="+strings.TrimSpace(result.Message))
		}
		if len(result.NextActions) > 0 {
			reason = append(reason, "next="+strings.Join(result.NextActions, ","))
		}
		if len(result.EvidenceRefs) > 0 {
			reason = append(reason, "evidence="+strings.Join(result.EvidenceRefs, ","))
		}
		state.IdleSelfImprovementReason = strings.Join(reason, ";")
		tracker.lastIdleSelfImprovementAt = now.UTC()
		tracker.idleSelfImprovementInFlight = false
		tracker.idleSelfImprovementAccepted = result.Accepted
		tracker.idleSelfImprovementAttempts++
		if result.Accepted {
			tracker.idleSelfImprovementPrepared++
		}
		state.IdleSelfImprovementFlight = false
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
		state.LastError = ""
	})
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementScheduledV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = "scheduled"
		tracker.lastIdleSelfImprovementAt = now.UTC()
		tracker.idleSelfImprovementInFlight = true
		tracker.idleSelfImprovementAccepted = false
		state.IdleSelfImprovementFlight = true
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
	})
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementErrorV0(message string, now time.Time) StateV0 {
	message = strings.TrimSpace(message)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = "error:" + message
		tracker.lastIdleSelfImprovementAt = now.UTC()
		tracker.idleSelfImprovementInFlight = false
		tracker.idleSelfImprovementAccepted = false
		tracker.idleSelfImprovementAttempts++
		state.IdleSelfImprovementFlight = false
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
		state.LastError = message
	})
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementCheckedV0(reason string, now time.Time) StateV0 {
	reason = strings.TrimSpace(reason)
	return tracker.updateV0(func(state *StateV0) {
		if reason == "attempt_blocked" &&
			tracker.idleSelfImprovementAccepted &&
			strings.HasPrefix(state.IdleSelfImprovementReason, "prepared") {
			reason = state.IdleSelfImprovementReason
			if !strings.Contains(reason, "pending=accepted_attempt") {
				reason += ";pending=accepted_attempt"
			}
		}
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = reason
		state.IdleSelfImprovementFlight = tracker.idleSelfImprovementInFlight
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
	})
}

func (tracker *StatusTrackerV0) MarkErrorV0(message string, now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastError = strings.TrimSpace(message)
	})
}

func (tracker *StatusTrackerV0) MarkStoppedV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "stopped"
		state.LastHeartbeatAt = formatTimeV0(now)
	})
}

func (tracker *StatusTrackerV0) updateV0(fn func(*StateV0)) StateV0 {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	next := tracker.state
	fn(&next)
	tracker.state = next
	return next
}

func (tracker *StatusTrackerV0) IdleSelfImprovementWindowV0() (time.Time, time.Time, bool, bool) {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	return tracker.supervisorNoExecutionSince,
		tracker.lastIdleSelfImprovementAt,
		tracker.idleSelfImprovementInFlight,
		tracker.idleSelfImprovementAccepted
}

func (tracker *StatusTrackerV0) markSupervisorIdleWindowV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
	now time.Time,
) {
	if !supervisorResultIsIdleForSelfImprovementV0(result) {
		tracker.supervisorNoExecutionSince = time.Time{}
		return
	}
	if tracker.supervisorNoExecutionSince.IsZero() {
		tracker.supervisorNoExecutionSince = now.UTC()
	}
}

func formatTimeV0(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
