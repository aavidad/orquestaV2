package orquestamcp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func mcpDirectorContextHasCurrentPhaseDurationV0(
	context *orquestaobservability.DirectorDecisionContextV0,
	phaseID string,
	duration int64,
) bool {
	for _, phase := range context.Phases {
		if phase.PhaseID == phaseID && phase.Current && phase.DurationSeconds == duration {
			return true
		}
	}
	return false
}

func mcpDirectorStatsRunForTestV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := mcpDirectorDecisionRunForTestV0(t, runRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	run.Tasks = []string{"task-ref-stats-001", "task-ref-stats-002", "task-ref-stats-003"}
	run.ClosedTasks = []string{"task-ref-stats-002"}
	run.Agents = []string{"agent-ref-stats-001", "agent-ref-stats-002"}
	run.StartedAgents = []string{"agent-ref-stats-001"}
	run.Deliveries = []string{"delivery-ref-stats-001"}
	return run
}

func mcpDirectorStatsAgentForTestV0(
	t *testing.T,
	stats orquestacionnucleoapp.DirectorRunStatsV0,
	agentRef string,
) orquestacionnucleoapp.DirectorAgentStatsV0 {
	t.Helper()
	for _, agent := range stats.Agents {
		if agent.AgentRequestID == agentRef {
			return agent
		}
	}
	t.Fatalf("agent no encontrado: %s en %+v", agentRef, stats.Agents)
	return orquestacionnucleoapp.DirectorAgentStatsV0{}
}

type mcpDirectorStatsProgressSourceForTestV0 struct {
	Observations []orquestacionnucleoapp.AgentProgressObservationV0
}

func (source mcpDirectorStatsProgressSourceForTestV0) BuildAgentProgressObservationsV0(
	context.Context,
	orquestacionnucleoapp.AgentProgressObservationRequestV0,
) ([]orquestacionnucleoapp.AgentProgressObservationV0, error) {
	return source.Observations, nil
}

type mcpDirectorStatsUsageSourceForTestV0 struct {
	Observations []orquestacionnucleoapp.AgentUsageStatsObservationV0
}

func (source mcpDirectorStatsUsageSourceForTestV0) BuildAgentUsageStatsV0(
	context.Context,
	orquestacionnucleoapp.AgentUsageStatsRequestV0,
) ([]orquestacionnucleoapp.AgentUsageStatsObservationV0, error) {
	return source.Observations, nil
}

type mcpDirectorExternalJobStatsSourceForTestV0 struct {
	Stats MCPDirectorExternalJobStatsV0
}

func (source mcpDirectorExternalJobStatsSourceForTestV0) ResolveDirectorExternalJobStatsV0(
	_ context.Context,
	request MCPDirectorExternalJobStatsRequestV0,
) (MCPDirectorExternalJobStatsV0, bool, error) {
	if source.Stats.JobRef != request.ExternalJobRef {
		return MCPDirectorExternalJobStatsV0{}, false, nil
	}
	if request.AppRef != "" && source.Stats.AppRef != request.AppRef {
		return MCPDirectorExternalJobStatsV0{}, false, nil
	}
	return source.Stats, true, nil
}

func mcpDirectorStatsProgressObservationForTestV0(
	runRef string,
) orquestacionnucleoapp.AgentProgressObservationV0 {
	observation := orquestacionnucleoapp.AgentProgressObservationV0{
		TaskRef:      "task-ref-stats-001",
		DeliveryRef:  "delivery-ref-stats-001",
		EvidenceRefs: []string{"evidence-ref-mcp-director-stats-progress-001"},
	}
	observation.Report.ReportID = "agent-progress-report-ref-mcp-stats-001"
	observation.Report.RunID = runRef
	observation.Report.AgentRequestID = "agent-ref-stats-001"
	observation.Report.Status = "progressing"
	observation.Report.Summary = "avance observado por puerto de progreso"
	observation.Report.EvidenceRefs = []string{"evidence-ref-mcp-director-stats-progress-report-001"}
	return observation
}
