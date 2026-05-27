package orquestamcp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func buildMCPDirectorDecisionContextV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	stats orquestacionnucleoapp.DirectorRunStatsV0,
	observedAt string,
) *orquestaobservability.DirectorDecisionContextV0 {
	observedAt = mcpDirectorObservedAtV0(observedAt)
	context := orquestaobservability.DirectorDecisionContextV0{
		SchemaVersion: orquestaobservability.DirectorDecisionContextSchemaVersionV0,
		RunRef:        strings.TrimSpace(stats.RunRef),
		ObservedAt:    observedAt,
		CurrentPhase:  strings.TrimSpace(stats.CurrentPhase),
		Progress:      mcpDirectorDecisionProgressV0(stats),
		Lifecycle:     mcpDirectorDecisionLifecycleV0(stats),
		Closure:       mcpDirectorDecisionClosureV0(stats),
		Quietness:     mcpDirectorDecisionQuietnessV0(stats),
		ReworkReplan:  mcpDirectorDecisionReworkReplanV0(stats),
		Phases:        mcpDirectorDecisionPhasesV0(run, stats, observedAt),
		Tasks:         mcpDirectorDecisionTasksV0(stats),
		Agents:        mcpDirectorDecisionAgentsV0(stats),
		Privacy:       orquestaobservability.NewDiagnosticoPrivacyMetadataOnlyV0(),
	}
	context.Blockers = mcpDirectorDecisionBlockersV0(stats)
	context.Activity = mcpDirectorDecisionActivityV0(context, observedAt)
	context.Warnings = mcpDirectorDecisionWarningsV0(stats, observedAt)
	return &context
}

func mcpDirectorDecisionProgressV0(stats orquestacionnucleoapp.DirectorRunStatsV0) orquestaobservability.DirectorDecisionProgressV0 {
	return orquestaobservability.DirectorDecisionProgressV0{
		PercentComplete:    stats.Progress.PercentComplete,
		TasksTotal:         stats.Progress.TasksTotal,
		TasksClosed:        stats.Progress.TasksClosed,
		TasksOpen:          stats.Counts.TasksOpen,
		TasksObserved:      stats.Progress.TasksObserved,
		ObservedAgents:     stats.Progress.ObservedAgents,
		ProgressingAgents:  stats.Progress.ProgressingAgents,
		StalledAgents:      stats.Progress.StalledAgents,
		LoopDetectedAgents: stats.Progress.LoopDetectedAgents,
		StoppedAgents:      stats.Progress.StoppedAgents,
		NoSignalAgentRefs:  compactStringsMCPV0(stats.Progress.NoSignalAgentRefs),
	}
}

func mcpDirectorDecisionLifecycleV0(stats orquestacionnucleoapp.DirectorRunStatsV0) orquestaobservability.DirectorDecisionLifecycleV0 {
	lifecycle := orquestaobservability.DirectorDecisionLifecycleV0{
		AgentsRequested:         stats.Counts.AgentsRequested,
		AgentsStarted:           stats.Counts.AgentsStarted,
		AgentsFailed:            stats.Counts.AgentsFailed,
		AgentsStopRequested:     stats.Counts.AgentsStopRequested,
		AgentsStopped:           stats.Counts.AgentsStopConfirmed,
		AgentsInFlight:          stats.Counts.AgentsInFlight,
		AgentsControlRegistered: stats.Counts.AgentsControlRegistered,
		AgentsControlMissing:    stats.Counts.AgentsControlMissing,
	}
	for _, agent := range stats.Agents {
		if agent.Status == orquestacionnucleoapp.DirectorAgentStatusRunningV0 {
			lifecycle.AgentsRunning++
		}
		if agent.NeedsAttention {
			lifecycle.AgentsNeedAttention++
		}
	}
	return lifecycle
}

func mcpDirectorDecisionClosureV0(stats orquestacionnucleoapp.DirectorRunStatsV0) orquestaobservability.DirectorDecisionClosureV0 {
	return orquestaobservability.DirectorDecisionClosureV0{
		Status:      strings.TrimSpace(stats.Closure.Status),
		Blocked:     stats.Closure.Blocked,
		Ready:       stats.Closure.Ready,
		Closed:      stats.Closure.Closed,
		BlockedBy:   compactStringsMCPV0(stats.Closure.BlockedBy),
		BlockerRefs: compactStringsMCPV0(stats.Closure.BlockerRefs),
	}
}

func mcpDirectorDecisionQuietnessV0(stats orquestacionnucleoapp.DirectorRunStatsV0) orquestaobservability.DirectorDecisionQuietnessV0 {
	quietness := orquestaobservability.DirectorDecisionQuietnessV0{
		NoSignalAgentRefs:  compactStringsMCPV0(stats.Progress.NoSignalAgentRefs),
		TasksWithoutSignal: nonNegativeMCPDirectorV0(stats.Progress.TasksTotal - stats.Progress.TasksObserved),
		StalledAgents:      stats.Progress.StalledAgents,
		LoopDetectedAgents: stats.Progress.LoopDetectedAgents,
		StoppedAgents:      stats.Progress.StoppedAgents,
	}
	for _, task := range stats.Progress.Tasks {
		quietness.MaxNoProgressTicks = maxMCPDirectorV0(quietness.MaxNoProgressTicks, task.NoProgressTicks)
	}
	for _, agent := range stats.Agents {
		if agent.LastProgress != nil {
			quietness.MaxNoProgressTicks = maxMCPDirectorV0(quietness.MaxNoProgressTicks, agent.LastProgress.NoProgressTicks)
		}
	}
	return quietness
}

func mcpDirectorDecisionReworkReplanV0(stats orquestacionnucleoapp.DirectorRunStatsV0) orquestaobservability.DirectorDecisionReworkReplanV0 {
	return orquestaobservability.DirectorDecisionReworkReplanV0{
		ReworkRequests:     stats.Counts.ReworkRequests,
		ReworkRequestRefs:  compactStringsMCPV0(stats.Refs.ReworkRequests),
		ReplanDecisions:    stats.Counts.ReplanDecisions,
		ReplanDecisionRefs: compactStringsMCPV0(stats.Refs.ReplanDecisions),
	}
}
