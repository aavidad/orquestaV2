package orquestamcp

import (
	"strconv"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const MCPAutoprogrammingEfficiencySummarySchemaVersionV0 = "autoprogramming_efficiency_summary.v0"

type MCPAutoprogrammingEfficiencySummaryV0 struct {
	SchemaVersion               string   `json:"schema_version"`
	State                       string   `json:"state"`
	OverallPercentage           int      `json:"overall_percentage"`
	OperationalHealthPercentage int      `json:"operational_health_percentage"`
	CompletionPercentage        int      `json:"completion_percentage"`
	QueueVisibilityPercentage   int      `json:"queue_visibility_percentage"`
	AlivePercentage             int      `json:"alive_percentage"`
	StuckPercentage             int      `json:"stuck_percentage"`
	AliveStuckSampleSize        int      `json:"alive_stuck_sample_size"`
	AliveStuckStatus            string   `json:"alive_stuck_status"`
	Confidence                  string   `json:"confidence"`
	QueueCandidates             int      `json:"queue_candidates"`
	ActiveRuns                  int      `json:"active_runs"`
	TasksTotal                  int      `json:"tasks_total"`
	TasksClosed                 int      `json:"tasks_closed"`
	TasksOpen                   int      `json:"tasks_open"`
	AgentsTotal                 int      `json:"agents_total"`
	AgentsInFlight              int      `json:"agents_in_flight"`
	AgentsProgressing           int      `json:"agents_progressing"`
	AgentsNeedAttention         int      `json:"agents_need_attention"`
	AgentsStalled               int      `json:"agents_stalled"`
	AgentsStuck                 int      `json:"agents_stuck"`
	ClosureStatus               string   `json:"closure_status,omitempty"`
	ClosureBlocked              bool     `json:"closure_blocked,omitempty"`
	RecommendedAction           string   `json:"recommended_action,omitempty"`
	Reasons                     []string `json:"reasons,omitempty"`
}

func buildMCPAutoprogrammingEfficiencySummaryV0(
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
	operator *MCPAutoprogrammingOperatorV0,
	diagnostics []MCPAutoprogrammingDiagnosticV0,
) *MCPAutoprogrammingEfficiencySummaryV0 {
	summary := &MCPAutoprogrammingEfficiencySummaryV0{
		SchemaVersion:    MCPAutoprogrammingEfficiencySummarySchemaVersionV0,
		State:            "unavailable",
		AliveStuckStatus: "no_data",
		Confidence:       "none",
		Reasons:          []string{},
	}
	queueVisible := queue != nil && queue.Estado == MCPRunQueuePriorityEstadoOKV0
	runVisible := run != nil && run.Estado == MCPDirectorStatsEstadoOKV0 && run.Stats != nil
	if queueVisible {
		summary.QueueVisibilityPercentage = 100
		summary.QueueCandidates = queue.Count
		if summary.QueueCandidates == 0 {
			summary.QueueCandidates = len(queue.Ranked)
		}
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "queue_live")
		if summary.QueueCandidates > 0 {
			summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "queue_has_candidates")
		}
	} else if queue != nil {
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "queue_not_available")
	}
	if operator != nil {
		summary.ActiveRuns = len(operator.ActiveRuns)
		summary.RecommendedAction = recommendedActionMCPAutoprogrammingEfficiencyV0(operator)
	}
	if runVisible {
		fillRunMCPAutoprogrammingEfficiencySummaryV0(summary, run)
	}
	summary.OperationalHealthPercentage = operationalHealthMCPAutoprogrammingEfficiencyV0(
		summary,
		queueVisible,
		runVisible,
		run != nil,
		diagnostics,
	)
	summary.State = stateMCPAutoprogrammingEfficiencyV0(summary, queueVisible, runVisible, run)
	summary.OverallPercentage = overallPercentageMCPAutoprogrammingEfficiencyV0(summary, runVisible)
	summary.Confidence = confidenceMCPAutoprogrammingEfficiencyV0(queueVisible, runVisible, summary)
	summary.Reasons = compactStringsMCPV0(summary.Reasons)
	return summary
}

func fillRunMCPAutoprogrammingEfficiencySummaryV0(
	summary *MCPAutoprogrammingEfficiencySummaryV0,
	run *MCPDirectorStatsToolResultV0,
) {
	stats := run.Stats
	counts := stats.Counts
	progress := stats.Progress
	summary.TasksTotal = firstNonZeroIntMCPV0(progress.TasksTotal, counts.TasksTotal)
	summary.TasksClosed = firstNonZeroIntMCPV0(progress.TasksClosed, counts.TasksClosed)
	summary.TasksOpen = counts.TasksOpen
	if summary.TasksOpen == 0 && summary.TasksTotal > summary.TasksClosed {
		summary.TasksOpen = summary.TasksTotal - summary.TasksClosed
	}
	summary.CompletionPercentage = progress.PercentComplete
	if summary.CompletionPercentage == 0 && summary.TasksTotal > 0 {
		summary.CompletionPercentage = percentageMCPAutoprogrammingEfficiencyV0(summary.TasksClosed, summary.TasksTotal)
	}
	summary.AgentsTotal = maxIntMCPAutoprogrammingEfficiencyV0(
		progress.ObservedAgents,
		len(stats.Agents),
		counts.AgentsStarted,
		counts.AgentsRequested,
	)
	summary.AgentsInFlight = counts.AgentsInFlight
	summary.AgentsProgressing = progress.ProgressingAgents + progress.OverBudgetButActiveAgents
	if summary.AgentsProgressing == 0 {
		summary.AgentsProgressing = healthyOpaqueInFlightAgentsMCPAutoprogrammingEfficiencyV0(stats.Agents)
	}
	summary.AgentsNeedAttention = countAgentsNeedAttentionMCPAutoprogrammingEfficiencyV0(stats.Agents)
	summary.AgentsStalled = progress.StalledAgents
	summary.AgentsStuck = progress.LoopDetectedAgents +
		progress.OverBudgetNoActivityAgents +
		counts.AgentsFailed +
		counts.AgentsLost
	if summary.AgentsTotal == 0 {
		summary.AgentsTotal = maxIntMCPAutoprogrammingEfficiencyV0(
			summary.AgentsInFlight+summary.AgentsStuck,
			summary.AgentsProgressing+summary.AgentsStuck,
		)
	}
	if summary.AgentsTotal > 0 {
		summary.AgentsProgressing = clampIntMCPAutoprogrammingEfficiencyV0(summary.AgentsProgressing, 0, summary.AgentsTotal)
		summary.AgentsStuck = clampIntMCPAutoprogrammingEfficiencyV0(summary.AgentsStuck, 0, summary.AgentsTotal)
		summary.AliveStuckSampleSize = summary.AgentsTotal
		summary.AliveStuckStatus = "observed"
		summary.AlivePercentage = percentageMCPAutoprogrammingEfficiencyV0(summary.AgentsProgressing, summary.AgentsTotal)
		summary.StuckPercentage = percentageMCPAutoprogrammingEfficiencyV0(summary.AgentsStuck, summary.AgentsTotal)
	}
	summary.ClosureStatus = strings.TrimSpace(stats.Closure.Status)
	summary.ClosureBlocked = stats.Closure.Blocked
	if summary.AgentsInFlight > 0 {
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "agents_in_flight="+strconv.Itoa(summary.AgentsInFlight))
	}
	if summary.AgentsStalled > 0 {
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "agents_stalled="+strconv.Itoa(summary.AgentsStalled))
	}
	if summary.AgentsStuck > 0 {
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "agents_stuck="+strconv.Itoa(summary.AgentsStuck))
	}
	if summary.AgentsNeedAttention > 0 {
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "agents_need_attention="+strconv.Itoa(summary.AgentsNeedAttention))
	}
	if summary.ClosureBlocked {
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "closure_blocked="+strings.TrimSpace(stats.Closure.Status))
	}
}

func operationalHealthMCPAutoprogrammingEfficiencyV0(
	summary *MCPAutoprogrammingEfficiencySummaryV0,
	queueVisible bool,
	runVisible bool,
	runRequested bool,
	diagnostics []MCPAutoprogrammingDiagnosticV0,
) int {
	components := []int{}
	if queueVisible {
		components = append(components, 100)
	} else {
		components = append(components, 0)
	}
	if runVisible {
		components = append(components, 100)
		if summary.AgentsTotal > 0 {
			components = append(components, summary.AlivePercentage)
		}
	} else if runRequested {
		components = append(components, 0)
	} else if summary.QueueCandidates > 0 {
		components = append(components, 70)
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "run_stats_not_requested")
	}
	health := averageMCPAutoprogrammingEfficiencyV0(components)
	if health == 0 && queueVisible && summary.QueueCandidates == 0 && !runRequested {
		health = 100
	}
	if summary.StuckPercentage > 0 {
		health = minIntMCPAutoprogrammingEfficiencyV0(health, 100-summary.StuckPercentage)
	}
	if summary.AgentsNeedAttention > 0 && summary.AgentsTotal > 0 {
		health = minIntMCPAutoprogrammingEfficiencyV0(
			health,
			100-percentageMCPAutoprogrammingEfficiencyV0(summary.AgentsNeedAttention, summary.AgentsTotal)/2,
		)
	}
	if summary.ClosureBlocked && summary.AgentsInFlight == 0 {
		health = minIntMCPAutoprogrammingEfficiencyV0(health, 70)
	}
	if hasDiagnosticMCPAutoprogrammingEfficiencyV0(diagnostics, "autoprogramming_goal_first_blocked") {
		health = minIntMCPAutoprogrammingEfficiencyV0(health, 40)
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "goal_first_blocked")
	}
	if hasDiagnosticMCPAutoprogrammingEfficiencyV0(diagnostics, "autoprogramming_goal_first_state_missing") {
		health = minIntMCPAutoprogrammingEfficiencyV0(health, 60)
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "goal_first_state_missing")
	}
	if hasDiagnosticMCPAutoprogrammingEfficiencyV0(diagnostics, "supervisor_replan_amplification_blocked") {
		health = minIntMCPAutoprogrammingEfficiencyV0(health, 30)
		summary.Reasons = appendUniqueMCPAutoprogrammingReasonV0(summary.Reasons, "supervisor_replan_amplification_blocked")
	}
	if hasDiagnosticMCPAutoprogrammingEfficiencyV0(diagnostics, "run_stats_error") ||
		hasDiagnosticMCPAutoprogrammingEfficiencyV0(diagnostics, "run_stats_unbound") {
		health = minIntMCPAutoprogrammingEfficiencyV0(health, 60)
	}
	return clampIntMCPAutoprogrammingEfficiencyV0(health, 0, 100)
}

func overallPercentageMCPAutoprogrammingEfficiencyV0(
	summary *MCPAutoprogrammingEfficiencySummaryV0,
	runVisible bool,
) int {
	if summary == nil {
		return 0
	}
	if !runVisible || summary.TasksTotal <= 0 {
		return summary.OperationalHealthPercentage
	}
	overall := ((summary.CompletionPercentage * 70) + (summary.OperationalHealthPercentage * 30) + 50) / 100
	if !strings.EqualFold(summary.ClosureStatus, "closed") && overall >= 100 {
		overall = 99
	}
	return clampIntMCPAutoprogrammingEfficiencyV0(overall, 0, 100)
}

func stateMCPAutoprogrammingEfficiencyV0(
	summary *MCPAutoprogrammingEfficiencySummaryV0,
	queueVisible bool,
	runVisible bool,
	run *MCPDirectorStatsToolResultV0,
) string {
	switch {
	case !queueVisible && !runVisible:
		return "unavailable"
	case strings.EqualFold(summary.ClosureStatus, "closed"):
		return "closed"
	case hasReasonMCPAutoprogrammingEfficiencyV0(summary, "goal_first_blocked") ||
		hasReasonMCPAutoprogrammingEfficiencyV0(summary, "goal_first_state_missing"):
		return "attention_required"
	case hasRunReplanAmplificationMCPAutoprogrammingEfficiencyV0(run):
		return "hung"
	case summary.StuckPercentage >= 50 || (summary.AgentsStuck > 0 && summary.AlivePercentage == 0):
		return "hung"
	case summary.AgentsStuck > 0 || summary.AgentsNeedAttention > 0:
		return "attention_required"
	case summary.AgentsInFlight > 0 || summary.AgentsProgressing > 0:
		return "live"
	case strings.EqualFold(summary.ClosureStatus, "ready"):
		return "closing"
	case summary.ClosureBlocked:
		return "attention_required"
	case runVisible && summary.TasksOpen > 0:
		return "waiting"
	case summary.QueueCandidates > 0:
		return "queued"
	case queueVisible:
		return "idle"
	default:
		return "unknown"
	}
}

func hasReasonMCPAutoprogrammingEfficiencyV0(
	summary *MCPAutoprogrammingEfficiencySummaryV0,
	reason string,
) bool {
	if summary == nil {
		return false
	}
	reason = strings.TrimSpace(reason)
	for _, candidate := range summary.Reasons {
		if strings.TrimSpace(candidate) == reason {
			return true
		}
	}
	return false
}

func confidenceMCPAutoprogrammingEfficiencyV0(
	queueVisible bool,
	runVisible bool,
	summary *MCPAutoprogrammingEfficiencySummaryV0,
) string {
	switch {
	case queueVisible && runVisible:
		return "high"
	case queueVisible && summary.QueueCandidates == 0:
		return "medium"
	case queueVisible || runVisible:
		return "partial"
	default:
		return "none"
	}
}

func recommendedActionMCPAutoprogrammingEfficiencyV0(
	operator *MCPAutoprogrammingOperatorV0,
) string {
	if operator == nil || len(operator.SafeActions) == 0 {
		return ""
	}
	action := operator.SafeActions[0]
	parts := []string{strings.TrimSpace(action.Action), strings.TrimSpace(action.Scope)}
	if runRef := strings.TrimSpace(action.RunRef); runRef != "" {
		parts = append(parts, runRef)
	}
	return strings.Join(compactStringsMCPV0(parts), ":")
}

func healthyOpaqueInFlightAgentsMCPAutoprogrammingEfficiencyV0(
	agents []orquestacionnucleoapp.DirectorAgentStatsV0,
) int {
	total := 0
	for _, agent := range agents {
		if agent.InFlight && !agent.NeedsAttention && !agent.Failed && !agent.Lost {
			total++
		}
	}
	return total
}

func countAgentsNeedAttentionMCPAutoprogrammingEfficiencyV0(
	agents []orquestacionnucleoapp.DirectorAgentStatsV0,
) int {
	total := 0
	for _, agent := range agents {
		if agent.NeedsAttention {
			total++
		}
	}
	return total
}

func hasRunReplanAmplificationMCPAutoprogrammingEfficiencyV0(
	run *MCPDirectorStatsToolResultV0,
) bool {
	return mcpAutoprogrammingReplanAmplificationBlockedV0(run)
}

func hasDiagnosticMCPAutoprogrammingEfficiencyV0(
	diagnostics []MCPAutoprogrammingDiagnosticV0,
	code string,
) bool {
	code = strings.TrimSpace(code)
	for _, diagnostic := range diagnostics {
		if strings.TrimSpace(diagnostic.Code) == code {
			return true
		}
	}
	return false
}

func appendUniqueMCPAutoprogrammingReasonV0(values []string, next string) []string {
	next = strings.TrimSpace(next)
	if next == "" {
		return values
	}
	for _, value := range values {
		if strings.TrimSpace(value) == next {
			return values
		}
	}
	return append(values, next)
}

func percentageMCPAutoprogrammingEfficiencyV0(part int, total int) int {
	if total <= 0 {
		return 0
	}
	return clampIntMCPAutoprogrammingEfficiencyV0((part*100)+(total/2), 0, total*100) / total
}

func averageMCPAutoprogrammingEfficiencyV0(values []int) int {
	if len(values) == 0 {
		return 0
	}
	total := 0
	for _, value := range values {
		total += clampIntMCPAutoprogrammingEfficiencyV0(value, 0, 100)
	}
	return percentageMCPAutoprogrammingEfficiencyV0(total, len(values)*100)
}

func clampIntMCPAutoprogrammingEfficiencyV0(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxIntMCPAutoprogrammingEfficiencyV0(values ...int) int {
	max := 0
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

func minIntMCPAutoprogrammingEfficiencyV0(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
