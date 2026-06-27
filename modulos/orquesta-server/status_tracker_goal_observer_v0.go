package orquestaserver

import (
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func (tracker *StatusTrackerV0) MarkGoalObserverTickActiveV0(
	active bool,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.GoalObserverTickActive = active
		if active {
			state.GoalObserverStatus = "running"
		}
	})
}

func (tracker *StatusTrackerV0) MarkGoalObserverV0(
	result orquestagoal.GoalWorkObserveActiveResultV0,
	now time.Time,
) StateV0 {
	observed := len(result.Observations)
	terminal := goalObserverTerminalCountV0(result)
	issues := len(result.Issues)
	status := "ok"
	if issues > 0 {
		status = "ok_with_issues"
	}
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.GoalObserverStatus = status
		state.GoalObserverTickActive = false
		state.GoalObserverLastTickAt = formatTimeV0(now)
		state.GoalObserverLastSuccessAt = formatTimeV0(now)
		state.GoalObserverLastErrorAt = ""
		state.GoalObserverLastError = ""
		state.GoalObserverTicks++
		state.GoalObserverObserved += observed
		state.GoalObserverTerminal += terminal
		state.GoalObserverIssues += issues
		state.GoalObserverOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:        "goal_observer",
				ReasonCode:   status,
				Status:       status,
				RunRefs:      goalObserverRunRefsV0(result),
				GoalRefs:     goalObserverGoalRefsV0(result),
				EvidenceRefs: result.EvidenceRefs,
				Counters: map[string]int{
					"observed": observed,
					"terminal": terminal,
					"issues":   issues,
				},
			},
		)
		state.LastError = ""
		state.LastErrorOperationalMessage = nil
	})
}

func (tracker *StatusTrackerV0) MarkGoalObserverErrorV0(
	message string,
	now time.Time,
) StateV0 {
	message = projectServerOperationalMessageV0("goal_observer", message)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.GoalObserverStatus = "error"
		state.GoalObserverTickActive = false
		state.GoalObserverLastTickAt = formatTimeV0(now)
		state.GoalObserverLastErrorAt = formatTimeV0(now)
		state.GoalObserverLastError = message
		state.GoalObserverTicks++
		state.GoalObserverErrorTicks++
		state.GoalObserverOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:      "goal_observer",
				ReasonCode: "goal_observer_error",
				Status:     "error",
				Message:    message,
			},
		)
		state.LastError = message
		state.LastErrorOperationalMessage = copyServerOperationalMessageV0(state.GoalObserverOperationalMessage)
		appendRecentServerErrorV0(
			state,
			now,
			"goal_observer_error",
			"goal_observer",
			message,
			nil,
		)
	})
}

func (tracker *StatusTrackerV0) MarkGoalObserverSkippedV0(
	reason string,
	now time.Time,
) StateV0 {
	reason = firstNonEmptyServerDiagnosticV0(reason, "skipped")
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.GoalObserverStatus = "skipped"
		state.GoalObserverTickActive = false
		state.GoalObserverLastTickAt = formatTimeV0(now)
		state.GoalObserverOperationalMessage = projectServerOperationalMessageRecordV0(
			serverOperationalMessageInputV0{
				Scope:      "goal_observer",
				ReasonCode: reason,
				Status:     "skipped",
			},
		)
	})
}

func goalObserverTerminalCountV0(result orquestagoal.GoalWorkObserveActiveResultV0) int {
	count := 0
	for _, observation := range result.Observations {
		if observation.Terminal || orquestagoal.GoalWorkResultTerminalV0(observation.State.Status) {
			count++
		}
	}
	return count
}

func goalObserverRunRefsV0(result orquestagoal.GoalWorkObserveActiveResultV0) []string {
	refs := make([]string, 0, len(result.Observations)+len(result.Issues))
	for _, observation := range result.Observations {
		refs = append(refs, observation.State.RunRef)
	}
	for _, issue := range result.Issues {
		refs = append(refs, issue.RunRef)
	}
	return compactConfigStringsV0(refs)
}

func goalObserverGoalRefsV0(result orquestagoal.GoalWorkObserveActiveResultV0) []string {
	refs := make([]string, 0, len(result.Observations)+len(result.Issues))
	for _, observation := range result.Observations {
		refs = append(refs, observation.State.GoalRef, observation.State.ExternalGoalRef)
	}
	for _, issue := range result.Issues {
		refs = append(refs, issue.GoalRef)
	}
	out := compactConfigStringsV0(refs)
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	return compactConfigStringsV0(out)
}
