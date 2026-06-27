package orquestaweb

import (
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type WebAutoprogrammingStatusQueryV0 struct {
	RequestID            string   `json:"request_id,omitempty"`
	CorrelationID        string   `json:"correlation_id,omitempty"`
	Locale               string   `json:"locale,omitempty"`
	OccurredAt           string   `json:"occurred_at,omitempty"`
	RunRef               string   `json:"run_ref,omitempty"`
	AppRef               string   `json:"app_ref,omitempty"`
	ExternalJobRef       string   `json:"external_job_ref,omitempty"`
	QueueRef             string   `json:"queue_ref,omitempty"`
	AppRefs              []string `json:"app_refs,omitempty"`
	QueueLimit           int      `json:"queue_limit,omitempty"`
	IncludeProcessRefs   bool     `json:"include_process_refs,omitempty"`
	IncludeAgentProgress bool     `json:"include_agent_progress,omitempty"`
	IncludeAgentUsage    bool     `json:"include_agent_usage,omitempty"`
}

type WebAutoprogrammingStatusViewModelV0 struct {
	SchemaVersion   string                                      `json:"schema_version"`
	Locale          string                                      `json:"locale,omitempty"`
	Estado          string                                      `json:"estado"`
	QueueLive       bool                                        `json:"queue_live"`
	RunLive         bool                                        `json:"run_live"`
	QueueRef        string                                      `json:"queue_ref,omitempty"`
	RunRef          string                                      `json:"run_ref,omitempty"`
	QueueHealth     *WebAutoprogrammingQueueHealthV0            `json:"queue_health,omitempty"`
	StaleRunning    []WebAutoprogrammingActionableRunV0         `json:"stale_running,omitempty"`
	SafeActions     []WebAutoprogrammingSafeActionV0            `json:"safe_actions,omitempty"`
	Runs            []WebAutoprogrammingRunProgressV0           `json:"runs,omitempty"`
	Agents          []WebAutoprogrammingAgentProgressV0         `json:"agents,omitempty"`
	Diagnostics     []WebAutoprogrammingDiagnosticV0            `json:"diagnostics,omitempty"`
	ErroresPublicos []WebAutoprogrammingPrepareRunPublicIssueV0 `json:"errores_publicos,omitempty"`
}

type WebAutoprogrammingQueueHealthV0 struct {
	Queued                    int `json:"queued,omitempty"`
	RunningLive               int `json:"running_live,omitempty"`
	AgentsLive                int `json:"agents_live,omitempty"`
	RunningStale              int `json:"running_stale,omitempty"`
	RunningStaleNoProcess     int `json:"running_stale_no_process,omitempty"`
	RunningWithoutRecentStats int `json:"running_without_recent_stats,omitempty"`
	Blocked                   int `json:"blocked,omitempty"`
	Lost                      int `json:"lost,omitempty"`
	Completed                 int `json:"completed,omitempty"`
	Failed                    int `json:"failed,omitempty"`
	Unclassified              int `json:"unclassified,omitempty"`
	ObservedRuns              int `json:"observed_runs,omitempty"`
	QueueRuns                 int `json:"queue_runs,omitempty"`
	StatsRuns                 int `json:"stats_runs,omitempty"`
}

type WebAutoprogrammingActionableRunV0 struct {
	Code              string   `json:"code"`
	Severity          string   `json:"severity,omitempty"`
	RunRef            string   `json:"run_ref,omitempty"`
	AppRef            string   `json:"app_ref,omitempty"`
	Status            string   `json:"status,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	RecommendedAction string   `json:"recommended_action,omitempty"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type WebAutoprogrammingSafeActionV0 struct {
	Action       string         `json:"action"`
	Scope        string         `json:"scope,omitempty"`
	RunRef       string         `json:"run_ref,omitempty"`
	Method       string         `json:"method,omitempty"`
	Endpoint     string         `json:"endpoint,omitempty"`
	Reason       string         `json:"reason,omitempty"`
	RequiresPost bool           `json:"requires_post,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
}

type WebAutoprogrammingRunProgressV0 struct {
	WebRunQueueCandidateV0
	CurrentPhase       string   `json:"current_phase,omitempty"`
	PercentComplete    int      `json:"percent_complete,omitempty"`
	TasksTotal         int      `json:"tasks_total,omitempty"`
	TasksClosed        int      `json:"tasks_closed,omitempty"`
	AgentsInFlight     int      `json:"agents_in_flight,omitempty"`
	AgentsFailed       int      `json:"agents_failed,omitempty"`
	ProgressingAgents  int      `json:"progressing_agents,omitempty"`
	StalledAgents      int      `json:"stalled_agents,omitempty"`
	ClosureStatus      string   `json:"closure_status,omitempty"`
	ClosureBlockedBy   []string `json:"closure_blocked_by,omitempty"`
	ClosureBlockerRefs []string `json:"closure_blocker_refs,omitempty"`
	NeedsAttention     bool     `json:"needs_attention,omitempty"`
}

type WebAutoprogrammingAgentProgressV0 struct {
	AgentRef            string   `json:"agent_ref"`
	Status              string   `json:"status,omitempty"`
	InFlight            bool     `json:"in_flight,omitempty"`
	NeedsAttention      bool     `json:"needs_attention,omitempty"`
	CanStop             bool     `json:"can_stop,omitempty"`
	TaskRef             string   `json:"task_ref,omitempty"`
	ProcessRef          string   `json:"process_ref,omitempty"`
	SessionRef          string   `json:"session_ref,omitempty"`
	RuntimeKind         string   `json:"runtime_kind,omitempty"`
	CapacityLevel       string   `json:"capacity_level,omitempty"`
	QuotaStatus         string   `json:"quota_status,omitempty"`
	TotalTokens         int64    `json:"total_tokens,omitempty"`
	ProgressStatus      string   `json:"progress_status,omitempty"`
	NoProgressTicks     int      `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int      `json:"repeated_action_count,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
}

type WebAutoprogrammingDiagnosticV0 struct {
	Code         string   `json:"code"`
	Scope        string   `json:"scope,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

func NewWebAutoprogrammingStatusViewModelV0(locale string, result orquestamcp.MCPAutoprogrammingStatusToolResultV0) WebAutoprogrammingStatusViewModelV0 {
	vm := WebAutoprogrammingStatusViewModelV0{
		SchemaVersion:   "web_autoprogramming_status.v0",
		Locale:          normalizeDirectorStatsLocaleV0(locale),
		Estado:          webAutoprogrammingPrepareRunEstadoV0(result.Estado),
		QueueRef:        trimV0(result.QueueRef),
		RunRef:          trimV0(result.RunRef),
		StaleRunning:    webAutoprogrammingActionableRunsV0(result.StaleRunning),
		Diagnostics:     webAutoprogrammingDiagnosticsV0(result.Diagnostics),
		ErroresPublicos: webAutoprogrammingPrepareRunIssuesV0(result.Errores),
	}
	if result.Operator != nil {
		vm.SafeActions = webAutoprogrammingSafeActionsV0(result.Operator.SafeActions)
	}
	if result.Queue != nil {
		vm.QueueLive = result.Queue.Estado == WebAutoprogrammingPrepareRunEstadoOKV0
		vm.Runs = append(vm.Runs, webAutoprogrammingQueueRunsV0(result.Queue.Ranked)...)
	}
	vm.QueueHealth = webAutoprogrammingQueueHealthV0(result.QueueHealth)
	if result.Run != nil && result.Run.Stats != nil {
		vm.RunLive = result.Run.Estado == WebAutoprogrammingPrepareRunEstadoOKV0
		vm.Runs = append(vm.Runs, webAutoprogrammingRunProgressV0(result.Run))
		vm.Agents = webAutoprogrammingAgentProgressV0(result.Run)
	}
	return vm
}

func webAutoprogrammingQueueHealthV0(
	value *orquestamcp.MCPAutoprogrammingQueueHealthV0,
) *WebAutoprogrammingQueueHealthV0 {
	if value == nil {
		return nil
	}
	return &WebAutoprogrammingQueueHealthV0{
		Queued:                    value.Queued,
		RunningLive:               value.RunningLive,
		AgentsLive:                value.AgentsLive,
		RunningStale:              value.RunningStale,
		RunningStaleNoProcess:     value.RunningStaleNoProcess,
		RunningWithoutRecentStats: value.RunningWithoutRecentStats,
		Blocked:                   value.Blocked,
		Lost:                      value.Lost,
		Completed:                 value.Completed,
		Failed:                    value.Failed,
		Unclassified:              value.Unclassified,
		ObservedRuns:              value.ObservedRuns,
		QueueRuns:                 value.QueueRuns,
		StatsRuns:                 value.StatsRuns,
	}
}

func webAutoprogrammingActionableRunsV0(
	values []orquestamcp.MCPAutoprogrammingActionableRunV0,
) []WebAutoprogrammingActionableRunV0 {
	out := make([]WebAutoprogrammingActionableRunV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebAutoprogrammingActionableRunV0{
			Code:              webAutoprogrammingPublicActionTextV0(value.Code),
			Severity:          webAutoprogrammingPublicActionTextV0(value.Severity),
			RunRef:            trimV0(value.RunRef),
			AppRef:            trimV0(value.AppRef),
			Status:            webAutoprogrammingPublicActionTextV0(value.Status),
			Reason:            webAutoprogrammingPublicActionTextV0(value.Reason),
			RecommendedAction: webAutoprogrammingPublicActionTextV0(value.RecommendedAction),
			EvidenceRefs:      webAutoprogrammingPublicEvidenceRefsV0(value.EvidenceRefs),
		})
	}
	return out
}

func webAutoprogrammingPublicActionTextV0(value string) string {
	value = trimV0(value)
	value = strings.ReplaceAll(value, "provider_usage", "usage")
	value = strings.ReplaceAll(value, "provider-", "")
	return value
}

func webAutoprogrammingPublicEvidenceRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, webAutoprogrammingPublicActionTextV0(value))
	}
	return compactStringsV0(out)
}

func webAutoprogrammingSafeActionsV0(
	values []orquestamcp.MCPAutoprogrammingSafeActionV0,
) []WebAutoprogrammingSafeActionV0 {
	out := make([]WebAutoprogrammingSafeActionV0, 0, len(values))
	for _, value := range values {
		action := WebAutoprogrammingSafeActionV0{
			Action:       webAutoprogrammingPublicActionTextV0(value.Action),
			Scope:        webAutoprogrammingPublicActionTextV0(value.Scope),
			RunRef:       trimV0(value.RunRef),
			Method:       webAutoprogrammingPublicActionTextV0(value.Method),
			Endpoint:     webAutoprogrammingPublicEndpointV0(value.Endpoint),
			Reason:       webAutoprogrammingPublicActionTextV0(value.Reason),
			RequiresPost: value.RequiresPost,
			Payload:      webAutoprogrammingSafeActionPayloadV0(value.Payload),
		}
		if action.Action == "" && action.Endpoint == "" {
			continue
		}
		out = append(out, action)
	}
	return out
}

func webAutoprogrammingPublicEndpointV0(value string) string {
	value = trimV0(value)
	switch value {
	case orquestamcp.MCPAutoprogrammingSuperviseHTTPPathV0,
		orquestamcp.MCPAutoprogrammingObserveGoalHTTPPathV0:
		return value
	default:
		return ""
	}
}

func webAutoprogrammingSafeActionPayloadV0(value map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{
		"director_execution_mode",
		"run_ref",
		"resident_mode",
		"max_ticks",
		"max_runs_per_tick",
		"max_executions",
		"max_dispatches_per_wait",
		"max_outbox_per_cycle",
	} {
		raw, ok := value[key]
		if !ok {
			continue
		}
		if normalized, ok := webAutoprogrammingPublicPayloadScalarV0(raw); ok {
			out[key] = normalized
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func webAutoprogrammingPublicPayloadScalarV0(value any) (any, bool) {
	switch typed := value.(type) {
	case string:
		trimmed := trimV0(typed)
		return trimmed, trimmed != ""
	case bool:
		return typed, true
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return typed, true
	case uint:
		return typed, true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return typed, true
	case uint64:
		return typed, true
	case float32:
		return typed, true
	case float64:
		return typed, true
	default:
		return nil, false
	}
}

func webAutoprogrammingQueueRunsV0(values []orquestamcp.MCPRunQueueRankedCandidateCompactV0) []WebAutoprogrammingRunProgressV0 {
	out := make([]WebAutoprogrammingRunProgressV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebAutoprogrammingRunProgressV0{
			WebRunQueueCandidateV0: webRunQueueCandidateV0(value),
		})
	}
	return out
}

func webAutoprogrammingRunProgressV0(result *orquestamcp.MCPDirectorStatsToolResultV0) WebAutoprogrammingRunProgressV0 {
	stats := result.Stats
	return WebAutoprogrammingRunProgressV0{
		WebRunQueueCandidateV0: WebRunQueueCandidateV0{
			RunRef: trimV0(stats.RunRef), AppRef: trimV0(stats.ProjectRef), Status: trimV0(stats.Status),
		},
		CurrentPhase:       trimV0(stats.CurrentPhase),
		PercentComplete:    webAutoprogrammingLivePercentV0(stats),
		TasksTotal:         stats.Counts.TasksTotal,
		TasksClosed:        stats.Counts.TasksClosed,
		AgentsInFlight:     stats.Counts.AgentsInFlight,
		AgentsFailed:       stats.Counts.AgentsFailed,
		ProgressingAgents:  stats.Progress.ProgressingAgents,
		StalledAgents:      stats.Progress.StalledAgents,
		ClosureStatus:      trimV0(stats.Closure.Status),
		ClosureBlockedBy:   compactStringsV0(stats.Closure.BlockedBy),
		ClosureBlockerRefs: compactStringsV0(stats.Closure.BlockerRefs),
		NeedsAttention: stats.Counts.AgentsFailed > 0 ||
			stats.Progress.StalledAgents > 0 || stats.Closure.Blocked,
	}
}

func webAutoprogrammingAgentProgressV0(result *orquestamcp.MCPDirectorStatsToolResultV0) []WebAutoprogrammingAgentProgressV0 {
	out := make([]WebAutoprogrammingAgentProgressV0, 0, len(result.Stats.Agents))
	for _, agent := range result.Stats.Agents {
		item := WebAutoprogrammingAgentProgressV0{
			AgentRef:       trimV0(agent.AgentRequestID),
			Status:         trimV0(agent.Status),
			InFlight:       agent.InFlight,
			NeedsAttention: agent.NeedsAttention,
			CanStop:        agent.CanStop,
		}
		if agent.LastProgress != nil {
			item.TaskRef = trimV0(agent.LastProgress.TaskRef)
			item.ProgressStatus = trimV0(agent.LastProgress.Status)
			item.NoProgressTicks = agent.LastProgress.NoProgressTicks
			item.RepeatedActionCount = agent.LastProgress.RepeatedActionCount
			item.EvidenceRefs = append(item.EvidenceRefs, agent.LastProgress.EvidenceRefs...)
		}
		if agent.Process != nil {
			item.ProcessRef = trimV0(agent.Process.ProcessRef)
			item.SessionRef = trimV0(agent.Process.SessionRef)
			item.EvidenceRefs = append(item.EvidenceRefs, agent.Process.EvidenceRefs...)
		}
		if agent.Usage != nil {
			item.RuntimeKind = trimV0(agent.Usage.RuntimeKind)
			item.CapacityLevel = trimV0(agent.Usage.CapacityLevel)
			item.QuotaStatus = trimV0(agent.Usage.QuotaStatus)
			item.TotalTokens = agent.Usage.TotalTokens
			item.EvidenceRefs = append(item.EvidenceRefs, agent.Usage.EvidenceRefs...)
		}
		item.EvidenceRefs = compactStringsV0(item.EvidenceRefs)
		out = append(out, item)
	}
	return out
}

func webAutoprogrammingDiagnosticsV0(values []orquestamcp.MCPAutoprogrammingDiagnosticV0) []WebAutoprogrammingDiagnosticV0 {
	out := make([]WebAutoprogrammingDiagnosticV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebAutoprogrammingDiagnosticV0{
			Code: trimV0(value.Code), Scope: trimV0(value.Scope), Message: trimV0(value.Message), EvidenceRefs: compactStringsV0(value.EvidenceRefs),
		})
	}
	return out
}
