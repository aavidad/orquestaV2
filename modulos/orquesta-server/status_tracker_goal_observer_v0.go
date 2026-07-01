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
		if observed, ok := goalObserverIdleSelfImprovementObservationV0(*state, result); ok {
			tracker.applyIdleSelfImprovementGoalObservedV0(state, observed, nil, now)
		}
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
		if reason == "unchanged_fingerprint" && supervisorGoalBackendSnapshotActiveV0(supervisorGoalBackendActiveSnapshotFromStateV0(*state)) {
			return
		}
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

func goalObserverIdleSelfImprovementObservationV0(
	state StateV0,
	result orquestagoal.GoalWorkObserveActiveResultV0,
) (orquestagoal.GoalWorkResultV0, bool) {
	expected := map[string]struct{}{}
	if message := state.IdleSelfImprovementOperationalMessage; message != nil {
		for _, ref := range compactConfigStringsV0(message.GoalRefs) {
			expected[ref] = struct{}{}
		}
	}
	if state.IdleSelfImprovementGoalSpec != nil {
		spec := orquestagoal.NormalizeGoalWorkSpecV0(*state.IdleSelfImprovementGoalSpec)
		if strings.TrimSpace(spec.GoalRef) != "" {
			expected[strings.TrimSpace(spec.GoalRef)] = struct{}{}
		}
	}
	if state.IdleSelfImprovementGoalReceipt != nil {
		receipt := copyGoalLaunchReceiptForServerStateV0(*state.IdleSelfImprovementGoalReceipt)
		for _, ref := range []string{receipt.GoalRef, receipt.ExternalGoalRef} {
			ref = strings.TrimSpace(ref)
			if ref != "" {
				expected[ref] = struct{}{}
			}
		}
	}
	if len(expected) == 0 {
		return orquestagoal.GoalWorkResultV0{}, false
	}
	for _, observation := range result.Observations {
		observed := orquestagoal.NormalizeGoalWorkResultV0(observation.Result)
		for _, ref := range goalObserverObservationRefsV0(observation, observed) {
			if _, ok := expected[ref]; ok {
				if strings.TrimSpace(observed.GoalRef) == "" {
					observed.GoalRef = strings.TrimSpace(observation.State.GoalRef)
				}
				if strings.TrimSpace(observed.ExternalGoalRef) == "" {
					observed.ExternalGoalRef = strings.TrimSpace(observation.State.ExternalGoalRef)
				}
				return observed, true
			}
		}
	}
	return orquestagoal.GoalWorkResultV0{}, false
}

func goalObserverObservationRefsV0(
	observation orquestagoal.GoalWorkObserveResultV0,
	result orquestagoal.GoalWorkResultV0,
) []string {
	return compactConfigStringsV0([]string{
		observation.State.GoalRef,
		observation.State.ExternalGoalRef,
		result.GoalRef,
		result.ExternalGoalRef,
	})
}
