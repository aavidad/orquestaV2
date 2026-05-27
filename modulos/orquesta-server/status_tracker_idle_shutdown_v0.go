package orquestaserver

import (
	"strings"
	"time"
)

type ShutdownProjectionV0 struct {
	Status             string
	Ready              bool
	HTTPStatus         int
	RunsRequested      int
	RunsStopped        int
	AgentsInFlight     int
	CheckpointsPending int
}

func (tracker *StatusTrackerV0) MarkShutdownRequestedV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = true
		state.ShutdownStatus = "requested"
		state.ShutdownReady = false
		state.SupervisorFrozen = true
	})
}

func (tracker *StatusTrackerV0) MarkShutdownResultV0(
	result ShutdownProjectionV0,
	keepFrozen bool,
	now time.Time,
) StateV0 {
	result.Status = strings.TrimSpace(result.Status)
	if result.Status == "" {
		result.Status = "observed"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = keepFrozen
		state.ShutdownStatus = result.Status
		state.ShutdownReady = result.Ready
		state.ShutdownHTTPStatus = result.HTTPStatus
		state.ShutdownRunsRequested = result.RunsRequested
		state.ShutdownRunsStopped = result.RunsStopped
		state.ShutdownAgentsInFlight = result.AgentsInFlight
		state.ShutdownCheckpointsPending = result.CheckpointsPending
		state.SupervisorFrozen = keepFrozen
		if keepFrozen {
			state.SupervisorTickActive = false
		}
	})
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementPreparedV0(
	result IdleSelfImprovementResultV0,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = idleSelfImprovementPreparedReasonV0(result)
		state.IdleSelfImprovementOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "idle_self_improvement",
				ReasonCode:   "prepared",
				Status:       result.Status,
				Message:      result.Message,
				RunRefs:      []string{result.RunRef},
				RequestRefs:  []string{result.RequestRef},
				EvidenceRefs: result.EvidenceRefs,
				Counters: map[string]int{
					"attempts": tracker.idleSelfImprovementAttempts + 1,
					"accepted": tracker.idleSelfImprovementPrepared + boolToServerCounterV0(result.Accepted),
					"next":     len(result.NextActions),
				},
			},
		)
		tracker.lastIdleSelfImprovementAt = now.UTC()
		tracker.idleSelfImprovementInFlight = false
		tracker.idleSelfImprovementAccepted = result.Accepted
		tracker.idleSelfImprovementAttempts++
		if result.Accepted {
			tracker.idleSelfImprovementPrepared++
		}
		state.IdleSelfImprovementFlight = false
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
		state.LastError = ""
		state.LastErrorOperationalMessage = nil
	})
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementScheduledV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = "scheduled"
		state.IdleSelfImprovementOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:      "idle_self_improvement",
				ReasonCode: "scheduled",
				Status:     "scheduled",
				Counters: map[string]int{
					"attempts": tracker.idleSelfImprovementAttempts,
					"accepted": tracker.idleSelfImprovementPrepared,
				},
			},
		)
		tracker.lastIdleSelfImprovementAt = now.UTC()
		tracker.idleSelfImprovementInFlight = true
		tracker.idleSelfImprovementAccepted = false
		state.IdleSelfImprovementFlight = true
		state.IdleSelfImprovementRuns = tracker.idleSelfImprovementAttempts
		state.IdleSelfImprovementOK = tracker.idleSelfImprovementPrepared
	})
}

func idleSelfImprovementPreparedReasonV0(result IdleSelfImprovementResultV0) string {
	reason := []string{"prepared"}
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
		reason = append(reason, "evidence="+strings.Join(result.EvidenceRefs, ","))
	}
	return strings.Join(reason, ";")
}
