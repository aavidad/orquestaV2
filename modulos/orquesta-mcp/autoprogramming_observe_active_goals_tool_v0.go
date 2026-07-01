package orquestamcp

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	MCPAutoprogrammingObserveActiveGoalsToolNameV0    = "orquesta.autoprogramming.observe_active_goals.v0"
	MCPAutoprogrammingObserveActiveGoalsToolVersionV0 = "v0"
	MCPAutoprogrammingObserveActiveGoalsResourceURIV0 = "orquesta://contracts/autoprogramming-observe-active-goals/v0"
	MCPAutoprogrammingObserveActiveGoalsHTTPPathV0    = "/api/v0/autoprogramming/goals/observe-active"
	MCPAutoprogrammingObserveActiveGoalsEstadoOKV0    = "ok"
	MCPAutoprogrammingObserveActiveGoalsEstadoErrorV0 = "error"
)

type MCPAutoprogrammingObserveActiveGoalsToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAutoprogrammingObserveActiveGoalsToolInputV0 struct {
	RequestID     string   `json:"request_id,omitempty"`
	CorrelationID string   `json:"correlation_id,omitempty"`
	RunRefs       []string `json:"run_refs,omitempty"`
	Statuses      []string `json:"statuses,omitempty"`
	MaxItems      int      `json:"max_items,omitempty"`
	OccurredAt    string   `json:"occurred_at,omitempty"`
	RequestedBy   string   `json:"requested_by,omitempty"`
}

type MCPAutoprogrammingObserveActiveGoalsToolResultV0 struct {
	Estado        string                                      `json:"estado"`
	RequestID     string                                      `json:"request_id,omitempty"`
	CorrelationID string                                      `json:"correlation_id,omitempty"`
	OperationRef  string                                      `json:"operation_ref,omitempty"`
	Observations  []MCPAutoprogrammingObserveGoalToolResultV0 `json:"observations,omitempty"`
	Issues        []MCPValidationIssueV0                      `json:"issues,omitempty"`
	NextActions   []string                                    `json:"next_actions,omitempty"`
	Diagnostics   []MCPAutoprogrammingDiagnosticV0            `json:"diagnostics,omitempty"`
	EvidenceRefs  []string                                    `json:"evidence_refs,omitempty"`
	Errores       []MCPValidationIssueV0                      `json:"errores_publicos,omitempty"`
}

type MCPAutoprogrammingObserveActiveGoalsToolExecutorV0 struct {
	GoalStateStore orquestagoal.GoalWorkStateStorePortV0
	ObserveGoal    MCPTransportAutoprogrammingObserveGoalExecutorV0
}

func MCPAutoprogrammingObserveActiveGoalsDescriptorV0() MCPAutoprogrammingObserveActiveGoalsToolDescriptorV0 {
	return MCPAutoprogrammingObserveActiveGoalsToolDescriptorV0{
		Name:        MCPAutoprogrammingObserveActiveGoalsToolNameV0,
		Version:     MCPAutoprogrammingObserveActiveGoalsToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_refs?,statuses?,max_items?,occurred_at?,requested_by?}",
		Output:      "ok:{observations?,issues?,evidence_refs?}|error:{errores_publicos}",
		ResourceURI: MCPAutoprogrammingObserveActiveGoalsResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"lista goals de autoprogramacion activos por puerto durable",
			"observa cada run con observe_goal para reutilizar cierre y reconciliacion de composicion",
			"no ejecuta supervise legacy ni arranca proveedor",
			"un fallo por goal no cancela la pasada de los demas",
		},
	}
}

func NewMCPAutoprogrammingObserveActiveGoalsToolExecutorV0(
	goalStateStore orquestagoal.GoalWorkStateStorePortV0,
	observeGoal MCPTransportAutoprogrammingObserveGoalExecutorV0,
) MCPAutoprogrammingObserveActiveGoalsToolExecutorV0 {
	return MCPAutoprogrammingObserveActiveGoalsToolExecutorV0{
		GoalStateStore: goalStateStore,
		ObserveGoal:    observeGoal,
	}
}

func (executor MCPAutoprogrammingObserveActiveGoalsToolExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) (MCPAutoprogrammingObserveActiveGoalsToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := newMCPAutoprogrammingObserveActiveGoalsBaseV0(input)
	if executor.GoalStateStore == nil {
		return mcpAutoprogrammingObserveActiveGoalsErrorV0(input, "autoprogramming_goal_state_store_unbound", "goal_state_store", "goal_state_store no configurado"), nil
	}
	if executor.ObserveGoal == nil {
		return mcpAutoprogrammingObserveActiveGoalsErrorV0(input, "autoprogramming_observe_goal_unbound", "observe_goal", "observe_goal no configurado"), nil
	}
	lister, ok := executor.GoalStateStore.(orquestagoal.GoalWorkStateListPortV0)
	if !ok {
		return mcpAutoprogrammingObserveActiveGoalsErrorV0(input, "autoprogramming_goal_state_lister_unbound", "goal_state_store", "goal_state_store no lista goals activos"), nil
	}
	listRequest := mcpAutoprogrammingObserveActiveGoalsListRequestV0(input)
	states, err := lister.ListGoalWorkStatesV0(ctx, listRequest)
	if err != nil {
		return mcpAutoprogrammingObserveActiveGoalsErrorV0(input, "autoprogramming_goal_state_list_failed", "goal_state_store", publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_goal_state_list_failed", err)), nil
	}
	for _, state := range states {
		if !orquestagoal.GoalWorkStatePendingObservationV0(state) {
			if !orquestagoal.GoalWorkStateShouldReturnActiveSnapshotV0(state, listRequest) {
				continue
			}
			runRef := strings.TrimSpace(state.RunRef)
			observed, err := NewMCPObserveAppDirectorGoalPartialResultFromStateV0(
				MCPObserveAppDirectorGoalToolInputV0{
					RequestID:     strings.TrimSpace(input.RequestID),
					CorrelationID: strings.TrimSpace(input.CorrelationID),
					RunRef:        runRef,
					OccurredAt:    strings.TrimSpace(input.OccurredAt),
					RequestedBy:   firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-observe-active-goals"),
				},
				state,
			)
			if err != nil {
				result.Issues = append(result.Issues, MCPValidationIssueV0{
					Code:    "autoprogramming_goal_state_snapshot_invalid",
					Field:   firstNonEmptyMCPV0(runRef, strings.TrimSpace(state.GoalRef), "goal_state"),
					Message: publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_goal_state_snapshot_invalid", err),
				})
				continue
			}
			result.Observations = append(result.Observations, observed)
			mcpAutoprogrammingObserveActiveGoalsRecordObservationV0(&result, observed)
			continue
		}
		runRef := strings.TrimSpace(state.RunRef)
		if runRef == "" {
			result.Issues = append(result.Issues, MCPValidationIssueV0{
				Code:    "autoprogramming_goal_state_run_ref_missing",
				Field:   "run_ref",
				Message: "goal state sin run_ref",
			})
			continue
		}
		observed, err := executor.ObserveGoal.Execute(ctx, MCPAutoprogrammingObserveGoalToolInputV0{
			RequestID:     strings.TrimSpace(input.RequestID),
			CorrelationID: strings.TrimSpace(input.CorrelationID),
			RunRef:        runRef,
			OccurredAt:    strings.TrimSpace(input.OccurredAt),
			RequestedBy:   firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-observe-active-goals"),
		})
		if err != nil {
			result.Issues = append(result.Issues, MCPValidationIssueV0{
				Code:    "autoprogramming_observe_goal_failed",
				Field:   runRef,
				Message: publicMCPExecutorErrorMessageFromErrorV0("autoprogramming_observe_goal_failed", err),
			})
			continue
		}
		result.Observations = append(result.Observations, observed)
		mcpAutoprogrammingObserveActiveGoalsRecordObservationV0(&result, observed)
		if strings.TrimSpace(observed.Estado) != MCPAutoprogrammingObserveGoalEstadoOKV0 {
			result.Issues = append(result.Issues, MCPValidationIssueV0{
				Code:    "autoprogramming_observe_goal_non_ok",
				Field:   runRef,
				Message: firstNonEmptyMCPV0(observed.Estado, "autoprogramming_observe_goal_non_ok"),
			})
		}
		for _, issue := range observed.Errores {
			result.Issues = append(result.Issues, MCPValidationIssueV0{
				Code:    firstNonEmptyMCPV0(issue.Code, "autoprogramming_observe_goal_error"),
				Field:   firstNonEmptyMCPV0(issue.Field, runRef),
				Message: firstNonEmptyMCPV0(issue.Message, issue.Code, "autoprogramming_observe_goal_error"),
			})
		}
	}
	return result, nil
}

func mcpAutoprogrammingObserveActiveGoalsListRequestV0(
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) orquestagoal.GoalWorkStateListRequestV0 {
	request := orquestagoal.NormalizeGoalWorkStateListRequestV0(orquestagoal.GoalWorkStateListRequestV0{
		RunRefs:  input.RunRefs,
		Statuses: input.Statuses,
		MaxItems: input.MaxItems,
	})
	if len(request.Statuses) == 0 {
		request.ActiveOnly = false
		request.Statuses = []string{
			orquestagoal.GoalStatusRunningV0,
			orquestagoal.GoalStatusCompleteV0,
			orquestagoal.GoalStatusBlockedV0,
			orquestagoal.GoalStatusInvalidV0,
		}
	}
	return orquestagoal.NormalizeGoalWorkStateListRequestV0(request)
}

func mcpAutoprogrammingObserveActiveGoalsRecordObservationV0(
	result *MCPAutoprogrammingObserveActiveGoalsToolResultV0,
	observed MCPAutoprogrammingObserveGoalToolResultV0,
) {
	if result == nil {
		return
	}
	runRef := strings.TrimSpace(observed.RunRef)
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, observed.EvidenceRefs...))
	for _, issue := range observed.ClosureIssues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			continue
		}
		result.Issues = append(result.Issues, MCPValidationIssueV0{
			Code:    code,
			Field:   firstNonEmptyMCPV0(runRef, strings.TrimSpace(issue.Field), "goal"),
			Message: firstNonEmptyMCPV0(issue.Message, code),
		})
	}
	if mcpObserveAppDirectorGoalLooksUsageLimitedV0(observed) {
		result.Issues = append(result.Issues, MCPValidationIssueV0{
			Code:    "autoprogramming_goal_backend_limited",
			Field:   firstNonEmptyMCPV0(runRef, "goal_backend"),
			Message: "goal_backend_limited",
		})
		result.NextActions = compactStringsMCPV0(append(
			result.NextActions,
			"inspect_goal_backend_limits",
			"wait_for_goal_backend_capacity",
			"do_not_relaunch_until_limit_clears",
		))
		return
	}
	switch strings.TrimSpace(observed.GoalStatus) {
	case orquestagoal.GoalStatusBlockedV0, orquestagoal.GoalStatusInvalidV0:
		result.NextActions = compactStringsMCPV0(append(
			result.NextActions,
			"inspect_goal_snapshot",
			"reconcile_partial_artifacts",
		))
	}
}

func newMCPAutoprogrammingObserveActiveGoalsBaseV0(
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
) MCPAutoprogrammingObserveActiveGoalsToolResultV0 {
	return MCPAutoprogrammingObserveActiveGoalsToolResultV0{
		Estado:        MCPAutoprogrammingObserveActiveGoalsEstadoOKV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Errores:       []MCPValidationIssueV0{},
	}
}

func mcpAutoprogrammingObserveActiveGoalsErrorV0(
	input MCPAutoprogrammingObserveActiveGoalsToolInputV0,
	code string,
	field string,
	message string,
) MCPAutoprogrammingObserveActiveGoalsToolResultV0 {
	result := newMCPAutoprogrammingObserveActiveGoalsBaseV0(input)
	result.Estado = MCPAutoprogrammingObserveActiveGoalsEstadoErrorV0
	result.Errores = []MCPValidationIssueV0{{
		Code:    strings.TrimSpace(code),
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(firstNonEmptyMCPV0(message, code)),
	}}
	return result
}
