package orquestamcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MCPDomainWorkStatusHTTPPathV0      = "/api/v0/domain-work/status"
	MCPDomainWorkStatusSchemaVersionV0 = "domain_work_status.v0"
)

const defaultMCPDomainWorkStatusHTTPResponseTimeoutV0 = 2 * time.Second

type MCPDomainWorkStatusResultV0 struct {
	SchemaVersion string                           `json:"schema_version"`
	Estado        string                           `json:"estado"`
	RequestID     string                           `json:"request_id,omitempty"`
	CorrelationID string                           `json:"correlation_id,omitempty"`
	QueueRef      string                           `json:"queue_ref,omitempty"`
	Filters       MCPDomainWorkStatusFiltersV0     `json:"filters,omitempty"`
	Summary       MCPDomainWorkStatusSummaryV0     `json:"summary"`
	Items         []MCPDomainWorkStatusItemV0      `json:"items,omitempty"`
	SafeActions   []MCPAutoprogrammingSafeActionV0 `json:"safe_actions,omitempty"`
	Diagnostics   []MCPAutoprogrammingDiagnosticV0 `json:"diagnostics,omitempty"`
	Errores       []MCPValidationIssueV0           `json:"errores_publicos,omitempty"`
}

type MCPDomainWorkStatusFiltersV0 struct {
	Project        string `json:"project,omitempty"`
	CourseSlug     string `json:"course_slug,omitempty"`
	RunRef         string `json:"run_ref,omitempty"`
	AppRef         string `json:"app_ref,omitempty"`
	ExternalJobRef string `json:"external_job_ref,omitempty"`
}

type MCPDomainWorkStatusSummaryV0 struct {
	Status          string `json:"status"`
	Queued          int    `json:"queued,omitempty"`
	Running         int    `json:"running,omitempty"`
	Blocked         int    `json:"blocked,omitempty"`
	Failed          int    `json:"failed,omitempty"`
	Completed       int    `json:"completed,omitempty"`
	Stale           int    `json:"stale,omitempty"`
	WaitingQuota    int    `json:"waiting_quota,omitempty"`
	ActiveRuns      int    `json:"active_runs,omitempty"`
	NeedsAction     bool   `json:"needs_action,omitempty"`
	WillFinishAlone bool   `json:"will_finish_alone"`
}

type MCPDomainWorkStatusItemV0 struct {
	RunRef            string         `json:"run_ref,omitempty"`
	AppRef            string         `json:"app_ref,omitempty"`
	Status            string         `json:"status"`
	NeedsAction       bool           `json:"needs_action,omitempty"`
	RecommendedAction string         `json:"recommended_action,omitempty"`
	CurrentPhase      string         `json:"current_phase,omitempty"`
	DomainCounters    map[string]int `json:"domain_counters,omitempty"`
	NoActionReason    string         `json:"no_action_reason,omitempty"`
	EvidenceRefs      []string       `json:"evidence_refs,omitempty"`
}

type mcpDomainWorkStatusHTTPInputV0 struct {
	RequestID      string
	CorrelationID  string
	RunRef         string
	AppRef         string
	ExternalJobRef string
	QueueRef       string
	Project        string
	CourseSlug     string
	QueueLimit     int
}

func NewMCPDomainWorkStatusHTTPHandlerV0(
	executor MCPTransportAutoprogrammingStatusExecutorV0,
) http.Handler {
	return newMCPDomainWorkStatusHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPDomainWorkStatusHTTPResponseTimeoutV0,
	)
}

func newMCPDomainWorkStatusHTTPHandlerWithTimeoutV0(
	executor MCPTransportAutoprogrammingStatusExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpDomainWorkStatusHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpDomainWorkStatusHTTPHandlerV0 struct {
	executor        MCPTransportAutoprogrammingStatusExecutorV0
	responseTimeout time.Duration
}

func (handler mcpDomainWorkStatusHTTPHandlerV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != MCPDomainWorkStatusHTTPPathV0 {
		writeMCPDomainWorkStatusHTTPV0(w, http.StatusNotFound, newMCPDomainWorkStatusHTTPErrorV0(
			mcpDomainWorkStatusHTTPInputV0{},
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID")),
			"path",
			MCPPublicErrPathUnsupportedV0,
		))
		return
	}
	if handleMCPPublicHTTPOptionsV0(w, r, http.MethodGet) {
		return
	}
	if r.Method != http.MethodGet {
		setMCPPublicHTTPAllowV0(w, http.MethodGet)
		writeMCPDomainWorkStatusHTTPV0(w, http.StatusMethodNotAllowed, newMCPDomainWorkStatusHTTPErrorV0(
			mcpDomainWorkStatusInputFromQueryV0(r),
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID")),
			"method",
			MCPPublicErrMethodNotAllowedV0,
		))
		return
	}
	input := mcpDomainWorkStatusInputFromQueryV0(r)
	if handler.executor == nil {
		writeMCPDomainWorkStatusHTTPV0(w, http.StatusServiceUnavailable, newMCPDomainWorkStatusHTTPErrorV0(
			input,
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
			"executor",
			"domain_work_status_no_configurado",
		))
		return
	}
	result, err, timedOut := handler.executeDomainWorkStatusWithResponseTimeoutV0(r, input)
	if timedOut {
		writeMCPDomainWorkStatusHTTPV0(w, http.StatusGatewayTimeout, result)
		return
	}
	if err != nil {
		writeMCPDomainWorkStatusHTTPV0(w, http.StatusInternalServerError, newMCPDomainWorkStatusHTTPErrorV0(
			input,
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("domain_work_status_executor_error", err),
		))
		return
	}
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID)
	status := http.StatusOK
	if result.Estado == MCPAutoprogrammingStatusEstadoErrorV0 {
		status = http.StatusBadRequest
	}
	writeMCPDomainWorkStatusHTTPV0(w, status, result)
}

type mcpDomainWorkStatusHTTPExecutionV0 struct {
	result MCPAutoprogrammingStatusToolResultV0
	err    error
}

func (handler mcpDomainWorkStatusHTTPHandlerV0) executeDomainWorkStatusWithResponseTimeoutV0(
	r *http.Request,
	input mcpDomainWorkStatusHTTPInputV0,
) (MCPDomainWorkStatusResultV0, error, bool) {
	statusInput := mcpDomainWorkStatusAutoprogrammingInputV0(input)
	timeout := handler.responseTimeout
	if timeout <= 0 {
		status, err := handler.executor.Execute(r.Context(), statusInput)
		return newMCPDomainWorkStatusResultV0(input, status), err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpDomainWorkStatusHTTPExecutionV0, 1)
	go func() {
		status, err := handler.executor.Execute(ctx, statusInput)
		done <- mcpDomainWorkStatusHTTPExecutionV0{result: status, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return newMCPDomainWorkStatusResultV0(input, execution.result), execution.err, false
	case <-timer.C:
		return newMCPDomainWorkStatusTimeoutResultV0(r, input), nil, true
	}
}

func mcpDomainWorkStatusInputFromQueryV0(r *http.Request) mcpDomainWorkStatusHTTPInputV0 {
	query := r.URL.Query()
	input := mcpDomainWorkStatusHTTPInputV0{
		RequestID:      strings.TrimSpace(query.Get("request_id")),
		CorrelationID:  strings.TrimSpace(query.Get("correlation_id")),
		RunRef:         strings.TrimSpace(query.Get("run_ref")),
		AppRef:         strings.TrimSpace(query.Get("app_ref")),
		ExternalJobRef: strings.TrimSpace(firstNonEmptyMCPV0(query.Get("external_job_ref"), query.Get("job_ref"))),
		QueueRef:       strings.TrimSpace(query.Get("queue_ref")),
		Project:        strings.TrimSpace(query.Get("project")),
		CourseSlug:     strings.TrimSpace(query.Get("course_slug")),
	}
	if limit, err := strconv.Atoi(strings.TrimSpace(query.Get("queue_limit"))); err == nil {
		input.QueueLimit = limit
	}
	return input
}

func mcpDomainWorkStatusAutoprogrammingInputV0(
	input mcpDomainWorkStatusHTTPInputV0,
) MCPAutoprogrammingStatusToolInputV0 {
	queueRef := firstNonEmptyMCPV0(input.QueueRef, defaultMCPQueueGlobalStatusQueueRefV0)
	limit := input.QueueLimit
	if limit <= 0 {
		limit = defaultMCPQueueGlobalStatusQueueLimitV0
	}
	return MCPAutoprogrammingStatusToolInputV0{
		RequestID:            input.RequestID,
		CorrelationID:        input.CorrelationID,
		RunRef:               input.RunRef,
		AppRef:               input.AppRef,
		ExternalJobRef:       input.ExternalJobRef,
		QueueRef:             queueRef,
		QueueLimit:           limit,
		IncludeAgentProgress: mcpFlexibleBoolV0(true),
		IncludeAgentUsage:    mcpFlexibleBoolV0(true),
	}
}

func newMCPDomainWorkStatusResultV0(
	input mcpDomainWorkStatusHTTPInputV0,
	status MCPAutoprogrammingStatusToolResultV0,
) MCPDomainWorkStatusResultV0 {
	queue := newMCPQueueGlobalStatusResultV0(mcpDomainWorkStatusAutoprogrammingInputV0(input), status)
	items := mcpDomainWorkStatusItemsV0(queue.Items)
	diagnostics := append([]MCPAutoprogrammingDiagnosticV0(nil), queue.Diagnostics...)
	diagnostics = append(diagnostics, mcpDomainWorkStatusFilterDiagnosticsV0(input)...)
	summary := mcpDomainWorkStatusSummaryV0(queue.Summary, items, diagnostics)
	return MCPDomainWorkStatusResultV0{
		SchemaVersion: MCPDomainWorkStatusSchemaVersionV0,
		Estado:        firstNonEmptyMCPV0(queue.Estado, MCPAutoprogrammingStatusEstadoOKV0),
		RequestID:     firstNonEmptyMCPV0(queue.RequestID, input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(queue.CorrelationID, input.CorrelationID, input.RequestID),
		QueueRef:      firstNonEmptyMCPV0(queue.QueueRef, input.QueueRef, defaultMCPQueueGlobalStatusQueueRefV0),
		Filters:       mcpDomainWorkStatusFiltersV0(input),
		Summary:       summary,
		Items:         items,
		SafeActions:   append([]MCPAutoprogrammingSafeActionV0(nil), queue.SafeActions...),
		Diagnostics:   diagnostics,
		Errores:       append([]MCPValidationIssueV0(nil), queue.Errores...),
	}
}

func mcpDomainWorkStatusFiltersV0(input mcpDomainWorkStatusHTTPInputV0) MCPDomainWorkStatusFiltersV0 {
	return MCPDomainWorkStatusFiltersV0{
		Project:        strings.TrimSpace(input.Project),
		CourseSlug:     strings.TrimSpace(input.CourseSlug),
		RunRef:         strings.TrimSpace(input.RunRef),
		AppRef:         strings.TrimSpace(input.AppRef),
		ExternalJobRef: strings.TrimSpace(input.ExternalJobRef),
	}
}

func mcpDomainWorkStatusFilterDiagnosticsV0(
	input mcpDomainWorkStatusHTTPInputV0,
) []MCPAutoprogrammingDiagnosticV0 {
	if strings.TrimSpace(input.Project) == "" && strings.TrimSpace(input.CourseSlug) == "" {
		return nil
	}
	if strings.TrimSpace(input.RunRef) != "" ||
		strings.TrimSpace(input.AppRef) != "" ||
		strings.TrimSpace(input.ExternalJobRef) != "" {
		return nil
	}
	return []MCPAutoprogrammingDiagnosticV0{{
		Code:         "domain_work_status_context_filter_without_ref",
		Scope:        "filters",
		Message:      "project/course_slug se conservan como contexto publico; para acotar estado usa tambien run_ref, app_ref o external_job_ref",
		EvidenceRefs: []string{"evidence-ref-domain-work-status-context-filter"},
	}}
}

func mcpDomainWorkStatusItemsV0(
	items []MCPQueueGlobalStatusItemV0,
) []MCPDomainWorkStatusItemV0 {
	out := make([]MCPDomainWorkStatusItemV0, 0, len(items))
	for _, item := range items {
		out = append(out, MCPDomainWorkStatusItemV0{
			RunRef:            strings.TrimSpace(item.RunRef),
			AppRef:            strings.TrimSpace(item.AppRef),
			Status:            mcpDomainWorkStatusNormalizeStatusV0(item.Status, item.RecommendedAction),
			NeedsAction:       item.NeedsAction,
			RecommendedAction: strings.TrimSpace(item.RecommendedAction),
			CurrentPhase:      strings.TrimSpace(item.CurrentPhase),
			DomainCounters:    item.DomainCounters,
			NoActionReason:    strings.TrimSpace(item.NoActionReason),
			EvidenceRefs:      append([]string(nil), item.EvidenceRefs...),
		})
	}
	return out
}

func mcpDomainWorkStatusSummaryV0(
	queue MCPQueueGlobalStatusSummaryV0,
	items []MCPDomainWorkStatusItemV0,
	diagnostics []MCPAutoprogrammingDiagnosticV0,
) MCPDomainWorkStatusSummaryV0 {
	summary := MCPDomainWorkStatusSummaryV0{
		Queued:          queue.Queued + queue.QueuedNotDispatched,
		Running:         queue.RunningLive + queue.RunningWithoutRecentStats,
		Blocked:         queue.Blocked,
		Failed:          queue.Failed,
		Completed:       queue.Completed,
		Stale:           queue.RunningStale + queue.RunningStaleNoProcess,
		ActiveRuns:      queue.ActiveRuns,
		NeedsAction:     queue.NeedsAction,
		WillFinishAlone: queue.WillFinishAlone,
	}
	for _, item := range items {
		switch item.Status {
		case "waiting_quota":
			summary.WaitingQuota++
		case "blocked":
			if summary.Blocked == 0 {
				summary.Blocked = 1
			}
		case "stale":
			if summary.Stale == 0 {
				summary.Stale = 1
			}
		}
	}
	for _, diagnostic := range diagnostics {
		if mcpDomainWorkStatusMentionsQuotaV0(diagnostic.Code) || mcpDomainWorkStatusMentionsQuotaV0(diagnostic.Message) {
			summary.WaitingQuota++
		}
	}
	summary.Status = mcpDomainWorkStatusOverallV0(summary)
	return summary
}

func mcpDomainWorkStatusNormalizeStatusV0(status string, action string) string {
	value := strings.ToLower(strings.TrimSpace(status))
	if mcpDomainWorkStatusMentionsQuotaV0(value) || mcpDomainWorkStatusMentionsQuotaV0(action) {
		return "waiting_quota"
	}
	switch value {
	case mcpAutoprogrammingHealthQueuedV0, "ready", "queued_not_dispatched", "waiting_outbox":
		return "queued"
	case mcpAutoprogrammingHealthRunningLiveV0, mcpAutoprogrammingHealthRunningWithoutRecentStatsV0, "running", "in_progress", "working":
		return "running"
	case mcpAutoprogrammingHealthRunningStaleV0, mcpAutoprogrammingHealthRunningStaleNoProcessV0, "stale", "stale_running":
		return "stale"
	case mcpAutoprogrammingHealthBlockedV0, mcpAutoprogrammingActionGoalFirstBlockedV0, "needs_action", "queue_needs_action", "observer_required", "retry_required", "review_required":
		return "blocked"
	case mcpAutoprogrammingHealthFailedV0, "error":
		return "failed"
	case mcpAutoprogrammingHealthCompletedV0, "accepted", "closed", "delivered":
		return "completed"
	default:
		return firstNonEmptyMCPV0(value, "unknown")
	}
}

func mcpDomainWorkStatusMentionsQuotaV0(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "quota") || strings.Contains(value, "rate_limit")
}

func mcpDomainWorkStatusOverallV0(summary MCPDomainWorkStatusSummaryV0) string {
	switch {
	case summary.WaitingQuota > 0:
		return "waiting_quota"
	case summary.Blocked > 0:
		return "blocked"
	case summary.Failed > 0:
		return "failed"
	case summary.Stale > 0:
		return "stale"
	case summary.Running > 0:
		return "running"
	case summary.Queued > 0:
		return "queued"
	case summary.Completed > 0:
		return "completed"
	default:
		return "idle"
	}
}

func newMCPDomainWorkStatusTimeoutResultV0(
	r *http.Request,
	input mcpDomainWorkStatusHTTPInputV0,
) MCPDomainWorkStatusResultV0 {
	result := MCPDomainWorkStatusResultV0{
		SchemaVersion: MCPDomainWorkStatusSchemaVersionV0,
		Estado:        MCPAutoprogrammingStatusEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
		QueueRef:      firstNonEmptyMCPV0(input.QueueRef, defaultMCPQueueGlobalStatusQueueRefV0),
		Filters:       mcpDomainWorkStatusFiltersV0(input),
		Summary: MCPDomainWorkStatusSummaryV0{
			Status:          "blocked",
			NeedsAction:     true,
			WillFinishAlone: false,
		},
		Diagnostics: []MCPAutoprogrammingDiagnosticV0{{
			Code:         "domain_work_status_timeout",
			Scope:        "executor",
			Message:      "consulta de estado domain-work cancelada por timeout HTTP",
			EvidenceRefs: []string{"evidence-ref-domain-work-status-timeout"},
		}},
		Errores: []MCPValidationIssueV0{{
			Code:    "domain_work_status_timeout",
			Field:   "executor",
			Message: "consulta de estado domain-work excedio la ventana HTTP acotada",
		}},
	}
	return result
}

func newMCPDomainWorkStatusHTTPErrorV0(
	input mcpDomainWorkStatusHTTPInputV0,
	correlationID string,
	field string,
	message string,
) MCPDomainWorkStatusResultV0 {
	return MCPDomainWorkStatusResultV0{
		SchemaVersion: MCPDomainWorkStatusSchemaVersionV0,
		Estado:        MCPAutoprogrammingStatusEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: strings.TrimSpace(firstNonEmptyMCPV0(correlationID, input.CorrelationID, input.RequestID)),
		QueueRef:      firstNonEmptyMCPV0(input.QueueRef, defaultMCPQueueGlobalStatusQueueRefV0),
		Filters:       mcpDomainWorkStatusFiltersV0(input),
		Summary: MCPDomainWorkStatusSummaryV0{
			Status:          "blocked",
			NeedsAction:     true,
			WillFinishAlone: false,
		},
		Errores: []MCPValidationIssueV0{{
			Code:    "domain_work_status_http_error",
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(firstNonEmptyMCPV0(message, "domain_work_status_http_error")),
		}},
	}
}

func writeMCPDomainWorkStatusHTTPV0(
	w http.ResponseWriter,
	status int,
	result MCPDomainWorkStatusResultV0,
) {
	w.Header().Set("Content-Type", "application/json")
	if result.CorrelationID != "" {
		w.Header().Set("X-Correlation-ID", result.CorrelationID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
