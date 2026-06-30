package orquestamcp

import (
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

const (
	mcpAutoprogrammingHealthQueuedV0                    = "queued"
	mcpAutoprogrammingHealthRunningLiveV0               = "running_live"
	mcpAutoprogrammingHealthGoalClosurePendingV0        = "goal_closure_pending"
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
	goalStatesByRunRef map[string]orquestagoal.GoalWorkStateV0,
	goalRunMarkersByRunRef map[string]orquestagoal.GoalWorkRunMarkerV0,
	run *MCPDirectorStatsToolResultV0,
	observedRuns ...*MCPDirectorStatsToolResultV0,
) *MCPAutoprogrammingQueueHealthV0 {
	if queue == nil &&
		(run == nil || run.Stats == nil) &&
		len(goalStatesByRunRef) == 0 &&
		len(goalRunMarkersByRunRef) == 0 {
		return nil
	}
	health := MCPAutoprogrammingQueueHealthV0{}
	seen := map[string]bool{}
	statsSeen := map[string]bool{}
	if queue != nil {
		health.QueueRuns = len(queue.Ranked)
		for _, item := range queue.Ranked {
			runRef := strings.TrimSpace(item.RunRef)
			seen[runRef] = true
			if state, ok := goalStatesByRunRef[runRef]; ok {
				applyMCPAutoprogrammingGoalHealthV0(&health, state)
				continue
			}
			if marker, ok := goalRunMarkersByRunRef[runRef]; ok {
				applyMCPAutoprogrammingGoalMarkerHealthV0(&health, marker)
				continue
			}
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
				classifyMCPAutoprogrammingQueuedRunHealthV0(item, goalStatesByRunRef, goalRunMarkersByRunRef),
				-1,
			)
		}
		if state, ok := goalStatesByRunRef[runRef]; ok {
			applyMCPAutoprogrammingGoalHealthV0(&health, state)
		} else if marker, ok := goalRunMarkersByRunRef[runRef]; ok {
			applyMCPAutoprogrammingGoalMarkerHealthV0(&health, marker)
		} else {
			health.AgentsLive += liveAgentsMCPAutoprogrammingRunStatsV0(*run.Stats)
			applyMCPAutoprogrammingRunHealthV0(&health, *run.Stats)
		}
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
				classifyMCPAutoprogrammingQueuedRunHealthV0(item, goalStatesByRunRef, goalRunMarkersByRunRef),
				-1,
			)
		}
		if state, ok := goalStatesByRunRef[runRef]; ok {
			applyMCPAutoprogrammingGoalHealthV0(&health, state)
			continue
		}
		if marker, ok := goalRunMarkersByRunRef[runRef]; ok {
			applyMCPAutoprogrammingGoalMarkerHealthV0(&health, marker)
			continue
		}
		health.AgentsLive += liveAgentsMCPAutoprogrammingRunStatsV0(*observed.Stats)
		applyMCPAutoprogrammingRunHealthV0(&health, *observed.Stats)
	}
	health.QueuedNotDispatched = countMCPAutoprogrammingQueuedNotDispatchedV0(
		queue,
		mcpAutoprogrammingMergeGoalFirstStructSetsV0(
			mcpAutoprogrammingGoalRunRefSetV0(goalStatesByRunRef),
			mcpAutoprogrammingGoalRunMarkerRefSetV0(goalRunMarkersByRunRef),
		),
		append([]*MCPDirectorStatsToolResultV0{run}, observedRuns...)...,
	)
	for runRef, state := range goalStatesByRunRef {
		runRef = strings.TrimSpace(runRef)
		if runRef == "" || seen[runRef] {
			continue
		}
		seen[runRef] = true
		applyMCPAutoprogrammingGoalHealthV0(&health, state)
	}
	for runRef, marker := range goalRunMarkersByRunRef {
		runRef = strings.TrimSpace(runRef)
		if runRef == "" || seen[runRef] {
			continue
		}
		seen[runRef] = true
		applyMCPAutoprogrammingGoalMarkerHealthV0(&health, marker)
	}
	health.StatsRuns = countMCPAutoprogrammingSeenRunsV0(statsSeen)
	health.ObservedRuns = countMCPAutoprogrammingSeenRunsV0(seen)
	return &health
}

func classifyMCPAutoprogrammingQueuedRunHealthV0(
	item MCPRunQueueRankedCandidateCompactV0,
	goalStatesByRunRef map[string]orquestagoal.GoalWorkStateV0,
	goalRunMarkersByRunRef map[string]orquestagoal.GoalWorkRunMarkerV0,
) string {
	runRef := strings.TrimSpace(item.RunRef)
	if state, ok := goalStatesByRunRef[runRef]; ok {
		return classifyMCPAutoprogrammingGoalStateV0(state)
	}
	if _, ok := goalRunMarkersByRunRef[runRef]; ok {
		return mcpAutoprogrammingHealthBlockedV0
	}
	return classifyMCPAutoprogrammingQueueStatusV0(item.Status)
}

func classifyMCPAutoprogrammingQueueStatusV0(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "ready", "queued", "pending":
		return mcpAutoprogrammingHealthQueuedV0
	case "running":
		return mcpAutoprogrammingHealthRunningWithoutRecentStatsV0
	case "paused", "stopped", "stop_requested", "cancel_requested", "blocked", "needs_replan":
		return mcpAutoprogrammingHealthBlockedV0
	case "accepted", "delivered", "closed", "completed":
		return mcpAutoprogrammingHealthCompletedV0
	case "lost":
		return mcpAutoprogrammingHealthLostV0
	case "failed", "canceled", "cancelled":
		return mcpAutoprogrammingHealthFailedV0
	default:
		return mcpAutoprogrammingHealthUnclassifiedV0
	}
}

func applyMCPAutoprogrammingGoalHealthV0(
	health *MCPAutoprogrammingQueueHealthV0,
	state orquestagoal.GoalWorkStateV0,
) {
	applyMCPAutoprogrammingHealthClassV0(
		health,
		classifyMCPAutoprogrammingGoalStateV0(state),
		1,
	)
}

func applyMCPAutoprogrammingGoalMarkerHealthV0(
	health *MCPAutoprogrammingQueueHealthV0,
	_ orquestagoal.GoalWorkRunMarkerV0,
) {
	applyMCPAutoprogrammingHealthClassV0(
		health,
		mcpAutoprogrammingHealthBlockedV0,
		1,
	)
}

func mcpAutoprogrammingGoalRunMarkerRefSetV0(
	markers map[string]orquestagoal.GoalWorkRunMarkerV0,
) map[string]struct{} {
	out := map[string]struct{}{}
	for runRef := range markers {
		runRef = strings.TrimSpace(runRef)
		if runRef != "" {
			out[runRef] = struct{}{}
		}
	}
	return out
}

func mcpAutoprogrammingMergeGoalFirstStructSetsV0(
	sets ...map[string]struct{},
) map[string]struct{} {
	out := map[string]struct{}{}
	for _, set := range sets {
		for runRef := range set {
			runRef = strings.TrimSpace(runRef)
			if runRef != "" {
				out[runRef] = struct{}{}
			}
		}
	}
	return out
}

func classifyMCPAutoprogrammingGoalStateV0(
	state orquestagoal.GoalWorkStateV0,
) string {
	status := strings.ToLower(strings.TrimSpace(state.Status))
	if state.LastClosure != nil {
		closureStatus := strings.ToLower(strings.TrimSpace(state.LastClosure.Status))
		if state.LastClosure.Accepted || closureStatus == orquestagoal.GoalStatusAcceptedV0 {
			return mcpAutoprogrammingHealthCompletedV0
		}
		if state.LastClosure.NeedsRework || closureStatus == orquestagoal.GoalStatusBlockedV0 {
			return mcpAutoprogrammingHealthBlockedV0
		}
	}
	switch status {
	case orquestagoal.GoalStatusBlockedV0, orquestagoal.GoalStatusInvalidV0:
		return mcpAutoprogrammingHealthBlockedV0
	case orquestagoal.GoalStatusCompleteV0:
		return mcpAutoprogrammingHealthGoalClosurePendingV0
	case orquestagoal.GoalStatusRunningV0,
		orquestagoal.GoalStatusAcceptedV0:
		return mcpAutoprogrammingHealthRunningLiveV0
	default:
		return mcpAutoprogrammingHealthRunningLiveV0
	}
}

func applyMCPAutoprogrammingRunHealthV0(
	health *MCPAutoprogrammingQueueHealthV0,
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) {
	liveness := mcpAutoprogrammingRunLivenessV0(stats)
	if stats.Counts.AgentsFailed > 0 {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthFailedV0, 1)
	}
	if stats.Counts.AgentsLost > 0 {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthLostV0, 1)
	}
	if stats.Closure.Blocked {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthBlockedV0, 1)
	}
	if liveness.Class == orquestaruncoordinator.RunLivenessClassRunningLiveV0 {
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthRunningLiveV0, 1)
		return
	}
	if stats.Counts.AgentsInFlight > 0 {
		if liveness.Class == orquestaruncoordinator.RunLivenessClassRunningWithoutRecentStatsV0 {
			applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthRunningWithoutRecentStatsV0, 1)
			return
		}
		if liveness.Class == orquestaruncoordinator.RunLivenessClassRunningStaleNoProcessV0 {
			applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthRunningStaleNoProcessV0, 1)
			return
		}
		applyMCPAutoprogrammingHealthClassV0(health, mcpAutoprogrammingHealthRunningWithoutRecentStatsV0, 1)
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
	return mcpAutoprogrammingRunLivenessV0(stats).Class == orquestaruncoordinator.RunLivenessClassRunningLiveV0
}

func liveAgentsMCPAutoprogrammingRunStatsV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) int {
	return mcpAutoprogrammingRunLivenessV0(stats).AgentsLive
}

func mcpAutoprogrammingRunStatusCompletedV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "closed", "cerrada", "completed", "complete", "delivered":
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
	case mcpAutoprogrammingHealthGoalClosurePendingV0:
		health.GoalClosurePending = nonNegativeMCPAutoprogrammingHealthV0(health.GoalClosurePending + delta)
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
