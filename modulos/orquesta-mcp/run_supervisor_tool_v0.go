package orquestamcp

import "strings"

const (
	MCPRunSupervisorToolNameV0    = "orquesta.runs.supervisor.v0"
	MCPRunSupervisorToolVersionV0 = "v0"
	MCPRunSupervisorResourceURIV0 = "orquesta://contracts/run-supervisor/v0"
	MCPRunSupervisorEstadoOKV0    = "ok"
	MCPRunSupervisorEstadoErrorV0 = "error"
)

type MCPRunSupervisorToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPRunSupervisorToolInputV0 struct {
	RequestID                  string `json:"request_id,omitempty"`
	CorrelationID              string `json:"correlation_id,omitempty"`
	DirectorExecutionMode      string `json:"director_execution_mode,omitempty"`
	RunRef                     string `json:"run_ref,omitempty"`
	OperationalDirectorPlanRef string `json:"operational_director_plan_ref,omitempty"`
	QueueRef                   string `json:"queue_ref,omitempty"`
	ContinueMessage            string `json:"continue_message,omitempty"`
	OccurredAt                 string `json:"occurred_at,omitempty"`
	IdempotencyKey             string `json:"idempotency_key,omitempty"`

	MaxTicks             int  `json:"max_ticks,omitempty"`
	MaxRunsPerTick       int  `json:"max_runs_per_tick,omitempty"`
	MaxExecutions        int  `json:"max_executions,omitempty"`
	AllowRepeatedRuns    bool `json:"allow_repeated_runs,omitempty"`
	MaxBursts            int  `json:"max_bursts,omitempty"`
	MaxStepsPerBurst     int  `json:"max_steps_per_burst,omitempty"`
	MaxDispatchesPerWait int  `json:"max_dispatches_per_wait,omitempty"`
	MaxCommands          int  `json:"max_commands,omitempty"`
	MaxOutboxPerCycle    int  `json:"max_outbox_per_cycle,omitempty"`
	MaxDecisionCycles    int  `json:"max_decision_cycles,omitempty"`
	MaxExternalWaits     int  `json:"max_external_waits,omitempty"`
	ResidentMode         bool `json:"resident_mode,omitempty"`
}

type MCPRunSupervisorSnapshotV0 struct {
	Status       string   `json:"status,omitempty"`
	SessionRef   string   `json:"session_ref,omitempty"`
	AgentRef     string   `json:"agent_ref,omitempty"`
	ProcessRef   string   `json:"process_ref,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type MCPRunSupervisorTickV0 struct {
	TickNumber int                        `json:"tick_number"`
	Action     string                     `json:"action"`
	Snapshot   MCPRunSupervisorSnapshotV0 `json:"snapshot"`
}

type MCPRunSupervisorToolResultV0 struct {
	Estado         string                           `json:"estado"`
	RequestID      string                           `json:"request_id,omitempty"`
	CorrelationID  string                           `json:"correlation_id,omitempty"`
	RunRef         string                           `json:"run_ref,omitempty"`
	StopReason     string                           `json:"stop_reason,omitempty"`
	Ticks          int                              `json:"ticks,omitempty"`
	Last           MCPRunSupervisorSnapshotV0       `json:"last,omitempty"`
	History        []MCPRunSupervisorTickV0         `json:"history,omitempty"`
	EvidenceRefs   []string                         `json:"evidence_refs,omitempty"`
	IdempotencyKey string                           `json:"idempotency_key,omitempty"`
	RepairRunRefs  []string                         `json:"repair_run_refs,omitempty"`
	NextActions    []string                         `json:"next_actions,omitempty"`
	OperationRef   string                           `json:"operation_ref,omitempty"`
	Diagnostics    []MCPAutoprogrammingDiagnosticV0 `json:"diagnostics,omitempty"`
	Errores        []MCPValidationIssueV0           `json:"errores_publicos,omitempty"`
}

func MCPRunSupervisorDescriptorV0() MCPRunSupervisorToolDescriptorV0 {
	return MCPRunSupervisorToolDescriptorV0{
		Name:        MCPRunSupervisorToolNameV0,
		Version:     MCPRunSupervisorToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,director_execution_mode?:goal_first|legacy_director_loop,run_ref?,operational_director_plan_ref?,queue_ref?,continue_message?,occurred_at?,idempotency_key?,max_ticks?,max_runs_per_tick?,max_executions?,allow_repeated_runs?,max_bursts?,max_steps_per_burst?,max_dispatches_per_wait?,max_commands?,max_outbox_per_cycle?,max_decision_cycles?,max_external_waits?,resident_mode?}",
		Output:      "ok:{run_ref,stop_reason,ticks,last,history?,evidence_refs?,idempotency_key?,diagnostics?,next_actions?}|error:{errores_publicos,evidence_refs?,idempotency_key?,diagnostics?,next_actions?}",
		ResourceURI: MCPRunSupervisorResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"run_ref limita la accion a una run",
			"sin run_ref avanza solo cola legacy/resident inyectada si la composicion lo habilita y el payload declara director_execution_mode=legacy_director_loop",
			"run_ref legacy exige opt-in de composicion y director_execution_mode=legacy_director_loop",
			"runs goal-first deben observarse por observe_goal y no por supervise",
			"no usa stdin ni canal paralelo",
			"los lanzamientos salen por outbox y dispatcher existentes",
		},
	}
}

func NewMCPRunSupervisorOKResultV0(
	input MCPRunSupervisorToolInputV0,
	runRef string,
	stopReason string,
	ticks int,
	last MCPRunSupervisorSnapshotV0,
	history []MCPRunSupervisorTickV0,
) MCPRunSupervisorToolResultV0 {
	return MCPRunSupervisorToolResultV0{
		Estado:         MCPRunSupervisorEstadoOKV0,
		RequestID:      strings.TrimSpace(input.RequestID),
		CorrelationID:  firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:         firstNonEmptyMCPV0(runRef, input.RunRef, last.SessionRef),
		StopReason:     strings.TrimSpace(stopReason),
		Ticks:          ticks,
		Last:           last,
		History:        append([]MCPRunSupervisorTickV0(nil), history...),
		EvidenceRefs:   compactStringsMCPV0(last.EvidenceRefs),
		IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
		Diagnostics:    []MCPAutoprogrammingDiagnosticV0{},
		Errores:        []MCPValidationIssueV0{},
	}
}

func NewMCPRunSupervisorErrorResultV0(
	input MCPRunSupervisorToolInputV0,
	code string,
	field string,
	message string,
) MCPRunSupervisorToolResultV0 {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "run_supervisor_error"
	}
	return MCPRunSupervisorToolResultV0{
		Estado:         MCPRunSupervisorEstadoErrorV0,
		RequestID:      strings.TrimSpace(input.RequestID),
		CorrelationID:  firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:         strings.TrimSpace(input.RunRef),
		IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
		Diagnostics:    []MCPAutoprogrammingDiagnosticV0{},
		Errores: []MCPValidationIssueV0{{
			Code:    code,
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(firstNonEmptyMCPV0(message, code)),
		}},
	}
}

func NewMCPRunSupervisorExecutorErrorResultV0(
	input MCPRunSupervisorToolInputV0,
	field string,
	message string,
) MCPRunSupervisorToolResultV0 {
	result := NewMCPRunSupervisorErrorResultV0(
		input,
		"run_supervisor_execute_error",
		field,
		message,
	)
	result.OperationRef = mcpRunSupervisorOperationRefV0(input)
	result.EvidenceRefs = compactStringsMCPV0([]string{
		"evidence-ref-run-supervisor-execute-error",
		result.OperationRef,
	})
	result.Diagnostics = []MCPAutoprogrammingDiagnosticV0{{
		Code:         "run_supervisor_execute_error",
		Scope:        mcpRunSupervisorOperationScopeV0(input),
		Message:      "fallo publico al supervisar run",
		EvidenceRefs: result.EvidenceRefs,
	}}
	return result
}

func mcpRunSupervisorClassifiedOKResultDespiteExecutorErrorV0(
	result MCPRunSupervisorToolResultV0,
) bool {
	return result.Estado == MCPRunSupervisorEstadoOKV0 &&
		(strings.TrimSpace(result.StopReason) != "" ||
			len(result.Diagnostics) > 0 ||
			len(result.EvidenceRefs) > 0 ||
			len(result.NextActions) > 0)
}
