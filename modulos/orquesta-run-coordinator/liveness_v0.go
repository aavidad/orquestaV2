package orquestaruncoordinator

import "strings"

const (
	RunLivenessClassRunningLiveV0               = "running_live"
	RunLivenessClassRunningStaleNoProcessV0     = "running_stale_no_process"
	RunLivenessClassRunningWithoutRecentStatsV0 = "running_without_recent_stats"
	RunLivenessClassBlockedV0                   = "blocked"
	RunLivenessClassLostV0                      = "lost"
	RunLivenessClassCompletedV0                 = "completed"
	RunLivenessClassFailedV0                    = "failed"
	RunLivenessClassUnclassifiedV0              = "unclassified"
)

type RunLivenessInputV0 struct {
	RunStatus         string               `json:"run_status,omitempty"`
	AgentsInFlight    int                  `json:"agents_in_flight,omitempty"`
	AgentsFailed      int                  `json:"agents_failed,omitempty"`
	AgentsLost        int                  `json:"agents_lost,omitempty"`
	PendingAckCount   int                  `json:"pending_ack_count,omitempty"`
	ClosureBlocked    bool                 `json:"closure_blocked,omitempty"`
	ClosureClosed     bool                 `json:"closure_closed,omitempty"`
	LogicalWorkActive bool                 `json:"logical_work_active,omitempty"`
	ProgressingAgents int                  `json:"progressing_agents,omitempty"`
	Agents            []RunLivenessAgentV0 `json:"agents,omitempty"`
}

type RunLivenessAgentV0 struct {
	Status             string `json:"status,omitempty"`
	InFlight           bool   `json:"in_flight,omitempty"`
	NeedsAttention     bool   `json:"needs_attention,omitempty"`
	Completed          bool   `json:"completed,omitempty"`
	Failed             bool   `json:"failed,omitempty"`
	Lost               bool   `json:"lost,omitempty"`
	StopConfirmed      bool   `json:"stop_confirmed,omitempty"`
	LastProgressStatus string `json:"last_progress_status,omitempty"`
	ProcessRef         string `json:"process_ref,omitempty"`
	SessionRef         string `json:"session_ref,omitempty"`
	LaunchRef          string `json:"launch_ref,omitempty"`
	ReadinessRef       string `json:"readiness_ref,omitempty"`
	ProcessStatus      string `json:"process_status,omitempty"`
}

type RunLivenessClassificationV0 struct {
	Class                  string   `json:"class"`
	Reason                 string   `json:"reason,omitempty"`
	RecommendedAction      string   `json:"recommended_action,omitempty"`
	QueueStatusSuggestion  string   `json:"queue_status_suggestion,omitempty"`
	AgentsLive             int      `json:"agents_live,omitempty"`
	ProcessObserved        bool     `json:"process_observed,omitempty"`
	UnknownProcess         bool     `json:"unknown_process,omitempty"`
	ConfirmedNoLiveProcess bool     `json:"confirmed_no_live_process,omitempty"`
	Running                bool     `json:"running,omitempty"`
	Live                   bool     `json:"live,omitempty"`
	Stale                  bool     `json:"stale,omitempty"`
	Ambiguous              bool     `json:"ambiguous,omitempty"`
	Verifiable             bool     `json:"verifiable,omitempty"`
	SafeToReconcile        bool     `json:"safe_to_reconcile,omitempty"`
	ProcessRefs            []string `json:"process_refs,omitempty"`
}

func ClassifyRunLivenessV0(input RunLivenessInputV0) RunLivenessClassificationV0 {
	result := RunLivenessClassificationV0{
		Class:           RunLivenessClassUnclassifiedV0,
		ProcessObserved: runLivenessHasProcessObservedV0(input.Agents),
		ProcessRefs:     compactRunLivenessStringsV0(runLivenessProcessRefsV0(input.Agents)),
	}
	result.AgentsLive = runLivenessLiveAgentsV0(input)
	result.UnknownProcess = runLivenessHasUnknownProcessSignalV0(input.Agents)
	result.ConfirmedNoLiveProcess = runLivenessHasConfirmedNoLiveProcessV0(input.Agents)
	result.Running = input.LogicalWorkActive ||
		input.AgentsInFlight > 0 ||
		result.AgentsLive > 0 ||
		result.UnknownProcess ||
		result.ConfirmedNoLiveProcess
	if input.LogicalWorkActive {
		result.Class = RunLivenessClassRunningLiveV0
		result.Reason = "logical_work_active"
		result.RecommendedAction = "observe_logical_work"
		result.Live = true
		result.Verifiable = true
		result.Running = true
		return result
	}
	if result.AgentsLive > 0 {
		result.Class = RunLivenessClassRunningLiveV0
		result.Reason = "live_process_or_progress_detected"
		result.RecommendedAction = "wait_for_live_process_or_progress"
		result.Live = true
		result.Verifiable = true
		return result
	}
	if input.AgentsInFlight > 0 {
		if result.UnknownProcess {
			result.Class = RunLivenessClassRunningWithoutRecentStatsV0
			result.Reason = "running_requires_liveness_confirmation"
			result.RecommendedAction = "observe_run_ref_with_process_refs_before_reconcile"
			result.Ambiguous = true
			return result
		}
		if result.ConfirmedNoLiveProcess {
			result.Class = RunLivenessClassRunningStaleNoProcessV0
			result.Reason = "running_stale_no_live_process_detected"
			result.RecommendedAction = "reconcile_if_no_live_process_or_wait_for_late_ack"
			result.QueueStatusSuggestion = "stopped"
			result.Stale = true
			result.Verifiable = true
			result.SafeToReconcile = input.PendingAckCount == 0
			if input.PendingAckCount > 0 {
				result.Reason = "running_stale_no_live_process_pending_ack"
				result.RecommendedAction = "wait_for_ack_before_reconcile"
				result.QueueStatusSuggestion = ""
			}
			return result
		}
		result.Class = RunLivenessClassRunningWithoutRecentStatsV0
		result.Reason = "running_requires_liveness_confirmation"
		result.RecommendedAction = "observe_run_ref_with_process_refs_before_reconcile"
		result.Ambiguous = true
		return result
	}
	if input.AgentsFailed > 0 {
		result.Class = RunLivenessClassFailedV0
		result.Reason = "agents_failed"
		result.Verifiable = true
		return result
	}
	if input.AgentsLost > 0 {
		result.Class = RunLivenessClassLostV0
		result.Reason = "agents_lost"
		result.Verifiable = true
		return result
	}
	if input.ClosureBlocked {
		result.Class = RunLivenessClassBlockedV0
		result.Reason = "closure_blocked"
		result.Verifiable = true
		return result
	}
	if input.ClosureClosed || runLivenessCompletedStatusV0(input.RunStatus) {
		result.Class = RunLivenessClassCompletedV0
		result.Reason = "run_completed"
		result.Verifiable = true
	}
	return result
}

func runLivenessLiveAgentsV0(input RunLivenessInputV0) int {
	liveAgents := 0
	for _, agent := range input.Agents {
		if runLivenessLiveAgentSignalV0(agent) || runLivenessLiveProcessSignalV0(agent) {
			liveAgents++
		}
	}
	if input.ProgressingAgents > liveAgents {
		liveAgents = input.ProgressingAgents
	}
	return liveAgents
}

func runLivenessLiveAgentSignalV0(agent RunLivenessAgentV0) bool {
	if !agent.InFlight || agent.NeedsAttention || runLivenessAgentTerminalV0(agent) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(agent.LastProgressStatus)) {
	case "progressing", "working", "running":
		return true
	default:
		return false
	}
}

func runLivenessLiveProcessSignalV0(agent RunLivenessAgentV0) bool {
	if agent.NeedsAttention || runLivenessAgentTerminalV0(agent) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(agent.ProcessStatus)) {
	case "running", "stopping":
		return true
	default:
		return false
	}
}

func runLivenessHasUnknownProcessSignalV0(agents []RunLivenessAgentV0) bool {
	for _, agent := range agents {
		if runLivenessAgentTerminalV0(agent) || !runLivenessProcessObservedV0(agent) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(agent.ProcessStatus)) {
		case "", "unknown":
			return true
		}
	}
	return false
}

func runLivenessHasConfirmedNoLiveProcessV0(agents []RunLivenessAgentV0) bool {
	confirmedStopped := false
	for _, agent := range agents {
		if !agent.InFlight || runLivenessAgentTerminalV0(agent) {
			continue
		}
		if !runLivenessProcessObservedV0(agent) {
			return false
		}
		switch strings.ToLower(strings.TrimSpace(agent.ProcessStatus)) {
		case "stopped":
			confirmedStopped = true
		case "", "running", "stopping", "unknown":
			return false
		default:
			return false
		}
	}
	return confirmedStopped
}

func runLivenessHasProcessObservedV0(agents []RunLivenessAgentV0) bool {
	for _, agent := range agents {
		if runLivenessProcessObservedV0(agent) {
			return true
		}
	}
	return false
}

func runLivenessProcessObservedV0(agent RunLivenessAgentV0) bool {
	return strings.TrimSpace(agent.ProcessRef) != "" ||
		strings.TrimSpace(agent.SessionRef) != "" ||
		strings.TrimSpace(agent.LaunchRef) != "" ||
		strings.TrimSpace(agent.ReadinessRef) != "" ||
		strings.TrimSpace(agent.ProcessStatus) != ""
}

func runLivenessProcessRefsV0(agents []RunLivenessAgentV0) []string {
	out := []string{}
	for _, agent := range agents {
		out = append(out, agent.ProcessRef, agent.SessionRef, agent.LaunchRef, agent.ReadinessRef)
	}
	return out
}

func runLivenessAgentTerminalV0(agent RunLivenessAgentV0) bool {
	if agent.Failed || agent.Lost || agent.Completed || agent.StopConfirmed {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(agent.Status)) {
	case "completed", "failed", "lost", "stopped":
		return true
	default:
		return false
	}
}

func runLivenessCompletedStatusV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "closed", "cerrada", "completed", "complete", "delivered":
		return true
	default:
		return false
	}
}

func compactRunLivenessStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if out == nil {
		return nil
	}
	return out
}
