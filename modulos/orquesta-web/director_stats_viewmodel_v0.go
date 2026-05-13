package orquestaweb

import "strings"

type WebDirectorStatsViewModelV0 struct {
	SchemaVersion   string                          `json:"schema_version"`
	Locale          string                          `json:"locale,omitempty"`
	RunRef          string                          `json:"run_ref,omitempty"`
	Estado          string                          `json:"estado"`
	FaseActual      string                          `json:"fase_actual,omitempty"`
	Textos          WebDirectorStatsTextsV0         `json:"textos"`
	Counts          WebDirectorStatsCountsV0        `json:"counts"`
	Progress        WebDirectorStatsProgressV0      `json:"progress"`
	Resumen         WebDirectorStatsSummaryV0       `json:"resumen"`
	Checkpoint      WebDirectorStatsCheckpointV0    `json:"checkpoint"`
	Agents          []WebDirectorStatsAgentV0       `json:"agents,omitempty"`
	Agentes         []WebDirectorStatsAgentV0       `json:"agentes,omitempty"`
	Tasks           []WebDirectorStatsTaskV0        `json:"tasks,omitempty"`
	Tareas          []WebDirectorStatsTaskV0        `json:"tareas,omitempty"`
	ErroresPublicos []WebDirectorStatsPublicIssueV0 `json:"errores_publicos,omitempty"`
}

type WebDirectorStatsPanelV0 = WebDirectorStatsViewModelV0

type WebDirectorStatsTextsV0 struct {
	Title              string `json:"title"`
	Counts             string `json:"counts"`
	TaskProgress       string `json:"task_progress"`
	AgentProgress      string `json:"agent_progress"`
	CheckpointProgress string `json:"checkpoint_progress"`
	PublicErrors       string `json:"public_errors"`
	Refresh            string `json:"refresh"`
}

type WebDirectorStatsCountsV0 struct {
	TasksTotal              int `json:"tasks_total"`
	TasksClosed             int `json:"tasks_closed"`
	TasksOpen               int `json:"tasks_open"`
	TasksDelivered          int `json:"tasks_delivered"`
	Brainstorms             int `json:"brainstorms"`
	AgentsRequested         int `json:"agents_requested"`
	AgentsStarted           int `json:"agents_started"`
	AgentsDelivered         int `json:"agents_delivered"`
	AgentsInFlight          int `json:"agents_in_flight"`
	AgentsNeedAttention     int `json:"agents_need_attention"`
	CheckpointAgentsPending int `json:"checkpoint_agents_pending"`
	Deliveries              int `json:"deliveries"`
	Reviews                 int `json:"reviews"`
	Blockers                int `json:"blockers"`
}

type WebDirectorStatsProgressV0 struct {
	SourceStatus       string   `json:"source_status"`
	PercentComplete    int      `json:"percent_complete"`
	ObservedAgents     int      `json:"observed_agents"`
	ProgressingAgents  int      `json:"progressing_agents"`
	StalledAgents      int      `json:"stalled_agents"`
	LoopDetectedAgents int      `json:"loop_detected_agents"`
	StoppedAgents      int      `json:"stopped_agents"`
	NoSignalAgentRefs  []string `json:"no_signal_agent_refs,omitempty"`
}

type WebDirectorStatsSummaryV0 struct {
	TasksTotal         int      `json:"tasks_total"`
	TasksClosed        int      `json:"tasks_closed"`
	TasksObserved      int      `json:"tasks_observed"`
	PercentComplete    int      `json:"percent_complete"`
	ProgressSource     string   `json:"progress_source"`
	ObservedAgents     int      `json:"observed_agents"`
	ProgressingAgents  int      `json:"progressing_agents"`
	StalledAgents      int      `json:"stalled_agents"`
	LoopDetectedAgents int      `json:"loop_detected_agents"`
	StoppedAgents      int      `json:"stopped_agents"`
	UsageAgents        int      `json:"usage_agents,omitempty"`
	UsageQuotaStatus   string   `json:"usage_quota_status,omitempty"`
	UsageTotalTokens   int64    `json:"usage_total_tokens,omitempty"`
	UsageCostMicros    int64    `json:"usage_cost_micros,omitempty"`
	NoSignalAgentRefs  []string `json:"no_signal_agent_refs"`
}

type WebDirectorStatsCheckpointV0 struct {
	CheckpointAgentsPending    int      `json:"checkpoint_agents_pending"`
	PendingCheckpointAgentRefs []string `json:"pending_checkpoint_agent_refs"`
	CheckpointEvidenceRefs     []string `json:"checkpoint_evidence_refs"`
	RequiresAttention          bool     `json:"requires_attention"`
}

type WebDirectorStatsAgentV0 struct {
	AgentRequestID      string `json:"agent_request_id"`
	Status              string `json:"status"`
	ControlState        string `json:"control_state,omitempty"`
	InFlight            bool   `json:"in_flight"`
	CanStop             bool   `json:"can_stop"`
	NeedsAttention      bool   `json:"needs_attention"`
	ProgressStatus      string `json:"progress_status,omitempty"`
	TaskRef             string `json:"task_ref,omitempty"`
	NoProgressTicks     int    `json:"no_progress_ticks,omitempty"`
	RepeatedActionCount int    `json:"repeated_action_count,omitempty"`
	ModelAlias          string `json:"model_alias,omitempty"`
	CapacityLevel       string `json:"capacity_level,omitempty"`
	QuotaStatus         string `json:"quota_status,omitempty"`
	QuotaRemaining      int64  `json:"quota_remaining,omitempty"`
	QuotaLimit          int64  `json:"quota_limit,omitempty"`
	TotalTokens         int64  `json:"total_tokens,omitempty"`
}

type WebDirectorStatsTaskV0 struct {
	TaskRef        string `json:"task_ref"`
	Status         string `json:"status"`
	AgentRequestID string `json:"agent_request_id,omitempty"`
	ProgressStatus string `json:"progress_status,omitempty"`
}

func NewWebDirectorStatsPanelV0(
	locale string,
	result WebDirectorStatsInboundResultV0,
) WebDirectorStatsPanelV0 {
	stats := result.Stats
	vm := WebDirectorStatsViewModelV0{
		SchemaVersion:   WebDirectorStatsPanelSchemaV0,
		Locale:          normalizeDirectorStatsLocaleV0(locale),
		RunRef:          firstDirectorStatsNonEmptyV0(result.RunRef, runRefFromStatsV0(stats)),
		Estado:          directorStatsEstadoV0(result.Estado, stats),
		Textos:          directorStatsTextsV0(locale),
		Agents:          []WebDirectorStatsAgentV0{},
		Agentes:         []WebDirectorStatsAgentV0{},
		Tasks:           []WebDirectorStatsTaskV0{},
		Tareas:          []WebDirectorStatsTaskV0{},
		ErroresPublicos: directorStatsIssuesV0(result.Errores, nil),
	}
	if stats == nil {
		return vm
	}
	vm.RunRef = firstDirectorStatsNonEmptyV0(vm.RunRef, stats.RunRef)
	vm.FaseActual = trimDirectorStatsV0(stats.CurrentPhase)
	vm.Counts = directorStatsCountsV0(*stats)
	vm.Progress = directorStatsProgressV0(stats.Progress)
	vm.Resumen = directorStatsSummaryV0(stats.Progress, stats.UsageSummary)
	vm.Checkpoint = directorStatsCheckpointV0(*stats)
	vm.Agents = directorStatsAgentsV0(stats.Agents)
	vm.Agentes = vm.Agents
	vm.Tasks = directorStatsTasksV0(stats.Progress.Tasks)
	vm.Tareas = vm.Tasks
	vm.ErroresPublicos = directorStatsIssuesV0(result.Errores, stats.Progress.Issues)
	if vm.Estado == WebDirectorStatsInboundEstadoOKV0 {
		vm.Estado = directorStatsAttentionStateV0(vm)
	}
	return vm
}

func NewWebDirectorStatsViewModelV0(
	stats WebDirectorRunStatsContractV0,
) WebDirectorStatsViewModelV0 {
	return NewWebDirectorStatsPanelV0("", WebDirectorStatsInboundResultV0{
		Estado: WebDirectorStatsInboundEstadoOKV0,
		RunRef: stats.RunRef,
		Stats:  &stats,
	})
}

func NewWebDirectorStatsErrorViewModelV0(
	runRef string,
	code string,
) WebDirectorStatsViewModelV0 {
	return NewWebDirectorStatsPanelV0("", WebDirectorStatsInboundResultV0{
		Estado: WebDirectorStatsInboundEstadoErrorV0,
		RunRef: runRef,
		Errores: []WebDirectorStatsPublicIssueV0{{
			Code:    trimDirectorStatsV0(code),
			Message: trimDirectorStatsV0(code),
		}},
	})
}

func directorStatsCountsV0(stats WebDirectorRunStatsContractV0) WebDirectorStatsCountsV0 {
	return WebDirectorStatsCountsV0{
		TasksTotal:              stats.Counts["tasks_total"],
		TasksClosed:             stats.Counts["tasks_closed"],
		TasksOpen:               stats.Counts["tasks_open"],
		TasksDelivered:          stats.Counts["tasks_delivered"],
		Brainstorms:             stats.Counts["brainstorms"],
		AgentsRequested:         stats.Counts["agents_requested"],
		AgentsStarted:           stats.Counts["agents_started"],
		AgentsDelivered:         stats.Counts["agents_delivered"],
		AgentsInFlight:          stats.Counts["agents_in_flight"],
		AgentsNeedAttention:     webDirectorStatsAttentionCountV0(stats.Agents),
		CheckpointAgentsPending: stats.CheckpointAgentsPending,
		Deliveries:              stats.Counts["deliveries"],
		Reviews:                 stats.Counts["reviews"],
		Blockers:                stats.Counts["blockers"],
	}
}

func directorStatsProgressV0(progress WebDirectorProgressStatsContractV0) WebDirectorStatsProgressV0 {
	return WebDirectorStatsProgressV0{
		SourceStatus:       trimDirectorStatsV0(progress.SourceStatus),
		PercentComplete:    progress.PercentComplete,
		ObservedAgents:     progress.ObservedAgents,
		ProgressingAgents:  progress.ProgressingAgents,
		StalledAgents:      progress.StalledAgents,
		LoopDetectedAgents: progress.LoopDetectedAgents,
		StoppedAgents:      progress.StoppedAgents,
		NoSignalAgentRefs:  compactOperationalStringsV0(progress.NoSignalAgentRefs),
	}
}

func directorStatsSummaryV0(
	progress WebDirectorProgressStatsContractV0,
	usage *WebDirectorRunUsageSummaryV0,
) WebDirectorStatsSummaryV0 {
	summary := WebDirectorStatsSummaryV0{
		TasksTotal:         progress.TasksTotal,
		TasksClosed:        progress.TasksClosed,
		TasksObserved:      progress.TasksObserved,
		PercentComplete:    progress.PercentComplete,
		ProgressSource:     trimDirectorStatsV0(progress.SourceStatus),
		ObservedAgents:     progress.ObservedAgents,
		ProgressingAgents:  progress.ProgressingAgents,
		StalledAgents:      progress.StalledAgents,
		LoopDetectedAgents: progress.LoopDetectedAgents,
		StoppedAgents:      progress.StoppedAgents,
		NoSignalAgentRefs:  compactOperationalStringsV0(progress.NoSignalAgentRefs),
	}
	if usage != nil {
		summary.UsageAgents = usage.AgentsObserved
		summary.UsageQuotaStatus = trimDirectorStatsV0(usage.QuotaStatus)
		summary.UsageTotalTokens = usage.TotalTokens
		summary.UsageCostMicros = usage.CostMicros
	}
	return summary
}

func directorStatsAttentionStateV0(vm WebDirectorStatsViewModelV0) string {
	if vm.Progress.StalledAgents > 0 ||
		vm.Progress.LoopDetectedAgents > 0 ||
		vm.Counts.AgentsNeedAttention > 0 ||
		vm.Checkpoint.RequiresAttention {
		return WebDirectorStatsEstadoAtencionV0
	}
	return WebDirectorStatsEstadoOKV0
}

func directorStatsEstadoV0(estado string, stats *WebDirectorRunStatsContractV0) string {
	switch trimDirectorStatsV0(estado) {
	case WebDirectorStatsInboundEstadoErrorV0:
		return WebDirectorStatsEstadoErrorV0
	case WebDirectorStatsInboundEstadoOKV0:
		return WebDirectorStatsEstadoOKV0
	default:
		if stats != nil {
			return WebDirectorStatsEstadoOKV0
		}
		return WebDirectorStatsEstadoErrorV0
	}
}

func runRefFromStatsV0(stats *WebDirectorRunStatsContractV0) string {
	if stats == nil {
		return ""
	}
	return stats.RunRef
}

func trimDirectorStatsV0(value string) string {
	return strings.TrimSpace(value)
}
