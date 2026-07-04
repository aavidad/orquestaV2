package orquestaserver

import "time"

const ServerProcessStaleReasonCodeV0 = "server_process_stale"

func MarkServerProcessStaleStateV0(state StateV0, now time.Time) StateV0 {
	state.SchemaVersion = StateSchemaVersionV0
	state.Status = "stale"
	state.StartupReady = false
	state.StartupStatus = ServerProcessStaleReasonCodeV0
	state.StartupMessage = "server_process_not_alive"
	state.StartupOperationalMessage = projectServerOperationalMessageRecordV0(
		serverOperationalMessageInputV0{
			Scope:      "server",
			ReasonCode: ServerProcessStaleReasonCodeV0,
			Status:     "stale",
			Message:    "server_process_not_alive",
			Counters: map[string]int{
				"pid_recorded": boolIntV0(state.PID > 0),
			},
		},
	)
	state.SupervisorTickActive = false
	state.ResidentDirectorTickActive = false
	state.GoalObserverTickActive = false
	state.ExternalBridgeTickActive = false
	state.ShutdownInProgress = false
	state.ShutdownStatus = ""
	state.ShutdownReady = false
	state.ShutdownHTTPStatus = 0
	state.ShutdownRunsRequested = 0
	state.ShutdownRunsStopped = 0
	state.ShutdownAgentsInFlight = 0
	state.ShutdownCheckpointsPending = 0
	state.ShutdownCheckpointAgentsPending = 0
	state.ShutdownActiveWorkCount = 0
	state.ShutdownActiveWorkRefs = nil
	state.ShutdownGoalActions = nil
	state.ShutdownAsyncWorkActive = 0
	state.ShutdownStopTimeoutAt = ""
	state.LastError = "server_process_not_alive"
	state.LastErrorOperationalMessage = projectServerOperationalMessageRecordV0(
		serverOperationalMessageInputV0{
			Scope:      "server",
			ReasonCode: ServerProcessStaleReasonCodeV0,
			Status:     "stale",
			Message:    "server_process_not_alive",
		},
	)
	appendRecentServerErrorV0(
		&state,
		now,
		ServerProcessStaleReasonCodeV0,
		"server",
		"server_process_not_alive",
		[]string{"evidence-ref-server-statefile-process-not-alive"},
	)
	return state
}

func NormalizeStoppedServerSnapshotV0(state StateV0) (StateV0, bool) {
	if state.Status != "stopped" {
		return state, false
	}
	cleanStartup := stoppedStartupProjectionCleanV0(state)
	cleanShutdown := stoppedShutdownActiveProjectionCleanV0(state)
	if cleanStartup && cleanShutdown {
		return state, false
	}
	state.SchemaVersion = StateSchemaVersionV0
	clearStoppedStartupProjectionV0(&state)
	clearStoppedShutdownActiveProjectionV0(&state)
	return state, true
}

func stoppedStartupProjectionCleanV0(state StateV0) bool {
	return !state.StartupReady &&
		state.StartupStatus == "stopped" &&
		state.StartupMessage == "" &&
		state.StartupOperationalMessage == nil &&
		len(state.StartupBlockers) == 0 &&
		len(state.StartupEvidenceRefs) == 0 &&
		state.StartupRevision == (StartupRevisionSummaryV0{})
}

func stoppedShutdownActiveProjectionCleanV0(state StateV0) bool {
	return !state.ShutdownInProgress &&
		state.ShutdownHTTPStatus == 0 &&
		state.ShutdownActiveWorkCount == 0 &&
		len(state.ShutdownActiveWorkRefs) == 0 &&
		len(state.ShutdownGoalActions) == 0 &&
		state.ShutdownAsyncWorkActive == 0 &&
		state.ShutdownStopTimeoutAt == ""
}

func clearStoppedShutdownActiveProjectionV0(state *StateV0) {
	if state == nil {
		return
	}
	state.ShutdownInProgress = false
	state.ShutdownHTTPStatus = 0
	state.ShutdownActiveWorkCount = 0
	state.ShutdownActiveWorkRefs = nil
	state.ShutdownGoalActions = nil
	state.ShutdownAsyncWorkActive = 0
	state.ShutdownStopTimeoutAt = ""
}

func clearStoppedStartupProjectionV0(state *StateV0) {
	if state == nil {
		return
	}
	state.StartupReady = false
	state.StartupStatus = "stopped"
	state.StartupMessage = ""
	state.StartupOperationalMessage = nil
	state.StartupBlockers = nil
	state.StartupEvidenceRefs = nil
	state.StartupRevision = StartupRevisionSummaryV0{}
}

func boolIntV0(value bool) int {
	if value {
		return 1
	}
	return 0
}
