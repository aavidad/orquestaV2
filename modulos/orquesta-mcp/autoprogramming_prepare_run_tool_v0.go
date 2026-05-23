package orquestamcp

import (
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
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
	Continue         *MCPAutoprogrammingContinueRequestV0 `json:"continue,omitempty"`
	Errores          []MCPValidationIssueV0               `json:"errores_publicos,omitempty"`
}

func MCPAutoprogrammingPrepareRunDescriptorV0() MCPAutoprogrammingPrepareRunToolDescriptorV0 {
	return MCPAutoprogrammingPrepareRunToolDescriptorV0{
		Name:        MCPAutoprogrammingPrepareRunToolNameV0,
		Version:     MCPAutoprogrammingPrepareRunToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,autoprogramming_request,limits?,priority_score?}",
		Output:      "ok:{run_ref,workflow_task_refs,wait_agent_refs,continue{operational_director_plan_ref?}}|error:{errores_publicos}",
		ResourceURI: MCPAutoprogrammingPrepareRunResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"prepara run continuable por executor inyectado",
			"no arranca agentes por si mismo",
			"no conoce Codex OPES DB filesystem ni proveedor concreto",
			"la supervision posterior debe usar run_ref explicito",
		},
	}
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
