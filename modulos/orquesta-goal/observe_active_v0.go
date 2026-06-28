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
		request.Statuses = []string{GoalStatusRunningV0, GoalStatusCompleteV0}
	}
	return NormalizeGoalWorkStateListRequestV0(request)
}
