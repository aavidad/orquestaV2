package orquestadirectoragentworkflow

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestApplyDirectorAgentDecisionV0MaterializaPlanEquipoComoMicrotareas(t *testing.T) {
	run := directorAgentWorkflowPlanningRunForTestV0(t, "run-ref-001")
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ports := ApplyDirectorAgentDecisionPortsV0{
		RunStore:  store,
		EventSink: sink,
		TaskStore: taskStore,
	}
	if _, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(validDirectorAgentWorkflowContractRequestForTestV0()),
		ports,
	); err != nil {
		t.Fatalf("preparar contrato: %v", err)
	}

	result, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(validDirectorAgentWorkflowPlanTeamApplyRequestForTestV0()),
		ports,
	)
	if err != nil {
		t.Fatalf("ApplyDirectorAgentDecisionV0 plan team: %v", err)
	}
	if len(result.Issues) != 0 || result.EventsCount != 2 || len(result.Commands) != 2 {
		t.Fatalf("result=%+v", result)
	}
	for _, taskRef := range []string{"task-ref-agenda-domain-001", "task-ref-agenda-api-001"} {
		if !directorAgentWorkflowStringInSetV0(result.Run.Tasks, taskRef) {
			t.Fatalf("tasks=%v missing=%s", result.Run.Tasks, taskRef)
		}
	}
	storedTasks, err := taskStore.LoadWorkflowTasksV0(
		context.Background(),
		"run-ref-001",
		[]string{"task-ref-agenda-domain-001", "task-ref-agenda-api-001"},
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(storedTasks) != 2 ||
		storedTasks[1].DependsOn[0] != "task-ref-agenda-domain-001" ||
		storedTasks[0].PhaseID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("stored tasks=%+v", storedTasks)
	}
	if !directorAgentWorkflowSinkHasEventV0(sink, orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0) {
		t.Fatalf("sink sin MicrotaskCreated: %+v", sink.EventsV0())
	}
}

func TestApplyDirectorAgentDecisionV0RequiereTaskStoreParaPlanEquipo(t *testing.T) {
	run := directorAgentWorkflowPlanningRunForTestV0(t, "run-ref-001")
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := ApplyDirectorAgentDecisionV0(
		context.Background(),
		ApplyDirectorAgentDecisionRequestV0(validDirectorAgentWorkflowPlanTeamApplyRequestForTestV0()),
		ApplyDirectorAgentDecisionPortsV0{RunStore: store, EventSink: sink},
	)
	if err != nil {
		t.Fatalf("no debe devolver error tecnico: %v", err)
	}
	requireDirectorAgentWorkflowIssueV0(t, result.Issues, "director_agent_workflow_required")
}

func TestDirectorAgentAutonomousWorkUnitRequiredTestsV0PreservaDTOConRequiredTests(t *testing.T) {
	type workUnitWithRequiredTestsV0 struct {
		RequiredTests []string
	}

	got := directorAgentAutonomousWorkUnitRequiredTestsV0(workUnitWithRequiredTestsV0{
		RequiredTests: []string{"go test -count=1 ./modulos/orquesta-director-agent-workflow"},
	})
	if len(got) != 1 || got[0] != "go test -count=1 ./modulos/orquesta-director-agent-workflow" {
		t.Fatalf("required_tests=%v", got)
	}
}

func validDirectorAgentWorkflowPlanTeamApplyRequestForTestV0() DirectorAgentWorkflowCommandRequestV0 {
	request := validDirectorAgentWorkflowRequestForTestV0()
	request.Decision = orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-plan-team-apply-001",
		RunID:         "run-ref-001",
		PhaseID:       orquestadirectoragent.DirectorAgentPlanningPhaseIDV0,
		CommandType:   orquestadirectoragent.DirectorAgentCommandProposePlanTeamV0,
		CommandRef:    "command-ref-plan-team-apply-001",
		Summary:       "Materializar equipo autonomo en microtareas.",
		EvidenceRefs:  []string{"evidence-ref-plan-team-apply-001"},
		ProposePlanTeam: &orquestadirectoragent.DirectorAgentPlanTeamCommandV0{
			Plan: orquestadirectoragent.DirectorAgentAutonomousPlanTeamV0{
				SchemaVersion: orquestadirectoragent.DirectorAgentAutonomousPlanTeamSchemaVersionV0,
				PlanRef:       "plan-ref-autonomous-team-apply-001",
				RunID:         "run-ref-001",
				PhaseID:       orquestadirectoragent.DirectorAgentPlanningPhaseIDV0,
				GoalRef:       "goal-ref-agenda-apply-001",
				Summary:       "Dividir agenda en dos unidades ejecutables.",
				Team: []orquestadirectoragent.DirectorAgentTeamMemberV0{
					{MemberRef: "member-ref-domain-001", Role: "dominio", ResponsibilityRefs: []string{"task-ref-agenda-domain-001"}},
					{MemberRef: "member-ref-api-001", Role: "api", ResponsibilityRefs: []string{"task-ref-agenda-api-001"}},
				},
				WorkUnits: []orquestadirectoragent.DirectorAgentAutonomousWorkUnitV0{
					{
						WorkUnitRef:       "task-ref-agenda-domain-001",
						PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
						Title:             "Implementar dominio de agenda",
						Summary:           "Crear casos de uso de agenda compactos.",
						AssignedMemberRef: "member-ref-domain-001",
						WriteSet:          []string{"internal/agenda/domain"},
						AcceptanceCriteria: []string{
							"Compila con pruebas unitarias.",
							"Respeta contrato funcional publicado.",
						},
						FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{
							{ContractRef: "contract:function:agenda:v0", FunctionName: "AgendaUseCases"},
						},
					},
					{
						WorkUnitRef:       "task-ref-agenda-api-001",
						PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
						Title:             "Implementar API de agenda",
						Summary:           "Exponer interfaz compacta de agenda.",
						AssignedMemberRef: "member-ref-api-001",
						WriteSet:          []string{"internal/agenda/api"},
						AcceptanceCriteria: []string{
							"Usa casos de uso publicados.",
							"Propaga errores estables.",
						},
						FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{
							{ContractRef: "contract:function:agenda:v0", FunctionName: "AgendaUseCases"},
						},
						DependsOn: []string{"task-ref-agenda-domain-001"},
					},
				},
				EvidenceRefs: []string{"evidence-ref-plan-team-apply-002"},
			},
		},
	}
	return request
}
