package orquestamcp

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const (
	MCPQueueGlobalStatusHTTPPathV0          = "/api/v0/queue/global-status"
	MCPQueueGlobalStatusSchemaVersionV0     = "queue_global_status.v0"
	defaultMCPQueueGlobalStatusQueueRefV0   = "global"
	defaultMCPQueueGlobalStatusQueueLimitV0 = 200
)

type MCPQueueGlobalStatusResultV0 struct {
	SchemaVersion string                              `json:"schema_version"`
	Estado        string                              `json:"estado"`
	RequestID     string                              `json:"request_id,omitempty"`
	CorrelationID string                              `json:"correlation_id,omitempty"`
	QueueRef      string                              `json:"queue_ref,omitempty"`
	Summary       MCPQueueGlobalStatusSummaryV0       `json:"summary"`
	QueueHealth   *MCPAutoprogrammingQueueHealthV0    `json:"queue_health,omitempty"`
	ActiveRuns    []MCPAutoprogrammingActiveRunV0     `json:"active_runs,omitempty"`
	GoalRunRefs   []string                            `json:"goal_run_refs,omitempty"`
	StaleRunning  []MCPAutoprogrammingActionableRunV0 `json:"stale_running,omitempty"`
	SafeActions   []MCPAutoprogrammingSafeActionV0    `json:"safe_actions,omitempty"`
	Diagnostics   []MCPAutoprogrammingDiagnosticV0    `json:"diagnostics,omitempty"`
	Errores       []MCPValidationIssueV0              `json:"errores_publicos,omitempty"`
}

type MCPQueueGlobalStatusSummaryV0 struct {
	Queued                    int  `json:"queued,omitempty"`
	QueuedNotDispatched       int  `json:"queued_not_dispatched,omitempty"`
	RunningLive               int  `json:"running_live,omitempty"`
	AgentsLive                int  `json:"agents_live,omitempty"`
	RunningStale              int  `json:"running_stale,omitempty"`
	RunningStaleNoProcess     int  `json:"running_stale_no_process,omitempty"`
	RunningWithoutRecentStats int  `json:"running_without_recent_stats,omitempty"`
	Blocked                   int  `json:"blocked,omitempty"`
	Lost                      int  `json:"lost,omitempty"`
	Completed                 int  `json:"completed,omitempty"`
	Failed                    int  `json:"failed,omitempty"`
	ActiveRuns                int  `json:"active_runs,omitempty"`
	GoalRuns                  int  `json:"goal_runs,omitempty"`
	SafeActions               int  `json:"safe_actions,omitempty"`
	Diagnostics               int  `json:"diagnostics,omitempty"`
	NeedsAttention            bool `json:"needs_attention,omitempty"`
}

func NewMCPQueueGlobalStatusHTTPHandlerV0(
	executor MCPTransportAutoprogrammingStatusExecutorV0,
) http.Handler {
	return mcpQueueGlobalStatusHTTPHandlerV0{executor: executor}
}

type mcpQueueGlobalStatusHTTPHandlerV0 struct {
	executor MCPTransportAutoprogrammingStatusExecutorV0
}

func (handler mcpQueueGlobalStatusHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPQueueGlobalStatusHTTPPathV0 {
		writeMCPQueueGlobalStatusHTTPV0(w, http.StatusNotFound, newMCPQueueGlobalStatusHTTPErrorV0(
			MCPAutoprogrammingStatusToolInputV0{},
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID")),
			"path",
			MCPPublicErrPathUnsupportedV0,
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodGet, http.MethodPost) {
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		setMCPPublicHTTPAllowV0(w, http.MethodGet, http.MethodPost)
		writeMCPQueueGlobalStatusHTTPV0(w, http.StatusMethodNotAllowed, newMCPQueueGlobalStatusHTTPErrorV0(
			MCPAutoprogrammingStatusToolInputV0{},
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID")),
			"method",
			MCPPublicErrMethodNotAllowedV0,
		))
		return
	}
	input, decodeCode := mcpQueueGlobalStatusHTTPInputV0(w, r)
	if decodeCode != "" {
		writeMCPQueueGlobalStatusHTTPV0(w, http.StatusBadRequest, newMCPQueueGlobalStatusHTTPErrorV0(
			input,
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
			"body",
			decodeCode,
		))
		return
	}
	if handler.executor == nil {
		writeMCPQueueGlobalStatusHTTPV0(w, http.StatusServiceUnavailable, newMCPQueueGlobalStatusHTTPErrorV0(
			input,
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
			"executor",
			"queue_global_status_no_configurado",
		))
		return
	}
	status, err := handler.executor.Execute(r.Context(), input)
	if err != nil {
		writeMCPQueueGlobalStatusHTTPV0(w, http.StatusInternalServerError, newMCPQueueGlobalStatusHTTPErrorV0(
			input,
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("queue_global_status_executor_error", err),
		))
		return
	}
	result := newMCPQueueGlobalStatusResultV0(input, status)
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID)
	httpStatus := http.StatusOK
	if result.Estado == MCPAutoprogrammingStatusEstadoErrorV0 {
		httpStatus = http.StatusBadRequest
	}
	writeMCPQueueGlobalStatusHTTPV0(w, httpStatus, result)
}

func mcpQueueGlobalStatusHTTPInputV0(
	w http.ResponseWriter,
	r *http.Request,
) (MCPAutoprogrammingStatusToolInputV0, string) {
	if r.Method == http.MethodGet {
		return mcpQueueGlobalStatusInputFromQueryV0(r), ""
	}
	var input MCPAutoprogrammingStatusToolInputV0
	if code := decodeMCPPublicHTTPJSONProfileV0(w, r, &input, mcpPublicHTTPJSONProfileAutoprogrammingV0); code != "" {
		return input, code
	}
	return normalizeMCPQueueGlobalStatusInputV0(input), ""
}

func mcpQueueGlobalStatusInputFromQueryV0(r *http.Request) MCPAutoprogrammingStatusToolInputV0 {
	query := r.URL.Query()
	input := MCPAutoprogrammingStatusToolInputV0{
		RequestID:     strings.TrimSpace(query.Get("request_id")),
		CorrelationID: strings.TrimSpace(query.Get("correlation_id")),
		RunRef:        strings.TrimSpace(query.Get("run_ref")),
		AppRef:        strings.TrimSpace(query.Get("app_ref")),
		QueueRef:      strings.TrimSpace(query.Get("queue_ref")),
		OccurredAt:    strings.TrimSpace(query.Get("occurred_at")),
	}
	if limit, err := strconv.Atoi(strings.TrimSpace(query.Get("queue_limit"))); err == nil {
		input.QueueLimit = limit
	}
	return normalizeMCPQueueGlobalStatusInputV0(input)
}

func normalizeMCPQueueGlobalStatusInputV0(
	input MCPAutoprogrammingStatusToolInputV0,
) MCPAutoprogrammingStatusToolInputV0 {
	input.QueueRef = firstNonEmptyMCPV0(input.QueueRef, defaultMCPQueueGlobalStatusQueueRefV0)
	if input.QueueLimit <= 0 {
		input.QueueLimit = defaultMCPQueueGlobalStatusQueueLimitV0
	}
	input.IncludeAgentProgress = mcpFlexibleBoolV0(true)
	input.IncludeAgentUsage = mcpFlexibleBoolV0(true)
	return input
}

func newMCPQueueGlobalStatusResultV0(
	input MCPAutoprogrammingStatusToolInputV0,
	status MCPAutoprogrammingStatusToolResultV0,
) MCPQueueGlobalStatusResultV0 {
	operator := status.Operator
	var activeRuns []MCPAutoprogrammingActiveRunV0
	var safeActions []MCPAutoprogrammingSafeActionV0
	if operator != nil {
		activeRuns = append([]MCPAutoprogrammingActiveRunV0(nil), operator.ActiveRuns...)
		safeActions = append([]MCPAutoprogrammingSafeActionV0(nil), operator.SafeActions...)
	}
	goalRunRefs := mcpQueueGlobalStatusGoalRunRefsV0(safeActions)
	diagnostics := append([]MCPAutoprogrammingDiagnosticV0(nil), status.Diagnostics...)
	summary := mcpQueueGlobalStatusSummaryV0(status.QueueHealth, activeRuns, goalRunRefs, safeActions, diagnostics, status.StaleRunning)
	return MCPQueueGlobalStatusResultV0{
		SchemaVersion: MCPQueueGlobalStatusSchemaVersionV0,
		Estado:        firstNonEmptyMCPV0(status.Estado, MCPAutoprogrammingStatusEstadoOKV0),
		RequestID:     firstNonEmptyMCPV0(status.RequestID, input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(status.CorrelationID, input.CorrelationID, input.RequestID),
		QueueRef:      firstNonEmptyMCPV0(status.QueueRef, input.QueueRef),
		Summary:       summary,
		QueueHealth:   status.QueueHealth,
		ActiveRuns:    activeRuns,
		GoalRunRefs:   goalRunRefs,
		StaleRunning:  append([]MCPAutoprogrammingActionableRunV0(nil), status.StaleRunning...),
		SafeActions:   safeActions,
		Diagnostics:   diagnostics,
		Errores:       append([]MCPValidationIssueV0(nil), status.Errores...),
	}
}

func mcpQueueGlobalStatusSummaryV0(
	health *MCPAutoprogrammingQueueHealthV0,
	activeRuns []MCPAutoprogrammingActiveRunV0,
	goalRunRefs []string,
	safeActions []MCPAutoprogrammingSafeActionV0,
	diagnostics []MCPAutoprogrammingDiagnosticV0,
	staleRunning []MCPAutoprogrammingActionableRunV0,
) MCPQueueGlobalStatusSummaryV0 {
	summary := MCPQueueGlobalStatusSummaryV0{
		ActiveRuns:  len(activeRuns),
		GoalRuns:    len(goalRunRefs),
		SafeActions: len(safeActions),
		Diagnostics: len(diagnostics),
	}
	if health != nil {
		summary.Queued = health.Queued
		summary.QueuedNotDispatched = health.QueuedNotDispatched
		summary.RunningLive = health.RunningLive
		summary.AgentsLive = health.AgentsLive
		summary.RunningStale = health.RunningStale
		summary.RunningStaleNoProcess = health.RunningStaleNoProcess
		summary.RunningWithoutRecentStats = health.RunningWithoutRecentStats
		summary.Blocked = health.Blocked
		summary.Lost = health.Lost
		summary.Completed = health.Completed
		summary.Failed = health.Failed
	}
	summary.NeedsAttention = summary.RunningStale > 0 ||
		summary.RunningStaleNoProcess > 0 ||
		summary.Blocked > 0 ||
		summary.Lost > 0 ||
		summary.Failed > 0 ||
		len(staleRunning) > 0
	return summary
}

func mcpQueueGlobalStatusGoalRunRefsV0(
	actions []MCPAutoprogrammingSafeActionV0,
) []string {
	var refs []string
	for _, action := range actions {
		if strings.TrimSpace(action.Action) != "observe_goal" {
			continue
		}
		refs = append(refs, strings.TrimSpace(action.RunRef))
	}
	return compactStringsMCPV0(refs)
}

func newMCPQueueGlobalStatusHTTPErrorV0(
	input MCPAutoprogrammingStatusToolInputV0,
	correlationID string,
	field string,
	message string,
) MCPQueueGlobalStatusResultV0 {
	return MCPQueueGlobalStatusResultV0{
		SchemaVersion: MCPQueueGlobalStatusSchemaVersionV0,
		Estado:        MCPAutoprogrammingStatusEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: strings.TrimSpace(firstNonEmptyMCPV0(correlationID, input.CorrelationID, input.RequestID)),
		QueueRef:      firstNonEmptyMCPV0(input.QueueRef, defaultMCPQueueGlobalStatusQueueRefV0),
		Summary:       MCPQueueGlobalStatusSummaryV0{},
		Errores: []MCPValidationIssueV0{{
			Code:    "queue_global_status_http_error",
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(firstNonEmptyMCPV0(message, "queue_global_status_http_error")),
		}},
	}
}

func writeMCPQueueGlobalStatusHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPQueueGlobalStatusResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
