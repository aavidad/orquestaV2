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

func (tracker *StatusTrackerV0) MarkSupervisorV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastSupervisorAt = formatTimeV0(now)
		state.LastSupervisorStop = strings.TrimSpace(result.StopReason)
		state.SupervisorTicks++
		state.LastError = ""
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
