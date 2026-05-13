package orquestamcp

import (
	"strings"
	"time"

	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

const (
	MCPServerShutdownToolNameV0    = "orquesta.server.shutdown.v0"
	MCPServerShutdownToolVersionV0 = "v0"
	MCPServerShutdownResourceURIV0 = "orquesta://contracts/server-shutdown/v0"
	MCPServerShutdownEstadoOKV0    = "ok"
	MCPServerShutdownEstadoErrorV0 = "error"
)

type MCPServerShutdownToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPServerShutdownToolInputV0 struct {
	RequestID         string   `json:"request_id,omitempty"`
	CorrelationID     string   `json:"correlation_id,omitempty"`
	QueueRef          string   `json:"queue_ref,omitempty"`
	AppRefs           []string `json:"app_refs,omitempty"`
	QueueLimit        int      `json:"queue_limit,omitempty"`
	MaxTicks          int      `json:"max_ticks,omitempty"`
	MaxRunsPerTick    int      `json:"max_runs_per_tick,omitempty"`
	MaxExecutions     int      `json:"max_executions,omitempty"`
	Forced            bool     `json:"forced,omitempty"`
	RequestedBy       string   `json:"requested_by,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	IdempotencyKey    string   `json:"idempotency_key,omitempty"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
	OccurredAt        string   `json:"occurred_at,omitempty"`
	StopOnNoExecution bool     `json:"stop_on_no_execution,omitempty"`
}

type MCPServerShutdownToolResultV0 struct {
	Estado                  string                   `json:"estado"`
	RequestID               string                   `json:"request_id,omitempty"`
	CorrelationID           string                   `json:"correlation_id,omitempty"`
	Status                  string                   `json:"status,omitempty"`
	ShutdownReady           bool                     `json:"shutdown_ready"`
	RunsRequested           int                      `json:"runs_requested"`
	RunsStopped             int                      `json:"runs_stopped"`
	AgentsInFlight          int                      `json:"agents_in_flight"`
	CheckpointsPending      int                      `json:"checkpoints_pending"`
	CheckpointAgentsPending int                      `json:"checkpoint_agents_pending,omitempty"`
	Runs                    []MCPServerShutdownRunV0 `json:"runs,omitempty"`
	EvidenceRefs            []string                 `json:"evidence_refs,omitempty"`
	Errores                 []MCPValidationIssueV0   `json:"errores_publicos,omitempty"`
}

type MCPServerShutdownRunV0 struct {
	RunRef                     string   `json:"run_ref"`
	AppRef                     string   `json:"app_ref,omitempty"`
	ControlStatus              string   `json:"control_status,omitempty"`
	CheckpointRequired         bool     `json:"checkpoint_required,omitempty"`
	CheckpointRef              string   `json:"checkpoint_ref,omitempty"`
	PendingCheckpointAgentRefs []string `json:"pending_checkpoint_agent_refs,omitempty"`
	CheckpointEvidenceRefs     []string `json:"checkpoint_evidence_refs,omitempty"`
	Terminal                   bool     `json:"terminal,omitempty"`
	StopRequested              bool     `json:"stop_requested,omitempty"`
	AgentsInFlight             int      `json:"agents_in_flight"`
	AgentsStopRequested        int      `json:"agents_stop_requested"`
	AgentsStopConfirmed        int      `json:"agents_stop_confirmed"`
	Ready                      bool     `json:"ready"`
}

func MCPServerShutdownDescriptorV0() MCPServerShutdownToolDescriptorV0 {
	return MCPServerShutdownToolDescriptorV0{
		Name:        MCPServerShutdownToolNameV0,
		Version:     MCPServerShutdownToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,queue_ref?,app_refs?,forced?,max_ticks?,max_runs_per_tick?,max_executions?,requested_by?,reason?,idempotency_key?,evidence_refs?}",
		Output:      "ok:{status,shutdown_ready,runs_requested,runs_stopped,agents_in_flight,checkpoint_agents_pending,runs?}|error:{errores_publicos}",
		ResourceURI: MCPServerShutdownResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"delega en orquesta-server-shutdown",
			"no para procesos ni toca runtime directamente",
			"usa RunControl RunQueue Supervisor y stats por puertos",
		},
	}
}

func newMCPServerShutdownResultV0(
	input MCPServerShutdownToolInputV0,
	result orquestaservershutdown.ServerShutdownResultV0,
) MCPServerShutdownToolResultV0 {
	estado := MCPServerShutdownEstadoOKV0
	if serverShutdownStatusIsErrorV0(result.Status) {
		estado = MCPServerShutdownEstadoErrorV0
	}
	return MCPServerShutdownToolResultV0{
		Estado:                  estado,
		RequestID:               strings.TrimSpace(input.RequestID),
		CorrelationID:           firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Status:                  strings.TrimSpace(result.Status),
		ShutdownReady:           result.ShutdownReady,
		RunsRequested:           result.RunsRequested,
		RunsStopped:             result.RunsStopped,
		AgentsInFlight:          result.AgentsInFlight,
		CheckpointsPending:      result.CheckpointsPending,
		CheckpointAgentsPending: result.CheckpointAgentsPending,
		Runs:                    mcpServerShutdownRunsV0(result.Runs),
		EvidenceRefs:            compactStringsMCPV0(result.EvidenceRefs),
		Errores:                 serverShutdownErrorsV0(result.Status),
	}
}

func newMCPServerShutdownErrorV0(
	input MCPServerShutdownToolInputV0,
	code string,
	field string,
	message string,
) MCPServerShutdownToolResultV0 {
	return MCPServerShutdownToolResultV0{
		Estado:        MCPServerShutdownEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}

func serverShutdownStatusIsErrorV0(status string) bool {
	switch strings.TrimSpace(status) {
	case orquestaservershutdown.ServerShutdownStatusNoQueueReaderV0,
		orquestaservershutdown.ServerShutdownStatusNoRunControlReaderV0,
		orquestaservershutdown.ServerShutdownStatusNoRunControlWriterV0:
		return true
	default:
		return false
	}
}

func serverShutdownErrorsV0(status string) []MCPValidationIssueV0 {
	if !serverShutdownStatusIsErrorV0(status) {
		return []MCPValidationIssueV0{}
	}
	return []MCPValidationIssueV0{{
		Code:    strings.TrimSpace(status),
		Field:   "deps",
		Message: strings.TrimSpace(status),
	}}
}

func mcpServerShutdownRunsV0(
	runs []orquestaservershutdown.ServerShutdownRunResultV0,
) []MCPServerShutdownRunV0 {
	out := make([]MCPServerShutdownRunV0, 0, len(runs))
	for _, run := range runs {
		out = append(out, MCPServerShutdownRunV0{
			RunRef:                     strings.TrimSpace(run.RunRef),
			AppRef:                     strings.TrimSpace(run.AppRef),
			ControlStatus:              strings.TrimSpace(run.ControlStatus),
			CheckpointRequired:         run.CheckpointRequired,
			CheckpointRef:              strings.TrimSpace(run.CheckpointRef),
			PendingCheckpointAgentRefs: compactStringsMCPV0(run.PendingCheckpointAgentRefs),
			CheckpointEvidenceRefs:     compactStringsMCPV0(run.CheckpointEvidenceRefs),
			Terminal:                   run.Terminal,
			StopRequested:              run.StopRequested,
			AgentsInFlight:             run.AgentsInFlight,
			AgentsStopRequested:        run.AgentsStopRequested,
			AgentsStopConfirmed:        run.AgentsStopConfirmed,
			Ready:                      run.Ready,
		})
	}
	if out == nil {
		return []MCPServerShutdownRunV0{}
	}
	return out
}

func serverShutdownCommandFromMCPV0(
	input MCPServerShutdownToolInputV0,
) orquestaservershutdown.ServerShutdownCommandV0 {
	return orquestaservershutdown.ServerShutdownCommandV0{
		RequestID:         input.RequestID,
		CorrelationID:     input.CorrelationID,
		QueueRef:          input.QueueRef,
		AppRefs:           input.AppRefs,
		QueueLimit:        input.QueueLimit,
		MaxTicks:          input.MaxTicks,
		MaxRunsPerTick:    input.MaxRunsPerTick,
		MaxExecutions:     input.MaxExecutions,
		Forced:            input.Forced,
		RequestedBy:       input.RequestedBy,
		Reason:            input.Reason,
		IdempotencyKey:    input.IdempotencyKey,
		EvidenceRefs:      input.EvidenceRefs,
		OccurredAt:        parseMCPServerShutdownOccurredAtV0(input.OccurredAt),
		StopOnNoExecution: input.StopOnNoExecution,
	}
}

func parseMCPServerShutdownOccurredAtV0(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed
}
