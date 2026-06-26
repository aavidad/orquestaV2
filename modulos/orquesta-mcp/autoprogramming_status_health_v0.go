package orquestamcp

import (
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	mcpAutoprogrammingHealthQueuedV0                    = "queued"
	mcpAutoprogrammingHealthRunningLiveV0               = "running_live"
	mcpAutoprogrammingHealthRunningStaleV0              = "running_stale"
	mcpAutoprogrammingHealthRunningStaleNoProcessV0     = "running_stale_no_process"
	mcpAutoprogrammingHealthRunningWithoutRecentStatsV0 = "running_without_recent_stats"
	mcpAutoprogrammingHealthBlockedV0                   = "blocked"
	mcpAutoprogrammingHealthLostV0                      = "lost"
	mcpAutoprogrammingHealthCompletedV0                 = "completed"
	mcpAutoprogrammingHealthFailedV0                    = "failed"
	mcpAutoprogrammingHealthUnclassifiedV0              = "unclassified"
)

func buildMCPAutoprogrammingQueueHealthV0(
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
	observedRuns ...*MCPDirectorStatsToolResultV0,
) *MCPAutoprogrammingQueueHealthV0 {
	if queue == nil && (run == nil || run.Stats == nil) {
		return nil
	}
	health := MCPAutoprogrammingQueueHealthV0{}
	seen := map[string]bool{}
	statsSeen := map[string]bool{}
	if queue != nil {
		health.QueueRuns = len(queue.Ranked)
		for _, item := range queue.Ranked {
			seen[strings.TrimSpace(item.RunRef)] = true
			applyMCPAutoprogrammingHealthClassV0(
				&health,
				classifyMCPAutoprogrammingQueueStatusV0(item.Status),
				1,
			)
		}
	}
	if run != nil && run.Stats != nil {
		runRef := strings.TrimSpace(run.Stats.RunRef)
		if runRef != "" {
			seen[runRef] = true
			statsSeen[runRef] = true
		}
		if item, ok := queueCandidateForRunMCPAutoprogrammingHealthV0(queue, runRef); ok {
			applyMCPAutoprogrammingHealthClassV0(
				&health,
				classifyMCPAutoprogrammingQueueStatusV0(item.Status),
				-1,
			)
		}
		applyMCPAutoprogrammingRunHealthV0(&health, *run.Stats)
	}
	for _, observed := range observedRuns {
		if observed == nil || observed.Stats == nil {
			continue
		}
		runRef := strings.TrimSpace(observed.Stats.RunRef)
		if runRef == "" || statsSeen[runRef] {
			continue
		}
		statsSeen[runRef] = true
		seen[runRef] = true
		if item, ok := queueCandidateForRunMCPAutoprogrammingHealthV0(queue, runRef); ok {
			applyMCPAutoprogrammingHealthClassV0(
				&health,
				classifyMCPAutoprogrammingQueueStatusV0(item.Status),
				-1,
			)
		}
		applyMCPAutoprogrammingRunHealthV0(&health, *observed.Stats)
	}
	health.StatsRuns = countMCPAutoprogrammingSeenRunsV0(statsSeen)
	health.ObservedRuns = countMCPAutoprogrammingSeenRunsV0(seen)
	return &health
}

func classifyMCPAutoprogrammingQueueStatusV0(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "ready", "queued", "pending":
		return mcpAutoprogrammingHealthQueuedV0
	case "running":
		return mcpAutoprogrammingHealthRunningWithoutRecentStatsV0
	case "paused", "stopped", "stop_requested", "cancel_requested", "blocked", "needs_replan":
		return mcpAutoprogrammingHealthBlockedV0
	case "delivered", "closed", "completed":
		return mcpAutoprogrammingHealthCompletedV0
	case "lost":
		return mcpAutoprogrammingHealthLostV0
	case "failed", "canceled", "cancelled":
		return mcpAutoprogrammingHealthFailedV0
	default:
		return mcpAutoprogrammingHealthUnclassifiedV0
	}
}

func applyMCPAutoprogrammingRunHealthV0(
	health *MCPAutoprogrammingQueueHealthV0,
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) {
	if stats.Counts.AgentsFailed > 0 {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthFailedV0, 1)
	}
	if stats.Counts.AgentsLost > 0 {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthLostV0, 1)
	}
	if stats.Closure.Blocked {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthBlockedV0, 1)
	}
	if mcpAutoprogrammingRunStatsHasLiveSignalV0(stats) {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthRunningLiveV0, 1)
		return
	}
	if stats.Counts.AgentsInFlight > 0 {
		stale := stats.Progress.StalledAgents
		if stale > 0 || stats.Progress.ProgressingAgents == 0 {
			applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthRunningStaleNoProcessV0, 1)
		}
		return
	}
	if stats.Closure.Closed || mcpAutoprogrammingRunStatusCompletedV0(stats.Status) {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthCompletedV0, 1)
		return
	}
	if stats.Counts.AgentsFailed == 0 && stats.Counts.AgentsLost == 0 && !stats.Closure.Blocked {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthUnclassifiedV0, 1)
	}
}

func mcpAutoprogrammingRunStatsHasLiveSignalV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) bool {
	return stats.Progress.ProgressingAgents > 0 ||
		hasMCPAutoprogrammingLiveAgentSignalV0(stats.Agents) ||
		hasMCPAutoprogrammingLiveProcessSignalV0(stats.Agents)
}

func hasMCPAutoprogrammingLiveProcessSignalV0(
	agents []orquestacionnucleoapp.DirectorAgentStatsV0,
) bool {
	for _, agent := range agents {
		if agent.NeedsAttention || agent.Process == nil ||
			mcpAutoprogrammingAgentStatsTerminalV0(agent) {
			continue
		}
		if strings.TrimSpace(agent.Process.ProcessRef) != "" ||
			strings.TrimSpace(agent.Process.SessionRef) != "" ||
			strings.TrimSpace(agent.Process.LaunchRef) != "" ||
			strings.TrimSpace(agent.Process.ReadinessRef) != "" {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingAgentStatsTerminalV0(
	agent orquestacionnucleoapp.DirectorAgentStatsV0,
) bool {
	if agent.Failed || agent.Lost || agent.Completed || agent.StopConfirmed {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(agent.Status)) {
	case orquestacionnucleoapp.DirectorAgentStatusCompletedV0,
		orquestacionnucleoapp.DirectorAgentStatusFailedV0,
		orquestacionnucleoapp.DirectorAgentStatusLostV0,
		orquestacionnucleoapp.DirectorAgentStatusStoppedV0:
		return true
	default:
		return false
	}
}

func hasMCPAutoprogrammingLiveAgentSignalV0(
	agents []orquestacionnucleoapp.DirectorAgentStatsV0,
) bool {
	for _, agent := range agents {
		if !agent.InFlight || agent.NeedsAttention {
			continue
		}
		if agent.LastProgress == nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(agent.LastProgress.Status)) {
		case "progressing", "working", "running":
			return true
		}
	}
	return false
}

func mcpAutoprogrammingRunStatusCompletedV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "closed", "cerrada", "completed", "delivered":
		return true
	default:
		return false
	}
}

func queueCandidateForRunMCPAutoprogrammingHealthV0(
	queue *MCPRunQueuePriorityToolResultV0,
	runRef string,
) (MCPRunQueueRankedCandidateCompactV0, bool) {
	if queue == nil {
		return MCPRunQueueRankedCandidateCompactV0{}, false
	}
	runRef = strings.TrimSpace(runRef)
	for _, item := range queue.Ranked {
		if strings.TrimSpace(item.RunRef) == runRef {
			return item, true
		}
	}
	return MCPRunQueueRankedCandidateCompactV0{}, false
}

func applyMCPAutoprogrammingHealthClassV0(
	health *MCPAutoprogrammingQueueHealthV0,
	class string,
	delta int,
) {
	if health == nil || delta == 0 {
		return
	}
	switch class {
	case mcpAutoprogrammingHealthQueuedV0:
		health.Queued = nonNegativeMCPAutoprogrammingHealthV0(health.Queued + delta)
	case mcpAutoprogrammingHealthRunningLiveV0:
		health.RunningLive = nonNegativeMCPAutoprogrammingHealthV0(health.RunningLive + delta)
	case mcpAutoprogrammingHealthRunningStaleV0:
		health.RunningStale = nonNegativeMCPAutoprogrammingHealthV0(health.RunningStale + delta)
	case mcpAutoprogrammingHealthRunningStaleNoProcessV0:
		health.RunningStale = nonNegativeMCPAutoprogrammingHealthV0(health.RunningStale + delta)
		health.RunningStaleNoProcess = nonNegativeMCPAutoprogrammingHealthV0(health.RunningStaleNoProcess + delta)
	case mcpAutoprogrammingHealthRunningWithoutRecentStatsV0:
		health.RunningWithoutRecentStats = nonNegativeMCPAutoprogrammingHealthV0(health.RunningWithoutRecentStats + delta)
	case mcpAutoprogrammingHealthBlockedV0:
		health.Blocked = nonNegativeMCPAutoprogrammingHealthV0(health.Blocked + delta)
	case mcpAutoprogrammingHealthLostV0:
		health.Lost = nonNegativeMCPAutoprogrammingHealthV0(health.Lost + delta)
	case mcpAutoprogrammingHealthCompletedV0:
		health.Completed = nonNegativeMCPAutoprogrammingHealthV0(health.Completed + delta)
	case mcpAutoprogrammingHealthFailedV0:
		health.Failed = nonNegativeMCPAutoprogrammingHealthV0(health.Failed + delta)
	default:
		health.Unclassified = nonNegativeMCPAutoprogrammingHealthV0(health.Unclassified + delta)
	}
}

func nonNegativeMCPAutoprogrammingHealthV0(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func countMCPAutoprogrammingSeenRunsV0(seen map[string]bool) int {
	count := 0
	for ref := range seen {
		if strings.TrimSpace(ref) != "" {
			count++
		}
	}
	return count
}
