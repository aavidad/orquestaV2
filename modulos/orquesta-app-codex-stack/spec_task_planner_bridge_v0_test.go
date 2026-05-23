package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgrammingTaskV0PuenteAppPlannerWorkProfileAAgentStartTask(t *testing.T) {
	plan := orquestaappplanner.AppMicrotaskPlanV0{
		SchemaVersion: orquestaappplanner.AppMicrotaskPlanSchemaVersionV0,
		RunRef:        "run-ref-app-planner-bridge-001",
		AppRef:        "agenda",
		Units: []orquestaappplanner.AppWorkUnitV0{
			appPlannerBridgeUnitForTestV0("code-study", "analisis", orquestacoreworkflow.WorkProfileCodeStudyV0),
			appPlannerBridgeUnitForTestV0("implementation", "implementacion", orquestacoreworkflow.WorkProfileImplementationV0),
			appPlannerBridgeUnitForTestV0("refactor", "refactor", orquestacoreworkflow.WorkProfileRefactorV0),
			appPlannerBridgeUnitForTestV0("required-tests", "pruebas", orquestacoreworkflow.WorkProfileRequiredTestsV0),
		},
	}
	cases := appPlannerBridgeCasesForTestV0(plan)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profile, err := orquestaappplanner.WorkProfileForUnitV0(plan, tc.unit)
			if err != nil {
				t.Fatalf("WorkProfileForUnitV0: %v", err)
			}
			if profile.ProfileKind != tc.wantKind {
				t.Fatalf("profile_kind=%s want=%s", profile.ProfileKind, tc.wantKind)
			}
			task, err := orquestaappplanner.WorkflowTaskForUnitV0(plan, tc.unit)
			if err != nil {
				t.Fatalf("WorkflowTaskForUnitV0: %v", err)
			}
			if task.WorkProfileKind != tc.wantKind {
				t.Fatalf("task work_profile_kind=%s want=%s", task.WorkProfileKind, tc.wantKind)
			}
			assertAppPlannerBridgeAgentTaskForTestV0(t, plan, task, tc)
		})
	}
}

type appPlannerBridgeCaseV0 struct {
	name          string
	unit          orquestaappplanner.AppWorkUnitV0
	wantKind      orquestacoreworkflow.WorkProfileKindV0
	wantObjective string
	reject        string
}

func appPlannerBridgeCasesForTestV0(
	plan orquestaappplanner.AppMicrotaskPlanV0,
) []appPlannerBridgeCaseV0 {
	return []appPlannerBridgeCaseV0{
		{
			name:          "code_study",
			unit:          plan.Units[0],
			wantKind:      orquestacoreworkflow.WorkProfileCodeStudyV0,
			wantObjective: "Estudia el codigo",
			reject:        "Implementa solo",
		},
		{
			name:          "implementation",
			unit:          plan.Units[1],
			wantKind:      orquestacoreworkflow.WorkProfileImplementationV0,
			wantObjective: "Implementa solo",
		},
		{
			name:          "refactor",
			unit:          plan.Units[2],
			wantKind:      orquestacoreworkflow.WorkProfileRefactorV0,
			wantObjective: "Refactoriza solo",
			reject:        "Implementa solo",
		},
		{
			name:          "required_tests",
			unit:          plan.Units[3],
			wantKind:      orquestacoreworkflow.WorkProfileRequiredTestsV0,
			wantObjective: "pruebas requeridas",
			reject:        "Implementa solo",
		},
	}
}

func assertAppPlannerBridgeAgentTaskForTestV0(
	t *testing.T,
	plan orquestaappplanner.AppMicrotaskPlanV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	tc appPlannerBridgeCaseV0,
) {
	t.Helper()
	resolver := CodexLaunchSpecResolverV0{
		Config: CodexRuntimeConfigV0{
			RuntimeWorkDir: t.TempDir(),
		},
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}
	got, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), orquestaruntime.AgentLauncherInboundV0{
		CorrelationID: "corr-ref-app-planner-bridge-001",
		Payload: &orquestaruntime.LaunchRuntimeAgentRequestV0{
			AgentRequestID: tc.unit.AgentRequestID,
			RunID:          plan.RunRef,
			PhaseID:        string(task.PhaseID),
			TaskRef:        task.TaskID,
			Role:           tc.unit.Role,
		},
	})
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	agentTask := got.Spec.AgentPacket.Task
	if got.Spec.AgentPacket.WorkOrderRef != task.TaskID ||
		got.Spec.RequestID != tc.unit.AgentRequestID ||
		agentTask.TaskRef != task.TaskID ||
		agentTask.Title != tc.unit.Title {
		t.Fatalf("agent_start_task refs no propagados: packet=%+v task=%+v", got.Spec.AgentPacket, agentTask)
	}
	if len(agentTask.WriteSet) != 1 || agentTask.WriteSet[0] != tc.unit.WriteSet[0] {
		t.Fatalf("write_set=%v want=%v", agentTask.WriteSet, tc.unit.WriteSet)
	}
	if len(agentTask.RequiredTests) != 1 || agentTask.RequiredTests[0] != "go test ./..." {
		t.Fatalf("required_tests=%v", agentTask.RequiredTests)
	}
	if !strings.Contains(agentTask.Objective, tc.wantObjective) {
		t.Fatalf("objective no contiene %q:\n%s", tc.wantObjective, agentTask.Objective)
	}
	if tc.reject != "" && strings.Contains(agentTask.Objective, tc.reject) {
		t.Fatalf("objective generico para %s:\n%s", tc.name, agentTask.Objective)
	}
}

func appPlannerBridgeUnitForTestV0(
	key string,
	role string,
	kind orquestacoreworkflow.WorkProfileKindV0,
) orquestaappplanner.AppWorkUnitV0 {
	return orquestaappplanner.AppWorkUnitV0{
		TaskRef:         "task-ref-app-planner-bridge-" + key,
		ClaimRef:        "claim-ref-app-planner-bridge-" + key,
		AgentRequestID:  "agent-ref-app-planner-bridge-" + key,
		DeliveryRef:     "delivery-ref-app-planner-bridge-" + key,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: kind,
		Role:            role,
		Capacity:        orquestacoreworkflow.OrchestrationCapacityMediumV0,
		Title:           "Trabajo " + key,
		Summary:         "Resolver perfil " + key + " desde AppPlanner.",
		WriteSet:        []string{"internal/" + key},
		AcceptanceCriteria: []string{
			"ACK registra resultado del perfil.",
		},
		RequiredTests: []string{"go test ./..."},
	}
}
