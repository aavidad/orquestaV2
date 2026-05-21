package orquestadirectoragentworkflow

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	directorAgentWorkflowOperationalSourceCriterionV0 = "operational_director.task_source: director_decision"
	directorAgentWorkflowMaxAcceptanceCriteriaV0      = 40
)

func directorAgentWorkflowOperationalAcceptanceCriteriaV0(
	phaseID string,
	criteria []string,
) []string {
	if orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(phaseID)) !=
		orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		return append([]string(nil), criteria...)
	}
	return appendDirectorAgentWorkflowOperationalMarkerV0(criteria)
}

func appendDirectorAgentWorkflowOperationalMarkerV0(criteria []string) []string {
	out := make([]string, 0, len(criteria)+1)
	hasOperationalMarker := false
	for _, criterion := range criteria {
		trimmed := strings.TrimSpace(criterion)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "operational_director.") {
			hasOperationalMarker = true
		}
		out = append(out, trimmed)
	}
	if hasOperationalMarker || len(out) >= directorAgentWorkflowMaxAcceptanceCriteriaV0 {
		return out
	}
	return append(out, directorAgentWorkflowOperationalSourceCriterionV0)
}
