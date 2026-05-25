package orquestaweb

import orquestamcp "orquesta/modulos/orquesta-mcp"

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
	Runs            []WebAutoprogrammingRunProgressV0           `json:"runs,omitempty"`
	Agents          []WebAutoprogrammingAgentProgressV0         `json:"agents,omitempty"`
	Diagnostics     []WebAutoprogrammingDiagnosticV0            `json:"diagnostics,omitempty"`
	ErroresPublicos []WebAutoprogrammingPrepareRunPublicIssueV0 `json:"errores_publicos,omitempty"`
}

type WebAutoprogrammingRunProgressV0 struct {
	WebRunQueueCandidateV0
	CurrentPhase      string `json:"current_phase,omitempty"`
	PercentComplete   int    `json:"percent_complete,omitempty"`
	TasksTotal        int    `json:"tasks_total,omitempty"`
	TasksClosed       int    `json:"tasks_closed,omitempty"`
	AgentsInFlight    int    `json:"agents_in_flight,omitempty"`
	AgentsFailed      int    `json:"agents_failed,omitempty"`
	ProgressingAgents int    `json:"progressing_agents,omitempty"`
	StalledAgents     int    `json:"stalled_agents,omitempty"`
	ClosureStatus     string `json:"closure_status,omitempty"`
	NeedsAttention    bool   `json:"needs_attention,omitempty"`
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
		Diagnostics:     webAutoprogrammingDiagnosticsV0(result.Diagnostics),
		ErroresPublicos: webAutoprogrammingPrepareRunIssuesV0(result.Errores),
	}
	if result.Queue != nil {
		vm.QueueLive = result.Queue.Estado == WebAutoprogrammingPrepareRunEstadoOKV0
		vm.Runs = append(vm.Runs, webAutoprogrammingQueueRunsV0(result.Queue.Ranked)...)
	}
	if result.Run != nil && result.Run.Stats != nil {
		vm.RunLive = result.Run.Estado == WebAutoprogrammingPrepareRunEstadoOKV0
		vm.Runs = append(vm.Runs, webAutoprogrammingRunProgressV0(result.Run))
		vm.Agents = webAutoprogrammingAgentProgressV0(result.Run)
	}
	return vm
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
		CurrentPhase:      trimV0(stats.CurrentPhase),
		PercentComplete:   stats.Progress.PercentComplete,
		TasksTotal:        stats.Counts.TasksTotal,
		TasksClosed:       stats.Counts.TasksClosed,
		AgentsInFlight:    stats.Counts.AgentsInFlight,
		AgentsFailed:      stats.Counts.AgentsFailed,
		ProgressingAgents: stats.Progress.ProgressingAgents,
		StalledAgents:     stats.Progress.StalledAgents,
		ClosureStatus:     trimV0(stats.Closure.Status),
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
		out = append(out, item)
	}
	return out
}

func webAutoprogrammingDiagnosticsV0(values []orquestamcp.MCPAutoprogrammingDiagnosticV0) []WebAutoprogrammingDiagnosticV0 {
	out := make([]WebAutoprogrammingDiagnosticV0, 0, len(values))
	for _, value := range values {
		out = append(out, WebAutoprogrammingDiagnosticV0{
			Code: trimV0(value.Code), Scope: trimV0(value.Scope), Message: trimV0(value.Message), EvidenceRefs: value.EvidenceRefs,
		})
	}
	return out
}
