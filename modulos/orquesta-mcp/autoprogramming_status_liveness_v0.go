package orquestamcp

import (
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func mcpAutoprogrammingRunLivenessV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) orquestaruncoordinator.RunLivenessClassificationV0 {
	return orquestaruncoordinator.ClassifyRunLivenessV0(
		orquestaruncoordinator.RunLivenessInputV0{
			RunStatus:         stats.Status,
			AgentsInFlight:    stats.Counts.AgentsInFlight,
			AgentsFailed:      stats.Counts.AgentsFailed,
			AgentsLost:        stats.Counts.AgentsLost,
			PendingAckCount:   mcpAutoprogrammingAckCleanupCountV0(stats),
			ClosureBlocked:    stats.Closure.Blocked,
			ClosureClosed:     stats.Closure.Closed,
			ProgressingAgents: stats.Progress.ProgressingAgents,
			Agents:            mcpAutoprogrammingRunLivenessAgentsV0(stats.Agents),
		},
	)
}

func mcpAutoprogrammingRunLivenessAgentsV0(
	agents []orquestacionnucleoapp.DirectorAgentStatsV0,
) []orquestaruncoordinator.RunLivenessAgentV0 {
	out := make([]orquestaruncoordinator.RunLivenessAgentV0, 0, len(agents))
	for _, agent := range agents {
		out = append(out, mcpAutoprogrammingRunLivenessAgentV0(agent))
	}
	return out
}

func mcpAutoprogrammingAckCleanupCountV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) int {
	pendingByAgent := map[string]bool{}
	for _, task := range stats.Progress.Tasks {
		if !mcpAutoprogrammingProgressIsAckCleanupV0(task.DirectorProgressTemporalV0) {
			continue
		}
		agentRef := task.AgentRequestID
		if agentRef == "" {
			agentRef = task.TaskRef
		}
		if agentRef != "" {
			pendingByAgent[agentRef] = true
		}
	}
	for _, agent := range stats.Agents {
		if agent.LastProgress == nil ||
			!mcpAutoprogrammingProgressIsAckCleanupV0(agent.LastProgress.DirectorProgressTemporalV0) {
			continue
		}
		agentRef := agent.AgentRequestID
		if agentRef == "" {
			agentRef = agent.LastProgress.TaskRef
		}
		if agentRef != "" {
			pendingByAgent[agentRef] = true
		}
	}
	return len(pendingByAgent)
}

func mcpAutoprogrammingProgressIsAckCleanupV0(
	progress orquestacionnucleoapp.DirectorProgressTemporalV0,
) bool {
	return progress.Classification == orquestacionnucleoapp.DirectorProgressClassificationAckCleanupV0
}

func mcpAutoprogrammingRunLivenessAgentV0(
	agent orquestacionnucleoapp.DirectorAgentStatsV0,
) orquestaruncoordinator.RunLivenessAgentV0 {
	controlState := agent.ControlState
	out := orquestaruncoordinator.RunLivenessAgentV0{
		Status:         agent.Status,
		InFlight:       agent.InFlight,
		NeedsAttention: agent.NeedsAttention,
		Completed:      agent.Completed,
		Failed:         agent.Failed,
		Lost:           agent.Lost,
		StopConfirmed:  agent.StopConfirmed,
		ProcessLookupChecked: controlState == orquestacionnucleoapp.DirectorAgentControlStateRegisteredV0 ||
			controlState == orquestacionnucleoapp.DirectorAgentControlStateMissingV0,
		ProcessMissing: controlState == orquestacionnucleoapp.DirectorAgentControlStateMissingV0,
	}
	if agent.LastProgress != nil {
		out.LastProgressStatus = agent.LastProgress.Status
	}
	if agent.Process != nil {
		out.ProcessRef = agent.Process.ProcessRef
		out.SessionRef = agent.Process.SessionRef
		out.LaunchRef = agent.Process.LaunchRef
		out.ReadinessRef = agent.Process.ReadinessRef
		out.ProcessStatus = agent.Process.Status
	}
	return out
}
