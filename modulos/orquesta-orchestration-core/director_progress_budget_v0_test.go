package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestBuildDirectorRunStatsWithObservationsV0TransportaClasificacionTemporal(t *testing.T) {
	cases := []struct {
		suffix         string
		classification orquestaruntime.AgentProgressBudgetStatusV0
	}{
		{"working", orquestaruntime.AgentProgressBudgetWorkingV0},
		{"stalled", orquestaruntime.AgentProgressBudgetStalledV0},
		{"over-budget-active", orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0},
		{"over-budget-no-activity", orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0},
		{"ack-cleanup", orquestaruntime.AgentProgressBudgetAckCleanupV0},
	}

	for _, tc := range cases {
		t.Run(tc.suffix, func(t *testing.T) {
			runRef := "run-nucleo-progress-budget-" + tc.suffix
			taskRef := "task-ref-progress-budget-" + tc.suffix
			agentRef := "agent-ref-progress-budget-" + tc.suffix
			run := mustActiveProgrammingRunV0(t, runRef)
			run.Tasks = []string{taskRef}
			run.Agents = []string{agentRef}
			run.StartedAgents = []string{agentRef}

			stats := BuildDirectorRunStatsWithObservationsV0(run, []AgentProgressObservationV0{
				directorBudgetProgressObservationForTestV0(
					runRef,
					agentRef,
					taskRef,
					tc.classification,
				),
			}, nil)

			agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
			if agent.LastProgress == nil ||
				agent.LastProgress.Classification != string(tc.classification) ||
				agent.LastProgress.Status != string(orquestaruntime.AgentProgressingV0) {
				t.Fatalf("last_progress=%+v", agent.LastProgress)
			}
			task := findDirectorTaskProgressForTestV0(t, stats.Progress, taskRef)
			if task.Classification != string(tc.classification) ||
				task.Status != DirectorTaskProgressInProgressV0 {
				t.Fatalf("task_progress=%+v", task)
			}
			assertDirectorBudgetCountForTestV0(t, stats.Progress, tc.classification)
		})
	}
}

func TestBuildDirectorRunStatsWithObservationsV0TransportaEdadActividadYAck(t *testing.T) {
	runRef := "run-nucleo-progress-budget-temporal-001"
	taskRef := "task-ref-progress-budget-temporal-001"
	agentRef := "agent-ref-progress-budget-temporal-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{taskRef}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}

	stats := BuildDirectorRunStatsWithObservationsV0(run, []AgentProgressObservationV0{
		directorBudgetProgressObservationForTestV0(
			runRef,
			agentRef,
			taskRef,
			orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0,
		),
	}, nil)

	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.LastProgress == nil {
		t.Fatalf("last_progress ausente: %+v", agent)
	}
	task := findDirectorTaskProgressForTestV0(t, stats.Progress, taskRef)
	assertDirectorTemporalProgressForTestV0(t, "agent", agent.LastProgress.DirectorProgressTemporalV0)
	assertDirectorTemporalProgressForTestV0(t, "task", task.DirectorProgressTemporalV0)
}

func TestBuildDirectorRunStatsWithObservationsV0BudgetNoHabilitaStopDeDirectorProtegido(t *testing.T) {
	runRef := "run-nucleo-progress-budget-protected-001"
	agentRef := "agent-spec-progress-budget-protected-director"
	run := mustActiveBrainstormingRunWithStartedAgentV0(t, runRef, agentRef)
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     "process-ref-progress-budget-protected-001",
		SessionRef:     "session-ref-progress-budget-protected-001",
		LaunchRef:      "launch-ref-progress-budget-protected-001",
		ReadinessRef:   "readiness-ref-progress-budget-protected-001",
		EvidenceRefs:   []string{"evidence-ref-progress-budget-protected-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	stats := BuildDirectorRunStatsWithProcessRegistryV0(context.Background(), run, registry)
	ApplyDirectorProgressObservationsV0(&stats, []AgentProgressObservationV0{{
		Report: directorBudgetProgressReportForTestV0(
			runRef,
			agentRef,
			orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0,
		),
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
		TaskRef:      "task-ref-progress-budget-protected-001",
		EvidenceRefs: []string{"evidence-ref-progress-budget-protected-observation-001"},
	}}, nil)

	agent := findDirectorAgentStatsForTestV0(t, stats, agentRef)
	if agent.LastProgress == nil ||
		agent.LastProgress.Classification != DirectorProgressClassificationOverBudgetNoActivityV0 ||
		agent.CanStop {
		t.Fatalf("agent protegido=%+v", agent)
	}
}

func directorBudgetProgressObservationForTestV0(
	runRef string,
	agentRef string,
	taskRef string,
	classification orquestaruntime.AgentProgressBudgetStatusV0,
) AgentProgressObservationV0 {
	return AgentProgressObservationV0{
		Report:      directorBudgetProgressReportForTestV0(runRef, agentRef, classification),
		TaskRef:     taskRef,
		DeliveryRef: "delivery-ref-progress-budget-" + agentRef,
		EvidenceRefs: []string{
			"evidence-ref-progress-budget-observation-" + agentRef,
		},
	}
}

func directorBudgetProgressReportForTestV0(
	runRef string,
	agentRef string,
	classification orquestaruntime.AgentProgressBudgetStatusV0,
) orquestaruntime.AgentProgressReportV0 {
	return orquestaruntime.AgentProgressReportV0{
		ReportID:               "agent-progress-report-ref-budget-" + agentRef,
		RunID:                  runRef,
		AgentRequestID:         agentRef,
		Status:                 orquestaruntime.AgentProgressingV0,
		BudgetStatus:           classification,
		BudgetReason:           "Presupuesto observado con senal compacta.",
		NoProgressTicks:        1,
		RepeatedActionCount:    0,
		AgeSeconds:             900,
		SecondsSinceActivity:   45,
		SecondsSinceAck:        120,
		MaxExpectedSeconds:     600,
		NoActivityLimitSeconds: 180,
		StartedAt:              "2026-05-11T14:00:00Z",
		LastActivityAt:         "2026-05-11T14:14:15Z",
		LastAckAt:              "2026-05-11T14:13:00Z",
		DecisionRequired:       classification == orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0,
		Summary:                "Progreso compacto con presupuesto temporal.",
		EvidenceRefs:           []string{"evidence-ref-progress-budget-" + agentRef},
	}
}

func assertDirectorTemporalProgressForTestV0(
	t *testing.T,
	label string,
	progress DirectorProgressTemporalV0,
) {
	t.Helper()
	if progress.Classification != DirectorProgressClassificationOverBudgetNoActivityV0 ||
		progress.BudgetReason != "Presupuesto observado con senal compacta." ||
		progress.AgeSeconds != 900 ||
		progress.SecondsSinceActivity != 45 ||
		progress.SecondsSinceAck != 120 ||
		progress.MaxExpectedSeconds != 600 ||
		progress.NoActivityLimitSeconds != 180 ||
		progress.StartedAt != "2026-05-11T14:00:00Z" ||
		progress.LastActivityAt != "2026-05-11T14:14:15Z" ||
		progress.LastAckAt != "2026-05-11T14:13:00Z" ||
		!progress.DecisionRequired {
		t.Fatalf("%s temporal=%+v", label, progress)
	}
}

func assertDirectorBudgetCountForTestV0(
	t *testing.T,
	progress DirectorProgressStatsV0,
	classification orquestaruntime.AgentProgressBudgetStatusV0,
) {
	t.Helper()
	switch classification {
	case orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0:
		if progress.OverBudgetButActiveAgents != 1 ||
			progress.OverBudgetNoActivityAgents != 0 {
			t.Fatalf("budget counts=%+v", progress)
		}
	case orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0:
		if progress.OverBudgetButActiveAgents != 0 ||
			progress.OverBudgetNoActivityAgents != 1 {
			t.Fatalf("budget counts=%+v", progress)
		}
	default:
		if progress.OverBudgetButActiveAgents != 0 ||
			progress.OverBudgetNoActivityAgents != 0 {
			t.Fatalf("budget counts=%+v", progress)
		}
	}
}
