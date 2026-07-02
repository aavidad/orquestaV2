package orquestaserver

import (
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
		state.LastErrorOperationalMessage = nil
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
		state.StartupOperationalMessage = nil
		state.StartupRevision = StartupRevisionSummaryV0{}
		state.StartupBlockers = nil
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
		state.StartupOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "startup",
				ReasonCode:   "startup_ready",
				Status:       result.Status,
				Message:      result.Message,
				EvidenceRefs: result.EvidenceRefs,
			},
		)
		state.StartupRevision = result.StartupRevision
		state.StartupBlockers = nil
		state.StartupEvidenceRefs = append([]string(nil), result.EvidenceRefs...)
		state.LastError = ""
		state.LastErrorOperationalMessage = nil
		state.SupervisorFrozen = false
		state.ShutdownInProgress = false
		state.ShutdownStatus = ""
		state.ShutdownReady = false
		state.ShutdownHTTPStatus = 0
		state.ShutdownRunsRequested = 0
		state.ShutdownRunsStopped = 0
		state.ShutdownAgentsInFlight = 0
		state.ShutdownCheckpointsPending = 0
		state.ShutdownAsyncWorkActive = 0
		state.ShutdownStopTimeoutAt = ""
		if state.IdleSelfImprovementReason == "shutdown_in_progress" {
			state.IdleSelfImprovementReason = ""
			state.IdleSelfImprovementOperationalMessage = nil
		}
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
		state.StartupOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "startup",
				ReasonCode:   "startup_blocked",
				Status:       result.Status,
				Message:      result.Message,
				EvidenceRefs: result.EvidenceRefs,
			},
		)
		state.StartupRevision = result.StartupRevision
		state.StartupBlockers = append([]StartupBlockerV0(nil), result.Blockers...)
		state.StartupEvidenceRefs = append([]string(nil), result.EvidenceRefs...)
		state.LastError = projectServerOperationalMessageV0("startup", result.Message)
		state.LastErrorOperationalMessage = copyServerOperationalMessageV0(state.StartupOperationalMessage)
		appendRecentServerErrorV0(state, now, "startup_blocked", "startup", state.LastError, result.EvidenceRefs)
	})
}

func (tracker *StatusTrackerV0) MarkSupervisorV0(command orquestarunsupervisor.RunSupervisorCommandV0, result orquestarunsupervisor.RunSupervisorResultV0, now time.Time) StateV0 {
	metrics := collectSupervisorResultMetricsV0(result)
	baseProjection := supervisorPublicProjectionV0(result, metrics)
	return tracker.updateV0(func(state *StateV0) {
		projection, goalSnapshot := supervisorProjectionWithGoalBackendV0(baseProjection, metrics, *state)
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastSupervisorAt = formatTimeV0(now)
		state.LastSupervisorStatus = projection.Status
		state.LastSupervisorStop = strings.TrimSpace(metrics.StopReason)
		state.LastSupervisorStopPublic = strings.TrimSpace(projection.StopPublic)
		state.LastSupervisorStopCategory = strings.TrimSpace(projection.StopCategory)
		state.LastSupervisorError = ""
		state.LastSupervisorQueueRef = strings.TrimSpace(command.QueueRef)
		state.LastSupervisorQueueSize = metrics.QueueSize
		state.LastSupervisorTickNumber = metrics.LastTick
		state.LastSupervisorResultTicks = metrics.ResultTicks
		state.LastSupervisorExecutions = metrics.Executions
		state.LastSupervisorSkips = metrics.Skips
		state.LastSupervisorOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "supervisor",
				ReasonCode:   projection.Status,
				Status:       projection.Status,
				Message:      "supervisor_projection",
				RunRefs:      goalSnapshot.RunRefs,
				GoalRefs:     goalSnapshot.GoalRefs,
				EvidenceRefs: goalSnapshot.EvidenceRefs,
				Counters: supervisorCountersWithGoalBackendV0(
					supervisorPublicCountersV0(result),
					goalSnapshot,
				),
			},
		)
		tracker.markSupervisorIdleWindowV0(result, now)
		state.SupervisorTicks++
		state.SupervisorExecutions += metrics.Executions
		state.SupervisorSkips += metrics.Skips
		state.LastError = ""
		state.LastErrorOperationalMessage = nil
	})
}

func supervisorPublicProjectionV0(result orquestarunsupervisor.RunSupervisorResultV0, metrics supervisorResultMetricsV0) SupervisorPublicProjectionV0 {
	projection := SupervisorPublicProjectionV0{
		Status:       SupervisorPublicStatusOKV0,
		StopPublic:   strings.TrimSpace(metrics.PublicStop),
		StopCategory: strings.TrimSpace(metrics.StopCategory),
	}
	if supervisorResultHasRunningLiveV0(result) {
		projection.Status = SupervisorPublicStatusRunningLiveV0
		projection.StopPublic = SupervisorPublicStopRunningLiveV0
		projection.StopCategory = SupervisorPublicCategoryExternalProcessV0
		return projection
	}
	if supervisorResultHasExternalWaitV0(result) {
		projection.Status = SupervisorPublicStatusWaitingExternalV0
		projection.StopPublic = SupervisorPublicStopWaitingExternalV0
		projection.StopCategory = SupervisorPublicCategoryWaitExternalV0
		return projection
	}
	if supervisorResultHasExternalEmptyRunV0(result) {
		projection.Status = SupervisorPublicStatusExternalEmptyRunV0
		projection.StopPublic = SupervisorPublicStopExternalEmptyRunV0
		projection.StopCategory = SupervisorPublicCategoryExternalProcessV0
		return projection
	}
	if supervisorResultHasLaunchFailedV0(result) {
		projection.Status = SupervisorPublicStatusLaunchFailedV0
		projection.StopPublic = SupervisorPublicStopLaunchFailedV0
		projection.StopCategory = SupervisorPublicCategoryExternalProcessV0
		return projection
	}
	if supervisorResultHasUnverifiedRunningV0(result) {
		projection.Status = SupervisorPublicStatusStalledV0
		projection.StopPublic = SupervisorPublicStopStalledV0
		projection.StopCategory = SupervisorPublicCategoryExternalProcessV0
		return projection
	}
	if supervisorResultHasUnhandledOutboxWaitV0(result) {
		projection.Status = SupervisorPublicStatusWaitingOutboxV0
		projection.StopPublic = SupervisorPublicStopWaitingOutboxV0
		projection.StopCategory = SupervisorPublicCategoryWaitOutboxV0
		return projection
	}
	return projection
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
	projection := supervisorPublicProjectionV0(result, metrics)
	recoverable := supervisorRecoverableErrorProjectionV0(projection)
	evidenceRefs := supervisorResultEvidenceRefsV0(result)
	return tracker.updateV0(func(state *StateV0) {
		status := "error"
		stopPublic := strings.TrimSpace(metrics.PublicStop)
		stopCategory := strings.TrimSpace(metrics.StopCategory)
		reasonCode := firstNonEmptyServerDiagnosticV0(metrics.StopReason, "supervisor_error")
		messageStatus := "error"
		currentSupervisorError := message
		currentLastError := message
		if recoverable {
			status = strings.TrimSpace(projection.Status)
			stopPublic = strings.TrimSpace(projection.StopPublic)
			stopCategory = strings.TrimSpace(projection.StopCategory)
			reasonCode = firstNonEmptyServerDiagnosticV0(status, reasonCode)
			messageStatus = status
			currentSupervisorError = ""
			currentLastError = ""
		}
		counters := map[string]int{
			"queue_size":   metrics.QueueSize,
			"result_ticks": metrics.ResultTicks,
			"executions":   metrics.Executions,
			"skips":        metrics.Skips,
		}
		if recoverable {
			for key, value := range supervisorPublicCountersV0(result) {
				counters[key] = value
			}
			counters["supervisor_error_advisory"] = 1
		}
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastSupervisorAt = formatTimeV0(now)
		state.LastSupervisorStatus = status
		state.LastSupervisorStop = strings.TrimSpace(metrics.StopReason)
		state.LastSupervisorStopPublic = stopPublic
		state.LastSupervisorStopCategory = stopCategory
		state.LastSupervisorError = currentSupervisorError
		state.LastSupervisorOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "supervisor",
				ReasonCode:   reasonCode,
				Status:       messageStatus,
				Message:      message,
				RunRefs:      result.ErrorRunRefs,
				EvidenceRefs: evidenceRefs,
				Counters:     counters,
			},
		)
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
		state.LastError = currentLastError
		if recoverable {
			state.LastErrorOperationalMessage = nil
		} else {
			state.LastErrorOperationalMessage = copyServerOperationalMessageV0(state.LastSupervisorOperationalMessage)
		}
		appendRecentServerErrorV0(state, now, reasonCode, "supervisor", message, evidenceRefs)
	})
}

func supervisorRecoverableErrorProjectionV0(projection SupervisorPublicProjectionV0) bool {
	switch strings.TrimSpace(projection.Status) {
	case SupervisorPublicStatusRunningLiveV0,
		SupervisorPublicStatusWaitingExternalV0,
		SupervisorPublicStatusWaitingOutboxV0:
		return true
	default:
		return false
	}
}

func supervisorResultEvidenceRefsV0(result orquestarunsupervisor.RunSupervisorResultV0) []string {
	refs := append([]string(nil), result.StopProjection.EvidenceRefs...)
	for _, diagnostic := range result.Diagnostics {
		refs = append(refs, diagnostic.EvidenceRefs...)
	}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			refs = append(refs, execution.EvidenceRefs...)
			for _, diagnostic := range execution.Diagnostics {
				refs = append(refs, diagnostic.EvidenceRefs...)
			}
		}
	}
	return compactServerDiagnosticStringsV0(refs)
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
	observedAt := now.UTC()
	if tracker.supervisorNoExecutionSince.IsZero() ||
		observedAt.Before(tracker.supervisorNoExecutionSince) {
		tracker.supervisorNoExecutionSince = observedAt
	}
}

func formatTimeV0(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
