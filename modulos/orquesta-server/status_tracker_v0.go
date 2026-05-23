package orquestaserver

import (
	"os"
	"strings"
	"sync"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type StatusTrackerV0 struct {
	mu    sync.RWMutex
	state StateV0
}

func NewStatusTrackerV0(config ConfigV0, now time.Time) *StatusTrackerV0 {
	config = NormalizeConfigV0(config)
	return &StatusTrackerV0{state: StateV0{
		SchemaVersion:   StateSchemaVersionV0,
		Status:          "starting",
		PID:             os.Getpid(),
		Addr:            config.Addr,
		ProjectWorkDir:  config.ProjectWorkDir,
		RuntimeWorkDir:  config.RuntimeWorkDir,
		StartedAt:       formatTimeV0(now),
		LastHeartbeatAt: formatTimeV0(now),
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
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastSupervisorAt = formatTimeV0(now)
		state.LastSupervisorStatus = "ok"
		state.LastSupervisorStop = strings.TrimSpace(result.StopReason)
		state.LastSupervisorError = ""
		state.LastSupervisorQueueRef = strings.TrimSpace(command.QueueRef)
		state.LastSupervisorResultTicks = len(result.Ticks)
		state.LastSupervisorExecutions = result.TotalExecutions
		state.LastSupervisorSkips = result.TotalSkips
		state.SupervisorTicks++
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
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastSupervisorAt = formatTimeV0(now)
		state.LastSupervisorStatus = "error"
		state.LastSupervisorStop = strings.TrimSpace(result.StopReason)
		state.LastSupervisorError = message
		state.LastSupervisorQueueRef = strings.TrimSpace(command.QueueRef)
		state.LastSupervisorResultTicks = len(result.Ticks)
		state.LastSupervisorExecutions = result.TotalExecutions
		state.LastSupervisorSkips = result.TotalSkips
		state.SupervisorTicks++
		state.SupervisorErrorTicks++
		state.LastError = message
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

func formatTimeV0(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
