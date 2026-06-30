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
	MCPQueueGlobalStatusHTTPPathV0          = "/api/v0/queue/global-status"
	MCPQueueGlobalStatusSchemaVersionV0     = "queue_global_status.v0"
	defaultMCPQueueGlobalStatusQueueRefV0   = "global"
	defaultMCPQueueGlobalStatusQueueLimitV0 = 200
)

const defaultMCPQueueGlobalStatusHTTPResponseTimeoutV0 = 2 * time.Second

const (
	mcpQueueGlobalStatusActionReviewReplanGoalFirstV0          = "review_replan_goal_first"
	mcpQueueGlobalStatusActionRetryFromPhaseV0                 = "retry_from_phase"
	mcpQueueGlobalStatusActionCloseSupersededByLocalEvidenceV0 = "close_superseded_by_local_evidence"
)

type MCPQueueGlobalStatusResultV0 struct {
	SchemaVersion string                              `json:"schema_version"`
	Estado        string                              `json:"estado"`
	RequestID     string                              `json:"request_id,omitempty"`
	CorrelationID string                              `json:"correlation_id,omitempty"`
	QueueRef      string                              `json:"queue_ref,omitempty"`
	Summary       MCPQueueGlobalStatusSummaryV0       `json:"summary"`
	Items         []MCPQueueGlobalStatusItemV0        `json:"items,omitempty"`
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
	NeedsAction               bool `json:"needs_action,omitempty"`
	WillFinishAlone           bool `json:"will_finish_alone"`
}

type MCPQueueGlobalStatusItemV0 struct {
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

func NewMCPQueueGlobalStatusHTTPHandlerV0(
	executor MCPTransportAutoprogrammingStatusExecutorV0,
) http.Handler {
	return newMCPQueueGlobalStatusHTTPHandlerWithTimeoutV0(
		executor,
		defaultMCPQueueGlobalStatusHTTPResponseTimeoutV0,
	)
}

func newMCPQueueGlobalStatusHTTPHandlerWithTimeoutV0(
	executor MCPTransportAutoprogrammingStatusExecutorV0,
	responseTimeout time.Duration,
) http.Handler {
	return mcpQueueGlobalStatusHTTPHandlerV0{
		executor:        executor,
		responseTimeout: responseTimeout,
	}
}

type mcpQueueGlobalStatusHTTPHandlerV0 struct {
	executor        MCPTransportAutoprogrammingStatusExecutorV0
	responseTimeout time.Duration
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
	result, err, timedOut := handler.executeQueueGlobalStatusWithResponseTimeoutV0(r, input)
	if timedOut {
		writeMCPQueueGlobalStatusHTTPV0(w, http.StatusGatewayTimeout, result)
		return
	}
	if err != nil {
		writeMCPQueueGlobalStatusHTTPV0(w, http.StatusInternalServerError, newMCPQueueGlobalStatusHTTPErrorV0(
			input,
			firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
			"executor",
			publicMCPExecutorErrorMessageFromErrorV0("queue_global_status_executor_error", err),
		))
		return
	}
	result.CorrelationID = firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), result.CorrelationID)
	httpStatus := http.StatusOK
	if result.Estado == MCPAutoprogrammingStatusEstadoErrorV0 {
		httpStatus = http.StatusBadRequest
	}
	writeMCPQueueGlobalStatusHTTPV0(w, httpStatus, result)
}

type mcpQueueGlobalStatusHTTPExecutionV0 struct {
	result MCPAutoprogrammingStatusToolResultV0
	err    error
}

func (handler mcpQueueGlobalStatusHTTPHandlerV0) executeQueueGlobalStatusWithResponseTimeoutV0(
	r *http.Request,
	input MCPAutoprogrammingStatusToolInputV0,
) (MCPQueueGlobalStatusResultV0, error, bool) {
	timeout := handler.responseTimeout
	if timeout <= 0 {
		status, err := handler.executor.Execute(r.Context(), input)
		return newMCPQueueGlobalStatusResultV0(input, status), err, false
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan mcpQueueGlobalStatusHTTPExecutionV0, 1)
	go func() {
		status, err := handler.executor.Execute(ctx, input)
		done <- mcpQueueGlobalStatusHTTPExecutionV0{result: status, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case execution := <-done:
		return newMCPQueueGlobalStatusResultV0(input, execution.result), execution.err, false
	case <-timer.C:
		return newMCPQueueGlobalStatusTimeoutResultV0(r, input), nil, true
	}
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
	items := mcpQueueGlobalStatusItemsV0(status.Queue, activeRuns, safeActions, diagnostics, status.StaleRunning)
	summary := mcpQueueGlobalStatusSummaryV0(status.QueueHealth, activeRuns, goalRunRefs, safeActions, diagnostics, status.StaleRunning)
	summary.NeedsAction = mcpQueueGlobalStatusNeedsActionV0(summary, items)
	summary.WillFinishAlone = !summary.NeedsAction
	return MCPQueueGlobalStatusResultV0{
		SchemaVersion: MCPQueueGlobalStatusSchemaVersionV0,
		Estado:        firstNonEmptyMCPV0(status.Estado, MCPAutoprogrammingStatusEstadoOKV0),
		RequestID:     firstNonEmptyMCPV0(status.RequestID, input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(status.CorrelationID, input.CorrelationID, input.RequestID),
		QueueRef:      firstNonEmptyMCPV0(status.QueueRef, input.QueueRef),
		Summary:       summary,
		Items:         items,
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

func mcpQueueGlobalStatusItemsV0(
	queue *MCPRunQueuePriorityToolResultV0,
	activeRuns []MCPAutoprogrammingActiveRunV0,
	safeActions []MCPAutoprogrammingSafeActionV0,
	diagnostics []MCPAutoprogrammingDiagnosticV0,
	staleRunning []MCPAutoprogrammingActionableRunV0,
) []MCPQueueGlobalStatusItemV0 {
	byRunRef := map[string]*MCPQueueGlobalStatusItemV0{}
	order := []string{}
	queueItem := (*MCPQueueGlobalStatusItemV0)(nil)
	ensure := func(runRef string) *MCPQueueGlobalStatusItemV0 {
		runRef = strings.TrimSpace(runRef)
		if runRef == "" {
			if queueItem == nil {
				queueItem = &MCPQueueGlobalStatusItemV0{Status: "queue_needs_action"}
				order = append(order, "")
			}
			return queueItem
		}
		if item := byRunRef[runRef]; item != nil {
			return item
		}
		item := &MCPQueueGlobalStatusItemV0{RunRef: runRef}
		byRunRef[runRef] = item
		order = append(order, runRef)
		return item
	}
	if queue != nil {
		for _, candidate := range queue.Ranked {
			item := ensure(candidate.RunRef)
			item.AppRef = firstNonEmptyMCPV0(item.AppRef, candidate.AppRef)
			item.Status = firstNonEmptyMCPV0(
				mcpQueueGlobalStatusCandidateStatusV0(candidate),
				item.Status,
				"queued",
			)
			item.EvidenceRefs = compactStringsMCPV0(append(item.EvidenceRefs, candidate.EvidenceRefs...))
		}
		for _, candidate := range queue.Terminal {
			item := ensure(candidate.RunRef)
			item.AppRef = firstNonEmptyMCPV0(item.AppRef, candidate.AppRef)
			item.Status = firstNonEmptyMCPV0(
				mcpQueueGlobalStatusCandidateStatusV0(candidate),
				item.Status,
				mcpAutoprogrammingHealthCompletedV0,
			)
			item.EvidenceRefs = compactStringsMCPV0(append(item.EvidenceRefs, candidate.EvidenceRefs...))
		}
	}
	for _, active := range activeRuns {
		item := ensure(active.RunRef)
		item.AppRef = firstNonEmptyMCPV0(item.AppRef, active.AppRef)
		if mcpQueueGlobalStatusCanPromoteActiveRunV0(item.Status) {
			item.Status = mcpQueueGlobalStatusActiveRunStatusV0(active)
		}
	}
	for _, stale := range staleRunning {
		item := ensure(stale.RunRef)
		item.AppRef = firstNonEmptyMCPV0(item.AppRef, stale.AppRef)
		item.Status = firstNonEmptyMCPV0(stale.Code, "running_stale")
		item.NeedsAction = true
		item.RecommendedAction = mcpQueueGlobalStatusNormalizeRecommendedActionV0(
			stale.RecommendedAction,
			"cancel_stale",
		)
		item.CurrentPhase = firstNonEmptyMCPV0(item.CurrentPhase, stale.CurrentPhase)
		item.DomainCounters = mergeMCPQueueGlobalStatusCountersV0(item.DomainCounters, stale.DomainCounters)
		item.EvidenceRefs = compactStringsMCPV0(append(item.EvidenceRefs, stale.EvidenceRefs...))
	}
	for _, action := range safeActions {
		item := ensure(action.RunRef)
		if mcpQueueGlobalStatusShouldPromoteSafeActionStatusV0(item.Status, action) {
			item.Status = mcpQueueGlobalStatusSafeActionStatusV0(action)
		} else {
			item.Status = firstNonEmptyMCPV0(item.Status, mcpQueueGlobalStatusSafeActionStatusV0(action))
		}
		item.NeedsAction = true
		item.RecommendedAction = firstNonEmptyMCPV0(
			item.RecommendedAction,
			mcpQueueGlobalStatusRecommendedActionFromSafeActionV0(action),
		)
		item.EvidenceRefs = compactStringsMCPV0(append(
			item.EvidenceRefs,
			mcpQueueGlobalStatusEvidenceRefsFromSafeActionV0(action)...,
		))
	}
	for _, diagnostic := range diagnostics {
		runRef := mcpQueueGlobalStatusDiagnosticRunRefV0(diagnostic)
		if runRef == "" && strings.TrimSpace(diagnostic.Code) != mcpAutoprogrammingQueuedNotDispatchedV0 {
			continue
		}
		item := ensure(runRef)
		item.Status = firstNonEmptyMCPV0(
			mcpQueueGlobalStatusStatusFromDiagnosticV0(diagnostic),
			item.Status,
			"needs_action",
		)
		item.NeedsAction = true
		item.RecommendedAction = firstNonEmptyMCPV0(
			item.RecommendedAction,
			mcpQueueGlobalStatusRecommendedActionFromDiagnosticV0(diagnostic),
		)
		item.EvidenceRefs = compactStringsMCPV0(append(item.EvidenceRefs, diagnostic.EvidenceRefs...))
	}
	out := make([]MCPQueueGlobalStatusItemV0, 0, len(order))
	for _, runRef := range order {
		item := queueItem
		if runRef != "" {
			item = byRunRef[runRef]
		}
		if item == nil {
			continue
		}
		item.Status = firstNonEmptyMCPV0(item.Status, "unknown")
		mcpQueueGlobalStatusFinalizeItemV0(item)
		out = append(out, *item)
	}
	return out
}

func mcpQueueGlobalStatusFinalizeItemV0(item *MCPQueueGlobalStatusItemV0) {
	if item == nil {
		return
	}
	item.Status = firstNonEmptyMCPV0(strings.TrimSpace(item.Status), "unknown")
	if item.NeedsAction {
		item.RecommendedAction = mcpQueueGlobalStatusNormalizeRecommendedActionV0(
			item.RecommendedAction,
			mcpQueueGlobalStatusRecommendedActionForStatusV0(item.Status),
		)
		item.NoActionReason = ""
		return
	}
	if action := mcpQueueGlobalStatusRecommendedActionForStatusV0(item.Status); action != "" {
		item.NeedsAction = true
		item.RecommendedAction = action
		item.NoActionReason = ""
		return
	}
	item.NoActionReason = mcpQueueGlobalStatusNoActionReasonV0(item.Status)
}

func mergeMCPQueueGlobalStatusCountersV0(left map[string]int, right map[string]int) map[string]int {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	out := map[string]int{}
	for key, value := range left {
		key = strings.TrimSpace(key)
		if key == "" || value == 0 {
			continue
		}
		out[key] += value
	}
	for key, value := range right {
		key = strings.TrimSpace(key)
		if key == "" || value == 0 {
			continue
		}
		out[key] += value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mcpQueueGlobalStatusShouldPromoteSafeActionStatusV0(
	status string,
	action MCPAutoprogrammingSafeActionV0,
) bool {
	switch strings.TrimSpace(action.Action) {
	case "observe_goal", "observe_active_goals":
	default:
		return false
	}
	switch strings.TrimSpace(status) {
	case "",
		mcpAutoprogrammingHealthQueuedV0,
		"ready",
		mcpAutoprogrammingHealthRunningLiveV0,
		mcpAutoprogrammingHealthRunningWithoutRecentStatsV0:
		return true
	default:
		return false
	}
}

func mcpQueueGlobalStatusRecommendedActionForStatusV0(status string) string {
	switch strings.TrimSpace(status) {
	case mcpAutoprogrammingActionGoalFirstStateMissingV0:
		return "repair_goal_state"
	case mcpAutoprogrammingActionGoalFirstBlockedV0:
		return mcpQueueGlobalStatusActionReviewReplanGoalFirstV0
	case "queued_not_dispatched":
		return "reencolar"
	case mcpAutoprogrammingHealthRunningStaleV0,
		mcpAutoprogrammingHealthRunningStaleNoProcessV0,
		"stale_running":
		return "cancel_stale"
	case mcpAutoprogrammingHealthRunningWithoutRecentStatsV0,
		mcpAutoprogrammingHealthBlockedV0,
		mcpAutoprogrammingHealthLostV0,
		mcpAutoprogrammingHealthFailedV0,
		mcpAutoprogrammingHealthUnclassifiedV0,
		"needs_action",
		"queue_needs_action",
		"unknown":
		return "repair_runtime"
	case "observer_required":
		return "restart_observer"
	case "retry_required":
		return "retry"
	case "review_required":
		return mcpQueueGlobalStatusActionReviewReplanGoalFirstV0
	case "waiting_outbox":
		return "reencolar"
	default:
		return ""
	}
}

func mcpQueueGlobalStatusNoActionReasonV0(status string) string {
	switch strings.TrimSpace(status) {
	case mcpAutoprogrammingHealthRunningLiveV0:
		return "running_live_wait_processes"
	case mcpAutoprogrammingHealthQueuedV0, "ready":
		return "queued_waiting_scheduler"
	case mcpAutoprogrammingHealthCompletedV0, "accepted", "closed", "delivered":
		return "terminal_completed"
	default:
		return "no_operator_action_required"
	}
}

func mcpQueueGlobalStatusCanPromoteActiveRunV0(status string) bool {
	switch strings.TrimSpace(status) {
	case "", "ready", mcpAutoprogrammingHealthQueuedV0, mcpAutoprogrammingHealthRunningWithoutRecentStatsV0:
		return true
	default:
		return false
	}
}

func mcpQueueGlobalStatusNeedsActionV0(
	summary MCPQueueGlobalStatusSummaryV0,
	items []MCPQueueGlobalStatusItemV0,
) bool {
	if summary.NeedsAttention ||
		summary.QueuedNotDispatched > 0 ||
		summary.RunningWithoutRecentStats > 0 ||
		summary.SafeActions > 0 {
		return true
	}
	for _, item := range items {
		if item.NeedsAction {
			return true
		}
	}
	return false
}

func mcpQueueGlobalStatusCandidateStatusV0(candidate MCPRunQueueRankedCandidateCompactV0) string {
	if mcpAutoprogrammingQueuedNotDispatchedStatusV0(candidate.Status) {
		return "ready"
	}
	return classifyMCPAutoprogrammingQueueStatusV0(candidate.Status)
}

func mcpQueueGlobalStatusStatusFromDiagnosticV0(diagnostic MCPAutoprogrammingDiagnosticV0) string {
	switch strings.TrimSpace(diagnostic.Code) {
	case "autoprogramming_goal_first_state_missing":
		return mcpAutoprogrammingActionGoalFirstStateMissingV0
	case "autoprogramming_goal_first_blocked":
		return mcpAutoprogrammingActionGoalFirstBlockedV0
	default:
		return strings.TrimSpace(diagnostic.Code)
	}
}

func mcpQueueGlobalStatusActiveRunStatusV0(active MCPAutoprogrammingActiveRunV0) string {
	switch strings.ToLower(strings.TrimSpace(active.Status)) {
	case "", "running":
		return mcpAutoprogrammingHealthRunningLiveV0
	default:
		return strings.ToLower(strings.TrimSpace(active.Status))
	}
}

func mcpQueueGlobalStatusSafeActionStatusV0(action MCPAutoprogrammingSafeActionV0) string {
	switch strings.TrimSpace(action.Action) {
	case "observe_goal", "observe_active_goals":
		return "observer_required"
	case "retry":
		return "retry_required"
	case "review":
		return "review_required"
	case "supervise":
		return "waiting_outbox"
	default:
		return "needs_action"
	}
}

func mcpQueueGlobalStatusRecommendedActionFromSafeActionV0(action MCPAutoprogrammingSafeActionV0) string {
	switch strings.TrimSpace(action.Action) {
	case "observe_goal", "observe_active_goals":
		return "restart_observer"
	case "retry":
		return "retry"
	case "review":
		return mcpQueueGlobalStatusActionReviewReplanGoalFirstV0
	case "supervise":
		return "reencolar"
	default:
		return mcpQueueGlobalStatusNormalizeRecommendedActionV0(action.Action, "repair_runtime")
	}
}

func mcpQueueGlobalStatusRecommendedActionFromDiagnosticV0(diagnostic MCPAutoprogrammingDiagnosticV0) string {
	switch strings.TrimSpace(diagnostic.Code) {
	case mcpAutoprogrammingQueuedNotDispatchedV0:
		return "reencolar"
	case mcpAutoprogrammingActionGoalFirstStateMissingV0, "autoprogramming_goal_first_state_missing":
		return "repair_goal_state"
	case mcpAutoprogrammingActionGoalFirstBlockedV0, "autoprogramming_goal_first_blocked":
		return mcpQueueGlobalStatusActionReviewReplanGoalFirstV0
	default:
		return "repair_runtime"
	}
}

func mcpQueueGlobalStatusNormalizeRecommendedActionV0(action string, fallback string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "retry",
		"reencolar",
		"cancel_stale",
		"restart_observer",
		"repair_runtime",
		"repair_goal_state",
		"inspect_liveness",
		mcpQueueGlobalStatusActionReviewReplanGoalFirstV0,
		mcpQueueGlobalStatusActionRetryFromPhaseV0,
		mcpQueueGlobalStatusActionCloseSupersededByLocalEvidenceV0:
		return action
	}
	if strings.Contains(action, "goal_state") ||
		strings.Contains(action, "repair_goal") ||
		strings.Contains(action, "state_missing") {
		return "repair_goal_state"
	}
	if strings.Contains(action, "process_ref") ||
		strings.Contains(action, "liveness") ||
		strings.Contains(action, "live_process") ||
		strings.Contains(action, "before_reconcile") {
		return "inspect_liveness"
	}
	if strings.Contains(action, "observe") || strings.Contains(action, "observer") {
		return "restart_observer"
	}
	if strings.Contains(action, "queue") ||
		strings.Contains(action, "reencol") ||
		strings.Contains(action, "supervis") ||
		strings.Contains(action, "dispatch") {
		return "reencolar"
	}
	if strings.Contains(action, "retry") ||
		strings.Contains(action, "relaunch") ||
		strings.Contains(action, "replan") {
		return "retry"
	}
	if strings.Contains(action, "cancel") || strings.Contains(action, "stale") {
		return "cancel_stale"
	}
	return firstNonEmptyMCPV0(fallback, "repair_runtime")
}

func mcpQueueGlobalStatusEvidenceRefsFromSafeActionV0(action MCPAutoprogrammingSafeActionV0) []string {
	switch strings.TrimSpace(action.Action) {
	case "observe_goal", "observe_active_goals":
		return []string{"evidence-ref-queue-global-status-observer-required"}
	case "retry":
		return []string{"evidence-ref-queue-global-status-retry-required"}
	case "review":
		return []string{"evidence-ref-queue-global-status-review-required"}
	case "supervise":
		return []string{"evidence-ref-queue-global-status-supervision-required"}
	default:
		return []string{"evidence-ref-queue-global-status-operator-action"}
	}
}

func mcpQueueGlobalStatusDiagnosticRunRefV0(diagnostic MCPAutoprogrammingDiagnosticV0) string {
	const prefix = "run:"
	scope := strings.TrimSpace(diagnostic.Scope)
	if strings.HasPrefix(scope, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(scope, prefix))
	}
	return ""
}

func newMCPQueueGlobalStatusTimeoutResultV0(
	r *http.Request,
	input MCPAutoprogrammingStatusToolInputV0,
) MCPQueueGlobalStatusResultV0 {
	result := MCPQueueGlobalStatusResultV0{
		SchemaVersion: MCPQueueGlobalStatusSchemaVersionV0,
		Estado:        MCPAutoprogrammingStatusEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(r.Header.Get("X-Correlation-ID"), input.CorrelationID, input.RequestID),
		QueueRef:      firstNonEmptyMCPV0(input.QueueRef, defaultMCPQueueGlobalStatusQueueRefV0),
		Summary:       MCPQueueGlobalStatusSummaryV0{},
		Diagnostics: []MCPAutoprogrammingDiagnosticV0{{
			Code:         "queue_global_status_timeout",
			Scope:        "executor",
			Message:      "consulta global de cola cancelada por timeout HTTP; reintentar lectura acotada o revisar runtime",
			EvidenceRefs: []string{"evidence-ref-queue-global-status-timeout"},
		}},
		Errores: []MCPValidationIssueV0{{
			Code:    "queue_global_status_timeout",
			Field:   "executor",
			Message: "consulta global de cola excedio la ventana HTTP acotada",
		}},
	}
	result.Summary.NeedsAction = true
	result.Summary.WillFinishAlone = false
	return result
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
