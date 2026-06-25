package orquestaserver

import (
	"strings"
	"time"
)

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementErrorV0(message string, now time.Time) StateV0 {
	message = projectServerOperationalMessageV0("idle_self_improvement", message)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = "error:" + message
		state.IdleSelfImprovementOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:      "idle_self_improvement",
				ReasonCode: "idle_self_improvement_error",
				Status:     "error",
				Message:    message,
				Counters: map[string]int{
					"attempts": tracker.idleSelfImprovementAttempts + 1,
					"accepted": tracker.idleSelfImprovementPrepared,
				},
			},
		)
		tracker.lastIdleSelfImprovementAt = now.UTC()
		tracker.idleSelfImprovementInFlight = false
		tracker.idleSelfImprovementAccepted = false
		tracker.idleSelfImprovementAttempts++
		state.IdleSelfImprovementFlight = false
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
		state.LastError = message
		state.LastErrorOperationalMessage = copyServerOperationalMessageV0(
			state.IdleSelfImprovementOperationalMessage,
		)
		appendRecentServerErrorV0(state, now, "idle_self_improvement_error", "idle_self_improvement", message, nil)
	})
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementPrepareFailedV0(
	result IdleSelfImprovementResultV0,
	now time.Time,
) StateV0 {
	result.Accepted = false
	if strings.TrimSpace(result.Status) == "" {
		result.Status = "error"
	}
	if strings.TrimSpace(result.Message) == "" {
		result.Message = "prepare_failed"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = idleSelfImprovementPrepareFailedReasonV0(result)
		state.IdleSelfImprovementOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "idle_self_improvement",
				ReasonCode:   "prepare_failed",
				Status:       result.Status,
				Message:      result.Message,
				RunRefs:      []string{result.RunRef},
				GoalRefs:     []string{result.GoalRef, result.ExternalGoalRef},
				RequestRefs:  []string{result.RequestRef},
				EvidenceRefs: result.EvidenceRefs,
				Counters: map[string]int{
					"attempts": tracker.idleSelfImprovementAttempts + 1,
					"accepted": tracker.idleSelfImprovementPrepared,
					"next":     len(result.NextActions),
				},
			},
		)
		tracker.lastIdleSelfImprovementAt = now.UTC()
		tracker.idleSelfImprovementInFlight = false
		tracker.idleSelfImprovementAccepted = false
		tracker.idleSelfImprovementAttempts++
		state.IdleSelfImprovementFlight = false
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
		state.LastError = projectServerOperationalMessageV0("idle_self_improvement", result.Message)
		state.LastErrorOperationalMessage = copyServerOperationalMessageV0(
			state.IdleSelfImprovementOperationalMessage,
		)
		appendRecentServerErrorV0(
			state,
			now,
			"idle_self_improvement_prepare_failed",
			"idle_self_improvement",
			result.Message,
			result.EvidenceRefs,
		)
	})
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementCheckedV0(reason string, now time.Time) StateV0 {
	reason = projectServerOperationalMessageV0("idle_self_improvement", reason)
	return tracker.updateV0(func(state *StateV0) {
		if reason == "attempt_blocked" {
			if tracker.idleSelfImprovementAccepted &&
				strings.HasPrefix(state.IdleSelfImprovementReason, "prepared") {
				reason = state.IdleSelfImprovementReason
				if !strings.Contains(reason, "pending=accepted_attempt") {
					reason += ";pending=accepted_attempt"
				}
			}
			if strings.HasPrefix(state.IdleSelfImprovementReason, "error:prepare_failed") {
				reason = state.IdleSelfImprovementReason
				if !strings.Contains(reason, "pending=failed_attempt") {
					reason += ";pending=failed_attempt"
				}
			}
		}
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = reason
		state.IdleSelfImprovementOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:      "idle_self_improvement",
				ReasonCode: idleSelfImprovementReasonCodeV0(reason),
				Status:     "checked",
				Message:    reason,
				Counters: map[string]int{
					"attempts": tracker.idleSelfImprovementAttempts,
					"accepted": tracker.idleSelfImprovementPrepared,
				},
			},
		)
		state.IdleSelfImprovementFlight = tracker.idleSelfImprovementInFlight
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
	})
}

func idleSelfImprovementPrepareFailedReasonV0(result IdleSelfImprovementResultV0) string {
	reason := []string{"error:prepare_failed"}
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
		reason = append(reason, "evidence="+strings.Join(compactServerDiagnosticStringsV0(result.EvidenceRefs), ","))
	}
	return strings.Join(compactConfigStringsV0(reason), ";")
}

func (tracker *StatusTrackerV0) MarkErrorV0(message string, now time.Time) StateV0 {
	message = projectServerOperationalMessageV0("server", message)
	return tracker.updateV0(func(state *StateV0) {
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
