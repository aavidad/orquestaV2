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

func boolIntV0(value bool) int {
	if value {
		return 1
	}
	return 0
}
