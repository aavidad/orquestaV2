package orquestaserver

import (
	"strings"
	"time"
)

type ShutdownProjectionV0 struct {
	Status                  string
	Ready                   bool
	ExitPending             bool
	PID                     int
	HTTPStatus              int
	RunsRequested           int
	RunsStopped             int
	AgentsInFlight          int
	CheckpointsPending      int
	CheckpointAgentsPending int
	AsyncWorkActive         int
	ActiveWorkCount         int
	ActiveWorkRefs          []string
	GoalActions             []ShutdownGoalActionV0
	EvidenceRefs            []string
}

type ShutdownGoalActionV0 struct {
	Kind               string   `json:"kind,omitempty"`
	RunRef             string   `json:"run_ref,omitempty"`
	WorkRef            string   `json:"work_ref,omitempty"`
	ExternalWorkRef    string   `json:"external_work_ref,omitempty"`
	Status             string   `json:"status,omitempty"`
	ActionTaken        string   `json:"action_taken"`
	ActionEvidenceRefs []string `json:"action_evidence_refs,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
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
		state.ShutdownCheckpointAgentsPending = result.CheckpointAgentsPending
		state.ShutdownAsyncWorkActive = nonNegativeServerIntV0(result.AsyncWorkActive)
		state.ShutdownActiveWorkCount = nonNegativeServerIntV0(result.ActiveWorkCount)
		state.ShutdownActiveWorkRefs = compactServerStringsV0(result.ActiveWorkRefs)
		state.ShutdownGoalActions = compactShutdownGoalActionsV0(result.GoalActions)
		state.SupervisorFrozen = keepFrozen
		if result.Ready {
			state.ShutdownStopTimeoutAt = ""
		}
		if keepFrozen {
			state.SupervisorTickActive = false
		}
	})
}

func compactShutdownGoalActionsV0(actions []ShutdownGoalActionV0) []ShutdownGoalActionV0 {
	out := make([]ShutdownGoalActionV0, 0, len(actions))
	seen := map[string]struct{}{}
	for _, action := range actions {
		action = normalizeShutdownGoalActionV0(action)
		if action.ActionTaken == "" ||
			(action.Kind == "" && action.RunRef == "" && action.WorkRef == "" && action.ExternalWorkRef == "") {
			continue
		}
		key := strings.Join([]string{
			action.Kind,
			action.RunRef,
			action.WorkRef,
			action.ExternalWorkRef,
			action.Status,
			action.ActionTaken,
		}, "\x00")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, action)
	}
	if out == nil {
		return []ShutdownGoalActionV0{}
	}
	return out
}

func normalizeShutdownGoalActionV0(action ShutdownGoalActionV0) ShutdownGoalActionV0 {
	action.Kind = strings.TrimSpace(action.Kind)
	action.RunRef = strings.TrimSpace(action.RunRef)
	action.WorkRef = strings.TrimSpace(action.WorkRef)
	action.ExternalWorkRef = strings.TrimSpace(action.ExternalWorkRef)
	action.Status = strings.TrimSpace(action.Status)
	action.ActionTaken = strings.TrimSpace(action.ActionTaken)
	action.ActionEvidenceRefs = compactServerStringsV0(action.ActionEvidenceRefs)
	action.EvidenceRefs = compactServerStringsV0(action.EvidenceRefs)
	return action
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementPreparedV0(
	result IdleSelfImprovementResultV0,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = idleSelfImprovementPreparedReasonV0(result)
		state.IdleSelfImprovementGoalSpec = nil
		if strings.TrimSpace(result.GoalSpec.GoalRef) != "" {
			goalSpec := copyGoalWorkSpecForServerStateV0(result.GoalSpec)
			state.IdleSelfImprovementGoalSpec = &goalSpec
		}
		state.IdleSelfImprovementGoalReceipt = nil
		if strings.TrimSpace(result.GoalReceipt.GoalRef) != "" ||
			strings.TrimSpace(result.GoalReceipt.ExternalGoalRef) != "" {
			goalReceipt := copyGoalLaunchReceiptForServerStateV0(result.GoalReceipt)
			state.IdleSelfImprovementGoalReceipt = &goalReceipt
		}
		state.IdleSelfImprovementGoalResult = nil
		state.IdleSelfImprovementGoalClosure = nil
		state.IdleSelfImprovementGoalUsefulProgressAt = ""
		state.IdleSelfImprovementGoalUsefulProgressSignature = ""
		state.IdleSelfImprovementGoalObservedConsumption = 0
		state.IdleSelfImprovementGoalInvalidCheckpointSignature = ""
		state.IdleSelfImprovementGoalInvalidCheckpointRepeats = 0
		state.IdleSelfImprovementOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "idle_self_improvement",
				ReasonCode:   "prepared",
				Status:       result.Status,
				Message:      result.Message,
				RunRefs:      []string{result.RunRef},
				GoalRefs:     []string{result.GoalRef, result.ExternalGoalRef},
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
	if strings.TrimSpace(result.GoalRef) != "" {
		reason = append(reason, "goal_ref="+strings.TrimSpace(result.GoalRef))
	}
	if strings.TrimSpace(result.ExternalGoalRef) != "" {
		reason = append(reason, "external_goal_ref="+strings.TrimSpace(result.ExternalGoalRef))
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
