package orquestamcp

import "strings"

func operatorFriendlyToolForCommandV0(input MCPOperatorFriendlyQueryV0) string {
	text := strings.ToLower(strings.TrimSpace(firstNonEmptyMCPV0(input.CommandText, input.Question)))
	switch {
	case strings.Contains(text, "agente"):
		return MCPOperatorFriendlyAgentsToolNameV0
	case strings.Contains(text, "proyecto"):
		return MCPOperatorFriendlyProjectsToolNameV0
	case strings.Contains(text, "tarea") || strings.Contains(text, "cola"):
		return MCPOperatorFriendlyTasksToolNameV0
	default:
		return MCPOperatorFriendlyStatusToolNameV0
	}
}

func filterMCPFriendlyTasksV0(
	tasks []MCPRunQueueRankedCandidateCompactV0,
	input MCPOperatorFriendlyQueryV0,
) []MCPRunQueueRankedCandidateCompactV0 {
	return filterMCPFriendlyTasksByProjectAndStatusV0(tasks, input, true)
}

func filterMCPFriendlyTasksByProjectV0(
	tasks []MCPRunQueueRankedCandidateCompactV0,
	input MCPOperatorFriendlyQueryV0,
) []MCPRunQueueRankedCandidateCompactV0 {
	return filterMCPFriendlyTasksByProjectAndStatusV0(tasks, input, false)
}

func filterMCPFriendlyTasksByProjectAndStatusV0(
	tasks []MCPRunQueueRankedCandidateCompactV0,
	input MCPOperatorFriendlyQueryV0,
	includeStatus bool,
) []MCPRunQueueRankedCandidateCompactV0 {
	projectRef := strings.ToLower(strings.TrimSpace(input.ProjectRef))
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if projectRef == "" && (!includeStatus || status == "") {
		return tasks
	}
	out := make([]MCPRunQueueRankedCandidateCompactV0, 0, len(tasks))
	for _, task := range tasks {
		if projectRef != "" && !strings.Contains(strings.ToLower(task.AppRef), projectRef) {
			continue
		}
		if includeStatus && status != "" && !strings.Contains(strings.ToLower(task.Status), status) {
			continue
		}
		out = append(out, task)
	}
	return out
}

func projectMCPFriendlyLiveTaskStatusesV0(
	tasks []MCPRunQueueRankedCandidateCompactV0,
	runs []MCPDirectorStatsToolResultV0,
) []MCPRunQueueRankedCandidateCompactV0 {
	liveRuns := map[string]struct{}{}
	for _, run := range runs {
		if mcpFriendlyRunHasLiveAgentV0(run) {
			if runRef := mcpFriendlyRunRefV0(run); runRef != "" {
				liveRuns[runRef] = struct{}{}
			}
		}
	}
	out := append([]MCPRunQueueRankedCandidateCompactV0(nil), tasks...)
	for index := range out {
		if _, ok := liveRuns[strings.TrimSpace(out[index].RunRef)]; ok {
			out[index].Status = "running"
		}
	}
	return out
}

func mcpFriendlyRunRefV0(run MCPDirectorStatsToolResultV0) string {
	if runRef := strings.TrimSpace(run.RunRef); runRef != "" {
		return runRef
	}
	if run.Stats == nil {
		return ""
	}
	return strings.TrimSpace(run.Stats.RunRef)
}

func mcpFriendlyRunHasLiveAgentV0(run MCPDirectorStatsToolResultV0) bool {
	if run.Stats == nil {
		return false
	}
	if run.Stats.Counts.AgentsInFlight > 0 {
		return true
	}
	for _, agent := range run.Stats.Agents {
		if agent.InFlight {
			return true
		}
	}
	return false
}

func projectRefsByMCPFriendlyRunV0(
	tasks []MCPRunQueueRankedCandidateCompactV0,
) map[string]string {
	refs := map[string]string{}
	for _, task := range tasks {
		runRef := strings.TrimSpace(task.RunRef)
		if runRef != "" {
			refs[runRef] = strings.TrimSpace(task.AppRef)
		}
	}
	return refs
}

func projectsFromMCPFriendlyTasksV0(tasks []MCPRunQueueRankedCandidateCompactV0) []string {
	values := make([]string, 0, len(tasks))
	seen := map[string]struct{}{}
	for _, task := range tasks {
		appRef := strings.TrimSpace(task.AppRef)
		if appRef == "" {
			continue
		}
		if _, ok := seen[appRef]; ok {
			continue
		}
		seen[appRef] = struct{}{}
		values = append(values, appRef)
	}
	return values
}

func runRefsFromMCPFriendlyTasksV0(tasks []MCPRunQueueRankedCandidateCompactV0) []string {
	refs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, task.RunRef)
	}
	return compactStringsMCPV0(refs)
}

func statusCountsFromMCPFriendlyTasksV0(tasks []MCPRunQueueRankedCandidateCompactV0) map[string]int {
	counts := map[string]int{}
	for _, task := range tasks {
		status := strings.TrimSpace(task.Status)
		if status == "" {
			status = "unknown"
		}
		counts[status]++
	}
	return counts
}

func mcpOperatorFriendlyNextActionsV0(
	toolName string,
	status MCPOperatorFriendlyStatusResultV0,
) []string {
	if len(status.Tasks) == 0 {
		return []string{"queue_empty_or_not_visible", "ask_orquesta_to_start_autonomy_review_if_needed"}
	}
	switch toolName {
	case MCPOperatorFriendlyAgentsToolNameV0:
		if len(status.Agents) == 0 {
			return []string{"agents_not_visible_from_current_runs", "query_status_with_include_run_stats"}
		}
	case MCPOperatorFriendlyTasksToolNameV0:
		return []string{"select_task_by_run_ref_to_prioritize_pause_cancel_or_supervise"}
	}
	return []string{"use_orquesta.tasks.list.v0_for_queue_detail", "use_orquesta.agents.list.v0_for_agent_detail"}
}
