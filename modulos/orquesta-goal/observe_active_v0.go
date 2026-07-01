package orquestagoal

import (
	"context"
	"strings"
)

type GoalWorkObserveActiveRequestV0 struct {
	List GoalWorkStateListRequestV0 `json:"list,omitempty"`
}

type GoalWorkObserveActiveIssueV0 struct {
	RunRef  string `json:"run_ref,omitempty"`
	GoalRef string `json:"goal_ref,omitempty"`
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type GoalWorkObserveActiveResultV0 struct {
	Observations []GoalWorkObserveResultV0      `json:"observations,omitempty"`
	Issues       []GoalWorkObserveActiveIssueV0 `json:"issues,omitempty"`
	EvidenceRefs []string                       `json:"evidence_refs,omitempty"`
}

func ObserveActiveGoalWorksV0(
	ctx context.Context,
	request GoalWorkObserveActiveRequestV0,
	ports GoalWorkLifecyclePortsV0,
) (GoalWorkObserveActiveResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ports.StateStore == nil {
		return GoalWorkObserveActiveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_store"}
	}
	if ports.Observer == nil {
		return GoalWorkObserveActiveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_observer"}
	}
	lister, ok := ports.StateStore.(GoalWorkStateListPortV0)
	if !ok {
		return GoalWorkObserveActiveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_lister"}
	}
	listRequest := defaultActiveGoalWorkStateListRequestV0(request.List)
	states, err := lister.ListGoalWorkStatesV0(ctx, listRequest)
	if err != nil {
		return GoalWorkObserveActiveResultV0{}, err
	}
	out := GoalWorkObserveActiveResultV0{}
	for _, state := range states {
		if !GoalWorkStatePendingObservationV0(state) {
			if !GoalWorkStateShouldReturnActiveSnapshotV0(state, listRequest) {
				continue
			}
			observed, err := GoalWorkObservationSnapshotFromStateV0(state)
			if err != nil {
				out.Issues = append(out.Issues, GoalWorkObserveActiveIssueV0{
					RunRef:  strings.TrimSpace(state.RunRef),
					GoalRef: strings.TrimSpace(state.GoalRef),
					Code:    "goal_state_snapshot_invalid",
					Field:   "goal_state",
					Message: err.Error(),
				})
				continue
			}
			out.Observations = append(out.Observations, observed)
			out.EvidenceRefs = compactGoalStringsV0(append(out.EvidenceRefs, observed.EvidenceRefs...))
			continue
		}
		runRef := strings.TrimSpace(state.RunRef)
		if runRef == "" {
			out.Issues = append(out.Issues, GoalWorkObserveActiveIssueV0{
				GoalRef: strings.TrimSpace(state.GoalRef),
				Code:    "goal_state_run_ref_missing",
				Field:   "run_ref",
			})
			continue
		}
		observed, err := ObserveGoalWorkV0(ctx, GoalWorkObserveRequestV0{RunRef: runRef}, ports)
		if err != nil {
			out.Issues = append(out.Issues, GoalWorkObserveActiveIssueV0{
				RunRef:  runRef,
				GoalRef: strings.TrimSpace(state.GoalRef),
				Code:    "observe_goal_failed",
				Field:   "run_ref",
				Message: err.Error(),
			})
			continue
		}
		out.Observations = append(out.Observations, observed)
		out.EvidenceRefs = compactGoalStringsV0(append(out.EvidenceRefs, observed.EvidenceRefs...))
	}
	return out, nil
}

func defaultActiveGoalWorkStateListRequestV0(
	request GoalWorkStateListRequestV0,
) GoalWorkStateListRequestV0 {
	request = NormalizeGoalWorkStateListRequestV0(request)
	if len(request.Statuses) == 0 {
		request.ActiveOnly = false
		request.Statuses = []string{
			GoalStatusRunningV0,
			GoalStatusCompleteV0,
			GoalStatusBlockedV0,
			GoalStatusInvalidV0,
		}
	}
	return NormalizeGoalWorkStateListRequestV0(request)
}

func GoalWorkObservationSnapshotFromStateV0(
	state GoalWorkStateV0,
) (GoalWorkObserveResultV0, error) {
	state, err := NewGoalWorkStateV0(state)
	if err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	result := NormalizeGoalWorkResultV0(GoalWorkResultV0{
		Status:          state.Status,
		GoalRef:         state.GoalRef,
		ExternalGoalRef: state.ExternalGoalRef,
		EvidenceRefs:    append([]string(nil), state.EvidenceRefs...),
	})
	if state.LastResult != nil {
		result = NormalizeGoalWorkResultV0(*state.LastResult)
		if strings.TrimSpace(result.Status) == "" {
			result.Status = strings.TrimSpace(state.Status)
		}
		if strings.TrimSpace(result.GoalRef) == "" {
			result.GoalRef = strings.TrimSpace(state.GoalRef)
		}
		if strings.TrimSpace(result.ExternalGoalRef) == "" {
			result.ExternalGoalRef = strings.TrimSpace(state.ExternalGoalRef)
		}
	}
	closure := GoalClosureValidationV0{}
	closureEvaluated := false
	if state.LastClosure != nil {
		closure = normalizeGoalClosureSnapshotV0(*state.LastClosure)
		closureEvaluated = true
	}
	terminal := GoalWorkResultTerminalV0(result.Status)
	return GoalWorkObserveResultV0{
		State:            state,
		Result:           result,
		Closure:          closure,
		Terminal:         terminal,
		ClosureEvaluated: closureEvaluated,
		Accepted:         closure.Accepted,
		NeedsRework:      closure.NeedsRework,
		EvidenceRefs: compactGoalStringsV0(append(
			append(append([]string(nil), state.EvidenceRefs...), result.EvidenceRefs...),
			closure.EvidenceRefs...,
		)),
	}, nil
}

func GoalWorkStateShouldReturnActiveSnapshotV0(
	state GoalWorkStateV0,
	request GoalWorkStateListRequestV0,
) bool {
	if GoalWorkStatePendingObservationV0(state) {
		return false
	}
	if len(request.RunRefs) > 0 {
		return true
	}
	switch strings.TrimSpace(state.Status) {
	case GoalStatusBlockedV0, GoalStatusInvalidV0:
		return true
	}
	if state.LastClosure != nil {
		closureStatus := strings.TrimSpace(state.LastClosure.Status)
		if state.LastClosure.NeedsRework || closureStatus == GoalStatusBlockedV0 {
			return true
		}
	}
	return state.LastResult != nil && len(state.LastResult.Issues) > 0
}

func normalizeGoalClosureSnapshotV0(
	closure GoalClosureValidationV0,
) GoalClosureValidationV0 {
	closure.Status = strings.TrimSpace(closure.Status)
	for i := range closure.EvidenceRefs {
		closure.EvidenceRefs[i] = strings.TrimSpace(closure.EvidenceRefs[i])
	}
	for i := range closure.Issues {
		closure.Issues[i].Code = strings.TrimSpace(closure.Issues[i].Code)
		closure.Issues[i].Field = strings.TrimSpace(closure.Issues[i].Field)
	}
	closure.EvidenceRefs = compactGoalStringsV0(closure.EvidenceRefs)
	return closure
}
