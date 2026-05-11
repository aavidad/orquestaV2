package orquestamcp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func mcpDirectorDecisionPhasesV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	stats orquestacionnucleoapp.DirectorRunStatsV0,
	observedAt string,
) []orquestaobservability.DirectorDecisionPhaseV0 {
	if len(run.Phases) == 0 {
		return mcpDirectorDecisionPhasesFromStatsV0(stats)
	}
	out := make([]orquestaobservability.DirectorDecisionPhaseV0, 0, len(run.Phases))
	for _, phase := range run.Phases {
		out = append(out, orquestaobservability.DirectorDecisionPhaseV0{
			PhaseID:             strings.TrimSpace(string(phase.ID)),
			Status:              strings.TrimSpace(string(phase.Status)),
			Current:             strings.TrimSpace(string(phase.ID)) == strings.TrimSpace(stats.CurrentPhase),
			OpenedAt:            strings.TrimSpace(phase.OpenedAt),
			ClosedAt:            strings.TrimSpace(phase.ClosedAt),
			DurationSeconds:     mcpDirectorDurationSecondsV0(phase.OpenedAt, phase.ClosedAt, observedAt),
			RecommendedCapacity: strings.TrimSpace(string(phase.RecommendedCapacity)),
			EntryCriteria:       len(compactStringsMCPV0(phase.EntryCriteria)),
			ExitCriteria:        len(compactStringsMCPV0(phase.ExitCriteria)),
			EvidenceRequired:    len(compactStringsMCPV0(phase.EvidenceRequired)),
		})
	}
	return out
}

func mcpDirectorDecisionPhasesFromStatsV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) []orquestaobservability.DirectorDecisionPhaseV0 {
	out := make([]orquestaobservability.DirectorDecisionPhaseV0, 0, len(stats.Phases))
	for _, phase := range stats.Phases {
		out = append(out, orquestaobservability.DirectorDecisionPhaseV0{
			PhaseID:             strings.TrimSpace(phase.PhaseID),
			Status:              strings.TrimSpace(phase.Status),
			Current:             strings.TrimSpace(phase.PhaseID) == strings.TrimSpace(stats.CurrentPhase),
			RecommendedCapacity: strings.TrimSpace(phase.RecommendedCapacity),
			EntryCriteria:       phase.EntryCriteria,
			ExitCriteria:        phase.ExitCriteria,
			EvidenceRequired:    phase.EvidenceRequired,
		})
	}
	return out
}

func mcpDirectorDecisionTasksV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) []orquestaobservability.DirectorDecisionTaskV0 {
	out := make([]orquestaobservability.DirectorDecisionTaskV0, 0, len(stats.Progress.Tasks))
	for _, task := range stats.Progress.Tasks {
		out = append(out, orquestaobservability.DirectorDecisionTaskV0{
			TaskRef:             strings.TrimSpace(task.TaskRef),
			Status:              strings.TrimSpace(task.Status),
			AgentRequestID:      strings.TrimSpace(task.AgentRequestID),
			DeliveryRef:         strings.TrimSpace(task.DeliveryRef),
			LastReportRef:       strings.TrimSpace(task.LastReportRef),
			ProgressStatus:      strings.TrimSpace(task.ProgressStatus),
			NoProgressTicks:     task.NoProgressTicks,
			RepeatedActionCount: task.RepeatedActionCount,
			EvidenceRefs:        compactStringsMCPV0(task.EvidenceRefs),
		})
	}
	return out
}

func mcpDirectorDecisionAgentsV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) []orquestaobservability.DirectorDecisionAgentV0 {
	out := make([]orquestaobservability.DirectorDecisionAgentV0, 0, len(stats.Agents))
	for _, agent := range stats.Agents {
		item := orquestaobservability.DirectorDecisionAgentV0{
			AgentRequestID: strings.TrimSpace(agent.AgentRequestID),
			Status:         strings.TrimSpace(agent.Status),
			ControlState:   strings.TrimSpace(agent.ControlState),
			Running:        agent.Status == orquestacionnucleoapp.DirectorAgentStatusRunningV0,
			Failed:         agent.Failed,
			StopRequested:  agent.StopRequested,
			Stopped:        agent.StopConfirmed,
			InFlight:       agent.InFlight,
			NeedsAttention: agent.NeedsAttention,
			CanStop:        agent.CanStop,
		}
		if agent.Process != nil {
			item.ProcessRef = strings.TrimSpace(agent.Process.ProcessRef)
			item.SessionRef = strings.TrimSpace(agent.Process.SessionRef)
			item.LaunchRef = strings.TrimSpace(agent.Process.LaunchRef)
			item.ReadinessRef = strings.TrimSpace(agent.Process.ReadinessRef)
			item.EvidenceRefs = compactStringsMCPV0(agent.Process.EvidenceRefs)
		}
		if agent.LastProgress != nil {
			item.ProgressTaskRef = strings.TrimSpace(agent.LastProgress.TaskRef)
			item.ProgressStatus = strings.TrimSpace(agent.LastProgress.Status)
			item.NoProgressTicks = agent.LastProgress.NoProgressTicks
			item.RepeatedActionCount = agent.LastProgress.RepeatedActionCount
			item.EvidenceRefs = compactStringsMCPV0(append(item.EvidenceRefs, agent.LastProgress.EvidenceRefs...))
		}
		out = append(out, item)
	}
	return out
}
