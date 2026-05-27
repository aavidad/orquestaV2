package orquestaserver

import (
	"strings"
	"time"
)

func (tracker *StatusTrackerV0) MarkRuntimeStoppingV0(reason string, active int, now time.Time) StateV0 {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "async_work_draining"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "stopping"
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = true
		state.ShutdownStatus = reason
		state.ShutdownReady = false
		state.ShutdownAsyncWorkActive = nonNegativeServerIntV0(active)
		state.SupervisorFrozen = true
	})
}

func (tracker *StatusTrackerV0) MarkRuntimeStopTimeoutV0(reason string, active int, now time.Time) StateV0 {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "stop_timeout"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "stop_timeout"
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = true
		state.ShutdownStatus = "stop_timeout"
		state.ShutdownReady = false
		state.ShutdownAsyncWorkActive = nonNegativeServerIntV0(active)
		state.ShutdownStopTimeoutAt = formatTimeV0(now)
		state.SupervisorFrozen = true
		state.LastError = "stop_timeout"
		if state.ShutdownSignalName != "" && state.ShutdownSignalCount <= 0 {
			state.ShutdownSignalCount = 1
		}
		appendRecentServerErrorV0(state, now, reason, "shutdown", "stop_timeout", nil)
	})
}

func (tracker *StatusTrackerV0) MarkRuntimeStoppedV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "stopped"
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = false
		state.ShutdownStatus = "stopped"
		state.ShutdownReady = true
		state.ShutdownAsyncWorkActive = 0
		state.SupervisorFrozen = false
		state.SupervisorTickActive = false
		state.IdleSelfImprovementFlight = false
		state.LastError = ""
	})
}
