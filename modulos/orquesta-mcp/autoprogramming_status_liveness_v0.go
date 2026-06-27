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
