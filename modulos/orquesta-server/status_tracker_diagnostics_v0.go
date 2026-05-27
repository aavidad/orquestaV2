//go:build ignore

package orquestaserver

import (
	"strings"
	"time"
)

const (
	statePersistFailedCodeV0    = "state_persist_failed"
	statePersistConfirmedCodeV0 = "ok"
	maxRecentServerErrorsV0     = 10
)

func (tracker *StatusTrackerV0) MarkErrorV0(message string, now time.Time) StateV0 {
	message = projectServerOperationalMessageV0("server", message)
	return tracker.updateV0(func(state *StateV0) {
		state.Status = "error"
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastError = message
		state.LastErrorOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:      "server",
				ReasonCode: "server_error",
				Status:     "error",
				Message:    message,
			},
		)
		appendRecentServerErrorV0(state, now, "server_error", "server", message, nil)
	})
}

func (tracker *StatusTrackerV0) MarkStatePersistFailedV0(
	transition string,
	now time.Time,
) StateV0 {
	transition = firstNonEmptyServerDiagnosticV0(transition, "state_update")
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.StatePersistStatus = "degraded"
		state.StatePersistFailures++
		state.StatePersistLastFailedAt = formatTimeV0(now)
		state.StatePersistLastCode = statePersistFailedCodeV0
		state.StatePersistLastTransition = transition
		appendRecentServerErrorV0(
			state,
			now,
			statePersistFailedCodeV0,
			"state",
			statePersistFailedCodeV0,
			nil,
		)
	})
}

func (tracker *StatusTrackerV0) MarkStatePersistConfirmedV0(
	transition string,
	now time.Time,
) StateV0 {
	transition = firstNonEmptyServerDiagnosticV0(transition, "state_update")
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.StatePersistStatus = statePersistConfirmedCodeV0
		state.StatePersistLastTransition = transition
		state.StatePersistLastConfirmed = formatTimeV0(now)
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
	item := ServerDiagnosticV0{
		OccurredAt:   formatTimeV0(now),
		Code:         firstNonEmptyServerDiagnosticV0(code, "unknown"),
		Scope:        firstNonEmptyServerDiagnosticV0(scope, "server"),
		Message:      projectServerOperationalMessageV0(scope, message),
		EvidenceRefs: compactServerDiagnosticStringsV0(evidenceRefs),
	}
	state.RecentErrors = append([]ServerDiagnosticV0{item}, state.RecentErrors...)
	if len(state.RecentErrors) > maxRecentServerErrorsV0 {
		state.RecentErrors = state.RecentErrors[:maxRecentServerErrorsV0]
	}
}

func firstNonEmptyServerDiagnosticV0(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func compactServerDiagnosticStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
