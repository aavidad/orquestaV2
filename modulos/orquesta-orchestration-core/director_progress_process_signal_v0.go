package orquestacionnucleoapp

import "strings"

func applyDirectorRegisteredProcessProgressV0(stats *DirectorRunStatsV0) {
	if stats == nil || len(stats.Progress.Tasks) == 0 {
		return
	}
	liveAgents := directorRegisteredProcessLiveAgentsV0(stats.Agents)
	if len(liveAgents) == 0 {
		return
	}
	taskIndexByAgent := directorProgressTaskIndexByAgentV0(stats.Progress.Tasks)
	changed := false
	for agentRef := range liveAgents {
		index, ok := taskIndexByAgent[agentRef]
		if !ok {
			index, ok = solePendingDirectorTaskProgressIndexV0(stats.Progress.Tasks, liveAgents)
		}
		if !ok || stats.Progress.Tasks[index].Status != DirectorTaskProgressPendingV0 {
			continue
		}
		stats.Progress.Tasks[index].Status = DirectorTaskProgressInProgressV0
		stats.Progress.Tasks[index].AgentRequestID = agentRef
		stats.Progress.Tasks[index].ProgressStatus = DirectorTaskProgressProcessRegisteredV0
		changed = true
	}
	if !changed {
		return
	}
	stats.Progress.TasksObserved = countObservedDirectorTasksV0(stats.Progress.Tasks)
	if stats.Progress.PercentComplete == 0 && stats.Progress.TasksTotal > 0 {
		stats.Progress.PercentComplete = 1
	}
}

func directorRegisteredProcessLiveAgentsV0(
	agents []DirectorAgentStatsV0,
) map[string]bool {
	live := map[string]bool{}
	for _, agent := range agents {
		agentRef := strings.TrimSpace(agent.AgentRequestID)
		if agentRef == "" || !agent.ControlRegistered ||
			!directorRegisteredProcessAgentIsLiveV0(agent) {
			continue
		}
		live[agentRef] = true
	}
	return live
}

func directorRegisteredProcessAgentIsLiveV0(agent DirectorAgentStatsV0) bool {
	switch strings.TrimSpace(agent.Status) {
	case DirectorAgentStatusCompletedV0,
		DirectorAgentStatusFailedV0,
		DirectorAgentStatusLostV0,
		DirectorAgentStatusStoppedV0:
		return false
	}
	if agent.Process != nil &&
		strings.TrimSpace(agent.Process.Status) == DirectorAgentProcessStatusStoppedV0 {
		return false
	}
	return true
}

func directorProgressTaskIndexByAgentV0(
	tasks []DirectorTaskProgressV0,
) map[string]int {
	indexByAgent := map[string]int{}
	for index, task := range tasks {
		if agentRef := strings.TrimSpace(task.AgentRequestID); agentRef != "" {
			indexByAgent[agentRef] = index
		}
		if taskRef := strings.TrimSpace(task.TaskRef); taskRef != "" {
			indexByAgent[WorkflowTaskAgentRequestRefV0(taskRef)] = index
		}
	}
	return indexByAgent
}

func solePendingDirectorTaskProgressIndexV0(
	tasks []DirectorTaskProgressV0,
	liveAgents map[string]bool,
) (int, bool) {
	if len(liveAgents) != 1 {
		return 0, false
	}
	pendingIndex := -1
	for index, task := range tasks {
		if task.Status != DirectorTaskProgressPendingV0 {
			continue
		}
		if pendingIndex >= 0 {
			return 0, false
		}
		pendingIndex = index
	}
	return pendingIndex, pendingIndex >= 0
}
