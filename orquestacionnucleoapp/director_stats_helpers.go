package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func buildDirectorPhaseStatsV0(
	phases []orquestacoreworkflow.OrchestrationPhaseV0,
) []DirectorPhaseStatsV0 {
	if len(phases) == 0 {
		return nil
	}
	stats := make([]DirectorPhaseStatsV0, 0, len(phases))
	for _, phase := range phases {
		stats = append(stats, DirectorPhaseStatsV0{
			PhaseID:             strings.TrimSpace(string(phase.ID)),
			Status:              strings.TrimSpace(string(phase.Status)),
			RecommendedCapacity: strings.TrimSpace(string(phase.RecommendedCapacity)),
			EntryCriteria:       len(compactStringsV0(phase.EntryCriteria)),
			ExitCriteria:        len(compactStringsV0(phase.ExitCriteria)),
			EvidenceRequired:    len(compactStringsV0(phase.EvidenceRequired)),
		})
	}
	return stats
}

func buildDirectorAgentStatsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []DirectorAgentStatsV0 {
	agentRefs := compactStringsV0(append(
		append(append(append([]string{}, run.Agents...), run.StartedAgents...),
			run.FailedAgents...),
		append(run.StoppedAgents, run.ConfirmedStoppedAgents...)...,
	))
	if len(agentRefs) == 0 {
		return nil
	}
	started := autonomousStringSetV0(run.StartedAgents)
	failed := autonomousStringSetV0(run.FailedAgents)
	stopped := autonomousStringSetV0(run.StoppedAgents)
	confirmed := autonomousStringSetV0(run.ConfirmedStoppedAgents)
	stats := make([]DirectorAgentStatsV0, 0, len(agentRefs))
	for _, ref := range agentRefs {
		agent := DirectorAgentStatsV0{
			AgentRequestID: strings.TrimSpace(ref),
			Requested:      true,
			Started:        started[ref],
			Failed:         failed[ref],
			StopRequested:  stopped[ref],
			StopConfirmed:  confirmed[ref],
		}
		agent.InFlight = agent.Started && !agent.Failed && !agent.StopConfirmed
		agent.Status = directorAgentStatusV0(agent)
		agent.NeedsAttention = directorAgentNeedsAttentionV0(agent)
		agent.ControlState = DirectorAgentControlStateNotLoadedV0
		stats = append(stats, agent)
	}
	return stats
}

func directorAgentStatusV0(agent DirectorAgentStatsV0) string {
	switch {
	case agent.Failed:
		return DirectorAgentStatusFailedV0
	case agent.StopConfirmed:
		return DirectorAgentStatusStoppedV0
	case agent.StopRequested:
		return DirectorAgentStatusStopRequestedV0
	case agent.Started:
		return DirectorAgentStatusRunningV0
	default:
		return DirectorAgentStatusRequestedV0
	}
}

func directorAgentNeedsAttentionV0(agent DirectorAgentStatsV0) bool {
	return agent.Failed || (agent.StopRequested && !agent.StopConfirmed)
}

func directorProcessStatsFromRecordV0(
	record AgentProcessRegistryRecordV0,
) DirectorAgentProcessStatsV0 {
	return DirectorAgentProcessStatsV0{
		ProcessRef:   strings.TrimSpace(record.ProcessRef),
		SessionRef:   strings.TrimSpace(record.SessionRef),
		LaunchRef:    strings.TrimSpace(record.LaunchRef),
		ReadinessRef: strings.TrimSpace(record.ReadinessRef),
		EvidenceRefs: compactStringsV0(record.EvidenceRefs),
	}
}

func refreshDirectorControlCountsV0(stats *DirectorRunStatsV0, registryLoaded bool) {
	if stats == nil {
		return
	}
	registered := 0
	missing := 0
	inFlight := 0
	for index := range stats.Agents {
		if stats.Agents[index].InFlight {
			inFlight++
			if stats.Agents[index].ControlRegistered {
				registered++
				stats.Agents[index].ControlState = DirectorAgentControlStateRegisteredV0
				stats.Agents[index].CanStop = true
			} else if registryLoaded {
				stats.Agents[index].ControlState = DirectorAgentControlStateMissingV0
				stats.Agents[index].CanStop = false
				stats.Agents[index].NeedsAttention = true
				missing++
			} else {
				stats.Agents[index].ControlState = DirectorAgentControlStateNotLoadedV0
				stats.Agents[index].CanStop = false
			}
			continue
		}
		stats.Agents[index].ControlState = DirectorAgentControlStateNotNeededV0
		stats.Agents[index].CanStop = false
	}
	stats.Counts.AgentsInFlight = inFlight
	stats.Counts.AgentsControlRegistered = registered
	stats.Counts.AgentsControlMissing = missing
}

func countOpenDirectorTasksV0(tasks []string, closed []string) int {
	closedSet := autonomousStringSetV0(closed)
	open := 0
	for _, task := range tasks {
		if !closedSet[strings.TrimSpace(task)] {
			open++
		}
	}
	return open
}

func openDirectorTaskRefsV0(tasks []string, closed []string) []string {
	closedSet := autonomousStringSetV0(closed)
	open := make([]string, 0, len(tasks))
	for _, task := range compactStringsV0(tasks) {
		if !closedSet[strings.TrimSpace(task)] {
			open = append(open, strings.TrimSpace(task))
		}
	}
	return open
}
