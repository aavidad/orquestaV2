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
		EffectiveConfig:           config.EffectiveConfig,
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
		state.StartupMessage = projectServerStartupMessageV0(result.Message)
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
		state.StartupMessage = projectServerStartupMessageV0(result.Message)
		state.StartupEvidenceRefs = append([]string(nil), result.EvidenceRefs...)
		state.LastError = projectServerOperationalMessageV0("startup", result.Message)
		appendRecentServerErrorV0(state, now, "startup_blocked", "startup", state.LastError, result.EvidenceRefs)
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
		state.LastSupervisorStopPublic = strings.TrimSpace(metrics.PublicStop)
		state.LastSupervisorStopCategory = strings.TrimSpace(metrics.StopCategory)
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

func (tracker *StatusTrackerV0) MarkSupervisorTickActiveV0(active bool, now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.SupervisorTickActive = active
	})
}

func (tracker *StatusTrackerV0) MarkSupervisorFrozenV0(reason string, now time.Time) StateV0 {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "shutdown_in_progress"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastSupervisorAt = formatTimeV0(now)
		state.LastSupervisorStatus = "skipped"
		state.LastSupervisorStop = reason
		state.LastSupervisorStopPublic = reason
		state.LastSupervisorStopCategory = "control"
		state.LastSupervisorExecutions = 0
		state.LastSupervisorSkips = 0
		state.SupervisorTickActive = false
		state.SupervisorFrozen = true
	})
}

func (tracker *StatusTrackerV0) MarkSupervisorErrorV0(
	command orquestarunsupervisor.RunSupervisorCommandV0,
	result orquestarunsupervisor.RunSupervisorResultV0,
	message string,
	now time.Time,
) StateV0 {
	message = projectServerOperationalMessageV0("supervisor", message)
	metrics := collectSupervisorResultMetricsV0(result)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastSupervisorAt = formatTimeV0(now)
		state.LastSupervisorStatus = "error"
		state.LastSupervisorStop = strings.TrimSpace(metrics.StopReason)
		state.LastSupervisorStopPublic = strings.TrimSpace(metrics.PublicStop)
		state.LastSupervisorStopCategory = strings.TrimSpace(metrics.StopCategory)
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
		appendRecentServerErrorV0(state, now, firstNonEmptyServerDiagnosticV0(metrics.StopReason, "supervisor_error"), "supervisor", message, result.ErrorRunRefs)
	})
}

type ShutdownProjectionV0 struct {
	Status             string
	Ready              bool
	HTTPStatus         int
	RunsRequested      int
	RunsStopped        int
	AgentsInFlight     int
	CheckpointsPending int
}

func (tracker *StatusTrackerV0) MarkShutdownRequestedV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = true
		state.ShutdownStatus = "requested"
		state.ShutdownReady = false
		state.SupervisorFrozen = true
	})
}

func (tracker *StatusTrackerV0) MarkShutdownResultV0(
	result ShutdownProjectionV0,
	keepFrozen bool,
	now time.Time,
) StateV0 {
	result.Status = strings.TrimSpace(result.Status)
	if result.Status == "" {
		result.Status = "observed"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = keepFrozen
		state.ShutdownStatus = result.Status
		state.ShutdownReady = result.Ready
		state.ShutdownHTTPStatus = result.HTTPStatus
		state.ShutdownRunsRequested = result.RunsRequested
		state.ShutdownRunsStopped = result.RunsStopped
		state.ShutdownAgentsInFlight = result.AgentsInFlight
		state.ShutdownCheckpointsPending = result.CheckpointsPending
		state.SupervisorFrozen = keepFrozen
		if keepFrozen {
			state.SupervisorTickActive = false
		}
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
			reason = append(reason, projectServerOperationalReasonFieldV0("status", result.Status))
		}
		if strings.TrimSpace(result.Message) != "" {
			reason = append(reason, projectServerOperationalReasonFieldV0("message", result.Message))
		}
		if len(result.NextActions) > 0 {
			reason = append(reason, projectServerOperationalReasonFieldV0("next", strings.Join(result.NextActions, ",")))
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
	message = projectServerOperationalMessageV0("idle_self_improvement", message)
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
		appendRecentServerErrorV0(state, now, "idle_self_improvement_error", "idle_self_improvement", message, nil)
	})
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementCheckedV0(reason string, now time.Time) StateV0 {
	reason = projectServerOperationalMessageV0("idle_self_improvement", reason)
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
	message = projectServerOperationalMessageV0("server", message)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastError = message
		appendRecentServerErrorV0(state, now, "server_error", "server", message, nil)
	})
}

func (tracker *StatusTrackerV0) MarkStatePersistFailedV0(transition string, now time.Time) StateV0 {
	transition = strings.TrimSpace(transition)
	if transition == "" {
		transition = "state_update"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.StatePersistStatus = "degraded"
		state.StatePersistFailures++
		state.StatePersistLastFailedAt = formatTimeV0(now)
		state.StatePersistLastCode = "state_persist_failed"
		state.StatePersistLastTransition = transition
		appendRecentServerErrorV0(
			state,
			now,
			"state_persist_failed",
			"state",
			"state_persist_failed",
			nil,
		)
	})
}

func (tracker *StatusTrackerV0) MarkStatePersistConfirmedV0(transition string, now time.Time) StateV0 {
	transition = strings.TrimSpace(transition)
	if transition == "" {
		transition = "state_update"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.StatePersistStatus = "ok"
		state.StatePersistLastConfirmed = formatTimeV0(now)
		state.StatePersistLastTransition = transition
	})
}

func (tracker *StatusTrackerV0) MarkStoppedV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "stopped"
		state.LastHeartbeatAt = formatTimeV0(now)
	})
}

func appendRecentServerErrorV0(
	state *StateV0,
	now time.Time,
	code string,
	scope string,
	message string,
	evidenceRefs []string,
) {
	if state == nil {
		return
	}
	message = projectServerOperationalMessageV0(scope, message)
	code = strings.TrimSpace(code)
	if message == "" && code == "" {
		return
	}
	if code == "" {
		code = "server_error"
	}
	entry := ServerDiagnosticV0{
		OccurredAt:   formatTimeV0(now),
		Code:         code,
		Scope:        strings.TrimSpace(scope),
		Message:      message,
		EvidenceRefs: compactServerDiagnosticStringsV0(evidenceRefs),
	}
	state.RecentErrors = append([]ServerDiagnosticV0{entry}, state.RecentErrors...)
	if len(state.RecentErrors) > 25 {
		state.RecentErrors = state.RecentErrors[:25]
	}
}

func compactServerDiagnosticStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func firstNonEmptyServerDiagnosticV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
