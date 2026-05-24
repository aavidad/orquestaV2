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
	RequestID            string   `json:"request_id,omitempty"`
	CorrelationID        string   `json:"correlation_id,omitempty"`
	RunRef               string   `json:"run_ref,omitempty"`
	AppRef               string   `json:"app_ref,omitempty"`
	ExternalJobRef       string   `json:"external_job_ref,omitempty"`
	QueueRef             string   `json:"queue_ref,omitempty"`
	AppRefs              []string `json:"app_refs,omitempty"`
	QueueLimit           int      `json:"queue_limit,omitempty"`
	OccurredAt           string   `json:"occurred_at,omitempty"`
	IncludeProcessRefs   bool     `json:"include_process_refs,omitempty"`
	IncludeAgentProgress bool     `json:"include_agent_progress,omitempty"`
	IncludeAgentUsage    bool     `json:"include_agent_usage,omitempty"`
}

type MCPAutoprogrammingStatusToolResultV0 struct {
	Estado        string                           `json:"estado"`
	RequestID     string                           `json:"request_id,omitempty"`
	CorrelationID string                           `json:"correlation_id,omitempty"`
	RunRef        string                           `json:"run_ref,omitempty"`
	QueueRef      string                           `json:"queue_ref,omitempty"`
	Queue         *MCPRunQueuePriorityToolResultV0 `json:"queue,omitempty"`
	Run           *MCPDirectorStatsToolResultV0    `json:"run,omitempty"`
	Operator      *MCPAutoprogrammingOperatorV0    `json:"operator,omitempty"`
	Diagnostics   []MCPAutoprogrammingDiagnosticV0 `json:"diagnostics,omitempty"`
	Errores       []MCPValidationIssueV0           `json:"errores_publicos,omitempty"`
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
		Output:      "ok:{queue?,run?,operator?,diagnostics?,operator_advice?}|error:{errores_publicos,diagnostics?,operator_advice?}",
		ResourceURI: MCPAutoprogrammingStatusResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"estado de cola via run_queue.priority inyectado",
			"estado de run via director.stats inyectado",
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
		IncludeProcessRefs:   input.IncludeProcessRefs,
		IncludeAgentProgress: input.IncludeAgentProgress,
		IncludeAgentUsage:    input.IncludeAgentUsage,
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
