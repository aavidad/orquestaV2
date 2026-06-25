package orquestaserver

import (
	"strings"
	"time"
)

const idleSelfImprovementPreparePendingStatusV0 = "reconcile_pending"

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementPreparePendingV0(
	result IdleSelfImprovementResultV0,
	now time.Time,
) StateV0 {
	result.Accepted = false
	result.Status = idleSelfImprovementPreparePendingStatusV0
	if strings.TrimSpace(result.Message) == "" {
		result.Message = "wait_for_live_agent_ack"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.IdleSelfImprovementCheck = formatTimeV0(now)
		state.IdleSelfImprovementReason = idleSelfImprovementPreparePendingReasonV0(result)
		state.IdleSelfImprovementOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "idle_self_improvement",
				ReasonCode:   idleSelfImprovementPreparePendingStatusV0,
				Status:       idleSelfImprovementPreparePendingStatusV0,
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
		state.LastError = ""
		state.LastErrorOperationalMessage = nil
	})
}

func idleSelfImprovementPreparePendingV0(result IdleSelfImprovementResultV0) bool {
	return idleSelfImprovementPreparePendingSignalV0(result.Status) ||
		idleSelfImprovementPreparePendingSignalV0(result.EvidenceRefs...) ||
		idleSelfImprovementPreparePendingSignalV0(result.NextActions...)
}

func idleSelfImprovementPreparePendingSignalV0(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case idleSelfImprovementPreparePendingStatusV0,
			"wait",
			"wait_external",
			"running_live",
			"process_live",
			"ack_pending",
			"submit_pending",
			"external_wait_live_process",
			"wait_for_live_agent_ack":
			return true
		}
	}
	return false
}

func idleSelfImprovementPreparePendingReasonV0(result IdleSelfImprovementResultV0) string {
	reason := []string{idleSelfImprovementPreparePendingStatusV0}
	if strings.TrimSpace(result.RunRef) != "" {
		reason = append(reason, "run_ref="+strings.TrimSpace(result.RunRef))
	}
	if strings.TrimSpace(result.RequestRef) != "" {
		reason = append(reason, "request_ref="+strings.TrimSpace(result.RequestRef))
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
