package orquestaserver

import (
	"context"
	"strings"
	"time"
)

func restoreStatusTrackerFromStoreV0(
	ctx context.Context,
	config ConfigV0,
	store StateStorePortV0,
	now time.Time,
) (*StatusTrackerV0, bool) {
	if store == nil {
		return nil, false
	}
	state, err := store.LoadServerStateV0(ctx)
	if err != nil || state.SchemaVersion != StateSchemaVersionV0 {
		return nil, false
	}
	return NewStatusTrackerFromDurableStateV0(config, state, now), true
}

func NewStatusTrackerFromDurableStateV0(config ConfigV0, durable StateV0, now time.Time) *StatusTrackerV0 {
	tracker := NewStatusTrackerV0(config, now)
	if durable.SchemaVersion != StateSchemaVersionV0 {
		return tracker
	}
	fresh := tracker.state
	restored := durable
	restored.SchemaVersion = fresh.SchemaVersion
	restored.Status = "starting"
	restored.PID = fresh.PID
	restored.Addr = fresh.Addr
	restored.ProcessRef = fresh.ProcessRef
	restored.DaemonEpochRef = fresh.DaemonEpochRef
	restored.ProjectWorkDir = fresh.ProjectWorkDir
	restored.RuntimeWorkDir = fresh.RuntimeWorkDir
	restored.RuntimeIdentity = fresh.RuntimeIdentity
	restored.DaemonLogPolicy = fresh.DaemonLogPolicy
	restored.ShutdownSignalPolicy = fresh.ShutdownSignalPolicy
	restored.EffectiveConfig = fresh.EffectiveConfig
	restored.StartedAt = fresh.StartedAt
	restored.LastHeartbeatAt = fresh.LastHeartbeatAt
	restored.StartupStatus = ""
	restored.StartupReady = false
	restored.StartupMessage = ""
	restored.StartupOperationalMessage = nil
	restored.SupervisorTickActive = false
	restored.ResidentDirectorTickActive = false
	restored.GoalObserverTickActive = false
	if strings.TrimSpace(restored.GoalObserverStatus) == "running" {
		restored.GoalObserverStatus = ""
	}
	switch strings.TrimSpace(restored.ResidentDirectorStatus) {
	case "running", "paused", "resumed":
		restored.ResidentDirectorStatus = ""
	}
	restored.SupervisorFrozen = false
	restored.ShutdownInProgress = false
	restored.ShutdownStatus = ""
	restored.ShutdownReady = false
	restored.ShutdownHTTPStatus = 0
	restored.ShutdownRunsRequested = 0
	restored.ShutdownRunsStopped = 0
	restored.ShutdownAgentsInFlight = 0
	restored.ShutdownCheckpointsPending = 0
	restored.ShutdownAsyncWorkActive = 0
	restored.ShutdownStopTimeoutAt = ""
	restored.IdleSelfImprovementAfter = fresh.IdleSelfImprovementAfter
	restored.IdleSelfImprovementTarget = fresh.IdleSelfImprovementTarget
	restored.IdleSelfImprovementFlight = false
	if restored.IdleSelfImprovementReason == "shutdown_in_progress" {
		restored.IdleSelfImprovementReason = ""
		restored.IdleSelfImprovementOperationalMessage = nil
	}
	tracker.state = restored
	tracker.restoreIdleSelfImprovementWindowV0(restored)
	return tracker
}

func (tracker *StatusTrackerV0) restoreIdleSelfImprovementWindowV0(state StateV0) {
	tracker.idleSelfImprovementAttempts = nonNegativeServerIntV0(state.IdleSelfImprovementRuns)
	tracker.idleSelfImprovementPrepared = nonNegativeServerIntV0(state.IdleSelfImprovementOK)
	tracker.idleSelfImprovementInFlight = false
	reasonCode := ""
	if state.IdleSelfImprovementOperationalMessage != nil {
		reasonCode = state.IdleSelfImprovementOperationalMessage.ReasonCode
	}
	tracker.idleSelfImprovementAccepted = strings.HasPrefix(state.IdleSelfImprovementReason, "prepared") ||
		idleSelfImprovementGoalObservationReasonV0(state.IdleSelfImprovementReason, reasonCode)
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(state.IdleSelfImprovementCheck)); err == nil {
		tracker.lastIdleSelfImprovementAt = parsed.UTC()
	}
}
