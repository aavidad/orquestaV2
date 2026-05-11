package orquestadirectoragentworkflow

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func BuildDirectorAgentCompactStatsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestadirectoragent.DirectorAgentCompactStatsV0 {
	return orquestadirectoragent.NormalizeDirectorAgentCompactStatsV0(
		orquestadirectoragent.DirectorAgentCompactStatsV0{
			SchemaVersion: orquestadirectoragent.DirectorAgentCompactStatsSchemaVersionV0,
			RunID:         run.RunID,
			Status:        string(run.Status),
			CurrentPhase:  string(run.CurrentPhase),
			Totals:        directorAgentStatsTotalsV0(run),
			PendingRefs:   directorAgentPendingStatsRefsV0(run),
			EvidenceRefs:  []string{"workflow-run:" + run.RunID},
		},
	)
}

func directorAgentStatsTotalsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) orquestadirectoragent.DirectorAgentStatsTotalsV0 {
	return orquestadirectoragent.DirectorAgentStatsTotalsV0{
		Phases:           len(run.Phases),
		Tasks:            len(run.Tasks),
		CapacityRequests: len(run.CapacityRequests),
		Agents:           len(run.Agents),
		Deliveries:       len(run.Deliveries),
		ReviewResults:    len(run.ReviewResults),
		ReworkRequests:   len(run.ReworkRequests),
		ReplanDecisions:  len(run.ReplanDecisions),
		ClosedTasks:      len(run.ClosedTasks),
		Validations:      len(run.Validations),
		Closures:         len(run.Closures),
	}
}

func directorAgentPendingStatsRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	refs := make([]string, 0, 4)
	if len(run.Tasks) > len(run.ClosedTasks) {
		refs = append(refs, "tasks_open")
	}
	if len(run.CapacityRequests) > len(run.CapacityDecisions) {
		refs = append(refs, "capacity_pending")
	}
	if len(run.Agents) > len(run.StartedAgents)+len(run.FailedAgents) {
		refs = append(refs, "agents_pending")
	}
	if len(run.DirectorQuestions) > len(run.DirectorAnsweredQuestions) {
		refs = append(refs, "director_questions_pending")
	}
	return refs
}
