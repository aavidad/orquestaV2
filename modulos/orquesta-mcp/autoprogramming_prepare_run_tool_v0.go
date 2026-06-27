package orquestamcp

import (
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	MCPAutoprogrammingPrepareRunToolNameV0    = "orquesta.autoprogramming.prepare_run.v0"
	MCPAutoprogrammingPrepareRunToolVersionV0 = "v0"
	MCPAutoprogrammingPrepareRunResourceURIV0 = "orquesta://contracts/autoprogramming-prepare-run/v0"
	MCPAutoprogrammingPrepareRunEstadoOKV0    = "ok"
	MCPAutoprogrammingPrepareRunEstadoErrorV0 = "error"
)

type MCPAutoprogrammingPrepareRunToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAutoprogrammingPrepareRunToolInputV0 struct {
	RequestID              string                                           `json:"request_id,omitempty"`
	CorrelationID          string                                           `json:"correlation_id,omitempty"`
	IdempotencyKey         string                                           `json:"idempotency_key,omitempty"`
	OccurredAt             string                                           `json:"occurred_at,omitempty"`
	RequestedBy            string                                           `json:"requested_by,omitempty"`
	AutoprogrammingRequest orquestaautoprogramming.AutoprogrammingRequestV0 `json:"autoprogramming_request"`
	MaxBursts              int                                              `json:"max_bursts,omitempty"`
	MaxStepsPerBurst       int                                              `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait   int                                              `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands            int                                              `json:"max_commands,omitempty"`
	MaxOutboxPerCycle      int                                              `json:"max_outbox_per_cycle,omitempty"`
	PriorityScore          int                                              `json:"priority_score,omitempty"`
}

type MCPAutoprogrammingContinueRequestV0 struct {
	RunRef                     string   `json:"run_ref"`
	OperationalDirectorPlanRef string   `json:"operational_director_plan_ref,omitempty"`
	OccurredAt                 string   `json:"occurred_at,omitempty"`
	CorrelationID              string   `json:"correlation_id,omitempty"`
	RequestedBy                string   `json:"requested_by,omitempty"`
	MaxBursts                  int      `json:"max_bursts,omitempty"`
	MaxStepsPerBurst           int      `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait       int      `json:"max_dispatches_per_wait,omitempty"`
	WaitAgentRefs              []string `json:"wait_agent_refs,omitempty"`
	MaxCommands                int      `json:"max_commands,omitempty"`
	MaxOutboxPerCycle          int      `json:"max_outbox_per_cycle,omitempty"`
}

type MCPAutoprogrammingPrepareRunToolResultV0 struct {
	Estado           string                               `json:"estado"`
	RequestID        string                               `json:"request_id,omitempty"`
	CorrelationID    string                               `json:"correlation_id,omitempty"`
	Accepted         bool                                 `json:"accepted"`
	RunRef           string                               `json:"run_ref,omitempty"`
	ProjectRef       string                               `json:"project_ref,omitempty"`
	WorktreeRef      string                               `json:"worktree_ref,omitempty"`
	BranchRef        string                               `json:"branch_ref,omitempty"`
	PhaseID          string                               `json:"phase_id,omitempty"`
	WorkflowTaskRefs []string                             `json:"workflow_task_refs,omitempty"`
	WaitAgentRefs    []string                             `json:"wait_agent_refs,omitempty"`
	GoalSpecs        []orquestagoal.GoalWorkSpecV0        `json:"goal_specs,omitempty"`
	Goal             *MCPAutoprogrammingGoalRunV0         `json:"goal,omitempty"`
	Goals            []MCPAutoprogrammingGoalRunV0        `json:"goals,omitempty"`
	Continue         *MCPAutoprogrammingContinueRequestV0 `json:"continue,omitempty"`
	Errores          []MCPValidationIssueV0               `json:"errores_publicos,omitempty"`
}

type MCPAutoprogrammingGoalRunV0 struct {
	DirectorExecutionMode string   `json:"director_execution_mode,omitempty"`
	RunRef                string   `json:"run_ref,omitempty"`
	GoalRef               string   `json:"goal_ref"`
	ExternalGoalRef       string   `json:"external_goal_ref,omitempty"`
	GoalStatus            string   `json:"goal_status,omitempty"`
	EvidenceRefs          []string `json:"evidence_refs,omitempty"`
}

func MCPAutoprogrammingPrepareRunDescriptorV0() MCPAutoprogrammingPrepareRunToolDescriptorV0 {
	return MCPAutoprogrammingPrepareRunToolDescriptorV0{
		Name:        MCPAutoprogrammingPrepareRunToolNameV0,
		Version:     MCPAutoprogrammingPrepareRunToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,idempotency_key?,occurred_at?,requested_by?,autoprogramming_request:AutoprogrammingRequestV0,max_bursts?,max_steps_per_burst?,max_dispatches_per_wait?,max_commands?,max_outbox_per_cycle?,priority_score?}",
		Output:      "ok:{run_ref?,workflow_task_refs?,wait_agent_refs?,goal_specs?,goal?{run_ref,goal_ref,external_goal_ref?,goal_status,evidence_refs?},goals?[]{run_ref,goal_ref,external_goal_ref?,goal_status,evidence_refs?},continue?{run_ref,operational_director_plan_ref?}}|error:{errores_publicos}",
		ResourceURI: MCPAutoprogrammingPrepareRunResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"prepara run legacy continuable o lanza/hace handoff goal-first por executor inyectado",
			"goal-first es la ruta preferente cuando devuelve goal o goal_specs; continue queda como compatibilidad legacy",
			"en batch goal-first goals[] es canonico; goal se omite y run_ref superior identifica el primer goal, no un run agregado",
			"no arranca agentes por si mismo",
			"no conoce Codex OPES DB filesystem ni proveedor concreto",
			"la supervision legacy posterior debe usar run_ref explicito; goal-first se observa por run_ref del goal",
		},
	}
}

func normalizeMCPAutoprogrammingPrepareRunIdentityV0(
	input MCPAutoprogrammingPrepareRunToolInputV0,
	headerCorrelationID string,
	headerIdempotencyKey string,
) (MCPAutoprogrammingPrepareRunToolInputV0, []MCPValidationIssueV0) {
	identity := NormalizeMCPPublicMutationIdentityV0(MCPPublicMutationIdentityInputV0{
		RequestID:            input.RequestID,
		CorrelationID:        input.CorrelationID,
		IdempotencyKey:       input.IdempotencyKey,
		HeaderCorrelationID:  headerCorrelationID,
		HeaderIdempotencyKey: headerIdempotencyKey,
		Mutating:             true,
	})
	input.RequestID = identity.RequestID
	input.CorrelationID = identity.CorrelationID
	input.IdempotencyKey = identity.IdempotencyKey
	return input, identity.Issues
}

func NewMCPAutoprogrammingPrepareRunErrorResultV0(
	input MCPAutoprogrammingPrepareRunToolInputV0,
	code string,
	field string,
	message string,
) MCPAutoprogrammingPrepareRunToolResultV0 {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "autoprogramming_prepare_run_error"
	}
	return MCPAutoprogrammingPrepareRunToolResultV0{
		Estado:        MCPAutoprogrammingPrepareRunEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Accepted:      false,
		Errores: []MCPValidationIssueV0{{
			Code:    code,
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(firstNonEmptyMCPV0(message, code)),
		}},
	}
}

func NewMCPAutoprogrammingPrepareRunIssuesResultV0(
	input MCPAutoprogrammingPrepareRunToolInputV0,
	issues []MCPValidationIssueV0,
) MCPAutoprogrammingPrepareRunToolResultV0 {
	out := make([]MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			code = "autoprogramming_prepare_run_error"
		}
		message := strings.TrimSpace(issue.Message)
		if message == "" {
			message = code
		}
		out = append(out, MCPValidationIssueV0{
			Code:    code,
			Field:   strings.TrimSpace(issue.Field),
			Message: message,
		})
	}
	if out == nil {
		out = []MCPValidationIssueV0{}
	}
	return MCPAutoprogrammingPrepareRunToolResultV0{
		Estado:        MCPAutoprogrammingPrepareRunEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Accepted:      false,
		Errores:       out,
	}
}
