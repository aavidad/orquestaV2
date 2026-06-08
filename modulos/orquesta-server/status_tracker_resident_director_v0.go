package orquestaserver

import (
	"strings"
	"time"
)

func (tracker *StatusTrackerV0) MarkResidentDirectorTickActiveV0(
	active bool,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.ResidentDirectorTickActive = active
		if active {
			state.ResidentDirectorStatus = "running"
		}
	})
}

func (tracker *StatusTrackerV0) MarkResidentDirectorV0(
	result ResidentDirectorResultV0,
	now time.Time,
) StateV0 {
	result = normalizeResidentDirectorResultV0(result)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.ResidentDirectorStatus = "ok"
		state.ResidentDirectorTickActive = false
		state.ResidentDirectorLastTickAt = formatTimeV0(now)
		state.ResidentDirectorLastSuccessAt = formatTimeV0(now)
		state.ResidentDirectorLastErrorAt = ""
		state.ResidentDirectorLastError = ""
		state.ResidentDirectorLastRunRef = result.RunRef
		state.ResidentDirectorLastResult = result.Status
		state.ResidentDirectorTicks++
		state.ResidentDirectorExecutedActions += nonNegativeServerIntV0(result.ExecutedActions)
		state.ResidentDirectorOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "resident_director",
				ReasonCode:   firstNonEmptyServerDiagnosticV0(result.Status, "ok"),
				Status:       "ok",
				RunRefs:      []string{result.RunRef},
				EvidenceRefs: result.EvidenceRefs,
				Counters: map[string]int{
					"executed_actions": nonNegativeServerIntV0(result.ExecutedActions),
				},
			},
		)
		state.LastError = ""
		state.LastErrorOperationalMessage = nil
	})
}

func (tracker *StatusTrackerV0) MarkResidentDirectorErrorV0(
	result ResidentDirectorResultV0,
	message string,
	now time.Time,
) StateV0 {
	result = normalizeResidentDirectorResultV0(result)
	message = projectServerOperationalMessageV0("resident_director", message)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.ResidentDirectorStatus = "error"
		state.ResidentDirectorTickActive = false
		state.ResidentDirectorLastTickAt = formatTimeV0(now)
		state.ResidentDirectorLastErrorAt = formatTimeV0(now)
		state.ResidentDirectorLastError = message
		state.ResidentDirectorLastRunRef = result.RunRef
		state.ResidentDirectorLastResult = firstNonEmptyServerDiagnosticV0(result.Status, "error")
		state.ResidentDirectorTicks++
		state.ResidentDirectorErrorTicks++
		state.ResidentDirectorExecutedActions += nonNegativeServerIntV0(result.ExecutedActions)
		state.ResidentDirectorOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "resident_director",
				ReasonCode:   firstNonEmptyServerDiagnosticV0(result.Status, "resident_director_error"),
				Status:       "error",
				Message:      message,
				RunRefs:      []string{result.RunRef},
				EvidenceRefs: result.EvidenceRefs,
				Counters: map[string]int{
					"executed_actions": nonNegativeServerIntV0(result.ExecutedActions),
				},
			},
		)
		state.LastError = message
		state.LastErrorOperationalMessage = copyServerOperationalMessageV0(state.ResidentDirectorOperationalMessage)
		appendRecentServerErrorV0(
			state,
			now,
			firstNonEmptyServerDiagnosticV0(result.Status, "resident_director_error"),
			"resident_director",
			message,
			result.EvidenceRefs,
		)
	})
}

func (tracker *StatusTrackerV0) MarkResidentDirectorSkippedV0(
	reason string,
	now time.Time,
) StateV0 {
	reason = firstNonEmptyServerDiagnosticV0(reason, "skipped")
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.ResidentDirectorStatus = "skipped"
		state.ResidentDirectorTickActive = false
		state.ResidentDirectorLastTickAt = formatTimeV0(now)
		state.ResidentDirectorLastResult = reason
	})
}

func normalizeResidentDirectorResultV0(result ResidentDirectorResultV0) ResidentDirectorResultV0 {
	result.Status = strings.TrimSpace(result.Status)
	result.RunRef = strings.TrimSpace(result.RunRef)
	result.EvidenceRefs = compactServerDiagnosticStringsV0(result.EvidenceRefs)
	if result.ExecutedActions < 0 {
		result.ExecutedActions = 0
	}
	return result
}
