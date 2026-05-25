package orquestamcp

import (
	"context"
	"strings"
)

const (
	MCPAutoprogrammingStatusToolNameV0    = "orquesta.autoprogramming.status.v0"
	MCPAutoprogrammingStatusToolVersionV0 = "v0"
	MCPAutoprogrammingStatusResourceURIV0 = "orquesta://contracts/autoprogramming-status/v0"
	MCPAutoprogrammingStatusEstadoOKV0    = "ok"
	MCPAutoprogrammingStatusEstadoErrorV0 = "error"
)

type MCPAutoprogrammingStatusToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAutoprogrammingStatusToolInputV0 struct {
	RequestID            string            `json:"request_id,omitempty"`
	CorrelationID        string            `json:"correlation_id,omitempty"`
	RunRef               string            `json:"run_ref,omitempty"`
	AppRef               string            `json:"app_ref,omitempty"`
	ExternalJobRef       string            `json:"external_job_ref,omitempty"`
	QueueRef             string            `json:"queue_ref,omitempty"`
	AppRefs              []string          `json:"app_refs,omitempty"`
	QueueLimit           int               `json:"queue_limit,omitempty"`
	OccurredAt           string            `json:"occurred_at,omitempty"`
	IncludeProcessRefs   mcpFlexibleBoolV0 `json:"include_process_refs,omitempty"`
	IncludeAgentProgress mcpFlexibleBoolV0 `json:"include_agent_progress,omitempty"`
	IncludeAgentUsage    mcpFlexibleBoolV0 `json:"include_agent_usage,omitempty"`
}

type MCPAutoprogrammingStatusToolResultV0 struct {
	Estado        string                           `json:"estado"`
	RequestID     string                           `json:"request_id,omitempty"`
	CorrelationID string                           `json:"correlation_id,omitempty"`
	RunRef        string                           `json:"run_ref,omitempty"`
	QueueRef      string                           `json:"queue_ref,omitempty"`
	Queue         *MCPRunQueuePriorityToolResultV0 `json:"queue,omitempty"`
	Run           *MCPDirectorStatsToolResultV0    `json:"run,omitempty"`
	Projects      []MCPAutoprogrammingProjectV0    `json:"projects,omitempty"`
	Tasks         []MCPAutoprogrammingTaskV0       `json:"tasks,omitempty"`
	Agents        []MCPAutoprogrammingAgentV0      `json:"agents,omitempty"`
	Operator      *MCPAutoprogrammingOperatorV0    `json:"operator,omitempty"`
	Diagnostics   []MCPAutoprogrammingDiagnosticV0 `json:"diagnostics,omitempty"`
	Errores       []MCPValidationIssueV0           `json:"errores_publicos,omitempty"`
}

type MCPAutoprogrammingProjectV0 struct {
	ProjectRef     string   `json:"project_ref,omitempty"`
	AppRef         string   `json:"app_ref,omitempty"`
	RunRefs        []string `json:"run_refs,omitempty"`
	QueueCount     int      `json:"queue_count,omitempty"`
	TasksTotal     int      `json:"tasks_total,omitempty"`
	TasksClosed    int      `json:"tasks_closed,omitempty"`
	AgentsInFlight int      `json:"agents_in_flight,omitempty"`
	Blocked        bool     `json:"blocked,omitempty"`
}

type MCPAutoprogrammingTaskV0 struct {
	RunRef              string   `json:"run_ref,omitempty"`
	TaskRef             string   `json:"task_ref"`
	Status              string   `json:"status,omitempty"`
	AgentRequestID      string   `json:"agent_request_id,omitempty"`
	DeliveryRef         string   `json:"delivery_ref,omitempty"`
	LastReportRef       string   `json:"last_report_ref,omitempty"`
	ProgressStatus      string   `json:"progress_status,omitempty"`
	Classification      string   `json:"classification,omitempty"`
	BudgetReason        string   `json:"budget_reason,omitempty"`
	AgeSeconds          int64    `json:"age_seconds,omitempty"`
	SecondsSinceAck     int64    `json:"seconds_since_ack,omitempty"`
	MaxExpectedSeconds  int64    `json:"max_expected_seconds,omitempty"`
	NoActivityLimit     int64    `json:"no_activity_limit_seconds,omitempty"`
	NoProgressTicks     int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int      `json:"repeated_action_count,omitempty"`
	DecisionRequired    bool     `json:"decision_required,omitempty"`
	Summary             string   `json:"summary,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type MCPAutoprogrammingAgentV0 struct {
	RunRef              string   `json:"run_ref,omitempty"`
	AgentRequestID      string   `json:"agent_request_id"`
	Status              string   `json:"status,omitempty"`
	InFlight            bool     `json:"in_flight,omitempty"`
	NeedsAttention      bool     `json:"needs_attention,omitempty"`
	Completed           bool     `json:"completed,omitempty"`
	Failed              bool     `json:"failed,omitempty"`
	Lost                bool     `json:"lost,omitempty"`
	StopRequested       bool     `json:"stop_requested,omitempty"`
	StopConfirmed       bool     `json:"stop_confirmed,omitempty"`
	StopReasonCode      string   `json:"stop_reason_code,omitempty"`
	StopReasonSource    string   `json:"stop_reason_source,omitempty"`
	StopReasonRef       string   `json:"stop_reason_ref,omitempty"`
	ControlRegistered   bool     `json:"control_registered,omitempty"`
	ControlState        string   `json:"control_state,omitempty"`
	CanStop             bool     `json:"can_stop,omitempty"`
	TaskRef             string   `json:"task_ref,omitempty"`
	DeliveryRef         string   `json:"delivery_ref,omitempty"`
	ReportRef           string   `json:"report_ref,omitempty"`
	ProgressStatus      string   `json:"progress_status,omitempty"`
	Classification      string   `json:"classification,omitempty"`
	BudgetReason        string   `json:"budget_reason,omitempty"`
	AgeSeconds          int64    `json:"age_seconds,omitempty"`
	SecondsSinceAck     int64    `json:"seconds_since_ack,omitempty"`
	MaxExpectedSeconds  int64    `json:"max_expected_seconds,omitempty"`
	NoActivityLimit     int64    `json:"no_activity_limit_seconds,omitempty"`
	NoProgressTicks     int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int      `json:"repeated_action_count,omitempty"`
	DecisionRequired    bool     `json:"decision_required,omitempty"`
	ProcessRef          string   `json:"process_ref,omitempty"`
	SessionRef          string   `json:"session_ref,omitempty"`
	LaunchRef           string   `json:"launch_ref,omitempty"`
	RuntimeKind         string   `json:"runtime_kind,omitempty"`
	ConnectorRef        string   `json:"connector_ref,omitempty"`
	ProfileRef          string   `json:"profile_ref,omitempty"`
	CapacityLevel       string   `json:"capacity_level,omitempty"`
	QuotaStatus         string   `json:"quota_status,omitempty"`
	TotalTokens         int64    `json:"total_tokens,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type MCPAutoprogrammingDiagnosticV0 struct {
	Code         string   `json:"code"`
	Scope        string   `json:"scope,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type MCPAutoprogrammingStatusToolExecutorV0 struct {
	Queue MCPTransportRunQueuePriorityExecutorV0
	Stats MCPTransportDirectorStatsExecutorV0
}

func MCPAutoprogrammingStatusDescriptorV0() MCPAutoprogrammingStatusToolDescriptorV0 {
	return MCPAutoprogrammingStatusToolDescriptorV0{
		Name:        MCPAutoprogrammingStatusToolNameV0,
		Version:     MCPAutoprogrammingStatusToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref?,external_job_ref?,queue_ref?,app_refs?,queue_limit?,telemetry_flags?,operator_advice?}",
		Output:      "ok:{queue?,run?,projects?,tasks?,agents?,operator?,diagnostics?,operator_advice?}|error:{errores_publicos,diagnostics?,operator_advice?}",
		ResourceURI: MCPAutoprogrammingStatusResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"estado de cola via run_queue.priority inyectado",
			"estado de run via director.stats inyectado",
			"proyecta proyectos tareas y agentes compactos para filtros externos",
			"diagnostico solo resume puertos y errores publicos",
			"operator_advice se conserva como observacion no bloqueante",
			"sin DB runtime filesystem Codex ni proveedor concreto",
		},
	}
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
) (MCPAutoprogrammingStatusToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := newMCPAutoprogrammingStatusBaseV0(input)
	okCount := 0
	if executor.Queue != nil {
		queue, err := executor.Queue.Execute(ctx, mcpAutoprogrammingQueueInputV0(input))
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0("queue_error", "queue", "cola no disponible"))
		} else {
			result.Queue = &queue
			result.QueueRef = queue.QueueRef
			if queue.Estado == MCPRunQueuePriorityEstadoOKV0 {
				okCount++
			}
			result.Diagnostics = append(result.Diagnostics, diagnosticsFromIssuesMCPAutoprogrammingV0("queue", queue.Errores)...)
		}
	} else {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0("queue_unbound", "queue", "run_queue no configurado"))
	}
	if wantsMCPAutoprogrammingRunStatusV0(input) {
		ok, run, diagnostics, err := executor.executeRunStatusV0(ctx, input)
		if err != nil {
			return result, err
		}
		result.Run = run
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if ok {
			okCount++
			result.RunRef = run.RunRef
		}
	}
	if okCount == 0 {
		result.Estado = MCPAutoprogrammingStatusEstadoErrorV0
		result.Errores = []MCPValidationIssueV0{{
			Code:    "autoprogramming_status_no_disponible",
			Field:   "ports",
			Message: "estado de autoprogramacion no disponible",
		}}
	}
	if mcpAutoprogrammingReplanAmplificationBlockedV0(result.Run) {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
			"supervisor_replan_amplification_blocked",
			"run",
			"supervision no recomendada: replan/stop repetidos sin entregas, reviews ni cierres",
		))
	}
	if mcpAutoprogrammingNeedsRunStatsForSafeSupervisionV0(result.Queue, result.Run) {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
			"run_stats_required_for_safe_supervision",
			"queue",
			"supervision de cola no declarada segura sin consultar stats del run candidato",
		))
	}
	if mcpAutoprogrammingQueueEmptyOrNotVisibleV0(result.Queue, result.Run) {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
			"queue_empty_or_not_visible",
			"queue",
			"cola sin candidatos visibles; no declarar supervision de cola como accion segura",
		))
	}
	result.Projects = buildMCPAutoprogrammingProjectsV0(result.Queue, result.Run)
	result.Tasks = buildMCPAutoprogrammingTasksV0(result.Run)
	result.Agents = buildMCPAutoprogrammingAgentsV0(result.Run)
	result.Operator = newMCPAutoprogrammingOperatorV0(result.Queue, result.Run, result.Diagnostics)
	return result, nil
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) executeRunStatusV0(
	ctx context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
) (bool, *MCPDirectorStatsToolResultV0, []MCPAutoprogrammingDiagnosticV0, error) {
	if executor.Stats == nil {
		return false, nil, []MCPAutoprogrammingDiagnosticV0{
			mcpAutoprogrammingDiagnosticV0("run_stats_unbound", "run", "director_stats no configurado"),
		}, nil
	}
	run, err := executor.Stats.Execute(ctx, mcpAutoprogrammingStatsInputV0(input))
	if err != nil {
		return false, nil, []MCPAutoprogrammingDiagnosticV0{
			mcpAutoprogrammingDiagnosticV0("run_stats_error", "run", "estado de run no disponible"),
		}, nil
	}
	diagnostics := diagnosticsFromIssuesMCPAutoprogrammingV0("run", run.Errores)
	return run.Estado == MCPDirectorStatsEstadoOKV0, &run, diagnostics, nil
}

func newMCPAutoprogrammingStatusBaseV0(
	input MCPAutoprogrammingStatusToolInputV0,
) MCPAutoprogrammingStatusToolResultV0 {
	return MCPAutoprogrammingStatusToolResultV0{
		Estado:        MCPAutoprogrammingStatusEstadoOKV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:        strings.TrimSpace(input.RunRef),
		QueueRef:      strings.TrimSpace(input.QueueRef),
		Diagnostics:   []MCPAutoprogrammingDiagnosticV0{},
		Errores:       []MCPValidationIssueV0{},
	}
}

func mcpAutoprogrammingQueueInputV0(
	input MCPAutoprogrammingStatusToolInputV0,
) MCPRunQueuePriorityToolInputV0 {
	return MCPRunQueuePriorityToolInputV0{
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Action:        MCPRunQueuePriorityActionRankV0,
		QueueRef:      input.QueueRef,
		AppRefs:       input.AppRefs,
		Limit:         input.QueueLimit,
		OccurredAt:    input.OccurredAt,
	}
}

func mcpAutoprogrammingStatsInputV0(
	input MCPAutoprogrammingStatusToolInputV0,
) MCPDirectorStatsToolInputV0 {
	return MCPDirectorStatsToolInputV0{
		RequestID:            input.RequestID,
		CorrelationID:        input.CorrelationID,
		RunRef:               input.RunRef,
		AppRef:               input.AppRef,
		ExternalJobRef:       input.ExternalJobRef,
		OccurredAt:           input.OccurredAt,
		IncludeProcessRefs:   bool(input.IncludeProcessRefs),
		IncludeAgentProgress: bool(input.IncludeAgentProgress),
		IncludeAgentUsage:    bool(input.IncludeAgentUsage),
	}
}

func wantsMCPAutoprogrammingRunStatusV0(input MCPAutoprogrammingStatusToolInputV0) bool {
	return strings.TrimSpace(input.RunRef) != "" || strings.TrimSpace(input.ExternalJobRef) != ""
}

func diagnosticsFromIssuesMCPAutoprogrammingV0(
	scope string,
	issues []MCPValidationIssueV0,
) []MCPAutoprogrammingDiagnosticV0 {
	out := make([]MCPAutoprogrammingDiagnosticV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, mcpAutoprogrammingDiagnosticV0(issue.Code, scope, issue.Message))
	}
	return out
}

func mcpAutoprogrammingDiagnosticV0(
	code string,
	scope string,
	message string,
) MCPAutoprogrammingDiagnosticV0 {
	return MCPAutoprogrammingDiagnosticV0{
		Code:    strings.TrimSpace(code),
		Scope:   strings.TrimSpace(scope),
		Message: strings.TrimSpace(message),
	}
}
