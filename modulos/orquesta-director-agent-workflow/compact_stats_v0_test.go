package orquestadirectoragentworkflow

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestBuildDirectorAgentCompactStatsV0ResumeRunSinDetallesOperativos(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:                     "run-ref-001",
		Status:                    orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:              orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:                    []orquestacoreworkflow.OrchestrationPhaseV0{{ID: orquestacoreworkflow.OrchestrationPhaseProgramacionV0}},
		Tasks:                     []string{"task-ref-001", "task-ref-002"},
		ClosedTasks:               []string{"task-ref-001"},
		CapacityRequests:          []string{"capacity-request-ref-001"},
		CapacityDecisions:         nil,
		Agents:                    []string{"agent-request-ref-001"},
		StartedAgents:             nil,
		DirectorQuestions:         []string{"question-ref-001"},
		DirectorAnsweredQuestions: nil,
		ReworkRequests:            []string{"rework-request-ref-001"},
		ReplanDecisions:           []string{"replan-ref-001"},
	}

	stats := BuildDirectorAgentCompactStatsV0(run)

	if issues := orquestadirectoragent.ValidateDirectorAgentCompactStatsV0(stats); len(issues) != 0 {
		t.Fatalf("stats invalidas: %+v", issues)
	}
	if stats.Totals.Tasks != 2 || stats.Totals.ClosedTasks != 1 ||
		stats.Totals.ReworkRequests != 1 || stats.Totals.ReplanDecisions != 1 {
		t.Fatalf("totals=%+v", stats.Totals)
	}
	if !statsHasPendingRefV0(stats, "tasks_open") ||
		!statsHasPendingRefV0(stats, "capacity_pending") ||
		!statsHasPendingRefV0(stats, "agents_pending") ||
		!statsHasPendingRefV0(stats, "director_questions_pending") {
		t.Fatalf("pending_refs=%+v", stats.PendingRefs)
	}
}

func statsHasPendingRefV0(stats orquestadirectoragent.DirectorAgentCompactStatsV0, ref string) bool {
	for _, candidate := range stats.PendingRefs {
		if candidate == ref {
			return true
		}
	}
	return false
}
