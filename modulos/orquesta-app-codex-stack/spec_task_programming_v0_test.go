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

func TestProgrammingTaskV0PropagaRequiredTestsYContratoGoCompleto(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		TaskID:   "task-programacion-api-001",
		RunID:    "run-programacion-api-001",
		PhaseID:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:    "Crear API Go",
		Summary:  "Implementar API de agenda.",
		WriteSet: []string{"cmd/server", "internal/agenda"},
		AcceptanceCriteria: []string{
			"go.mod e imports de modulo",
			"sin imports relativos ../",
		},
		ParentTaskRef:   "task-ref-parent-001",
		CohortRef:       "cohort-ref-recursive-001",
		WaveRef:         "wave-ref-recursive-001",
		DelegationDepth: 2,
		MaxChildAgents:  6,
		ChildTaskRefs:   []string{"task-ref-child-002"},
		RequiredTests:   []string{"go test ./..."},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:agenda-api:v0",
			FunctionName: "CrearAPI",
		}},
	}
	resolver := CodexLaunchSpecResolverV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}

	got, err := resolver.agentTaskV0(context.Background(), orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   task.RunID,
		PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef: task.TaskID,
	}, "programacion")
	if err != nil {
		t.Fatalf("agentTaskV0: %v", err)
	}
	if len(got.RequiredTests) != 1 || got.RequiredTests[0] != "go test ./..." {
		t.Fatalf("required_tests=%v", got.RequiredTests)
	}
	if got.ParentTaskRef != "task-ref-parent-001" ||
		got.CohortRef != "cohort-ref-recursive-001" ||
		got.WaveRef != "wave-ref-recursive-001" ||
		got.DelegationDepth != 2 ||
		got.MaxChildAgents != 6 ||
		len(got.ChildTaskRefs) != 1 ||
		got.ChildTaskRefs[0] != "task-ref-child-002" {
		t.Fatalf("linaje no propagado: %+v", got)
	}
	for _, want := range []string{
		"contrato de esta tarea completa",
		"go.mod",
		"cmd/server",
		"imports de modulo",
		"sin imports relativos ../",
	} {
		if !strings.Contains(got.Objective, want) && !codexStackDoneCriteriaContainsForTestV0(got.DoneCriteria, want) {
			t.Fatalf("contrato Go completo no contiene %q: objective=%s criteria=%v", want, got.Objective, got.DoneCriteria)
		}
	}
	if strings.Contains(got.Objective, "microtarea") {
		t.Fatalf("objective no debe pedir microtareas: %s", got.Objective)
	}
}

func TestProgrammingTaskV0TrabajoExternoUsaUnidadTrabajoNoMicrotareaMinima(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		TaskID:  "task-programacion-opes-001",
		RunID:   "run-programacion-opes-001",
		PhaseID: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:   "Redactar capitulo OPES",
		Summary: "Redactar unidad editorial amplia con paquete de dominio suficiente.",
		WriteSet: []string{
			"external/opes/draft_content_block",
		},
		AcceptanceCriteria: []string{
			"usar paquete de dominio suficiente",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:app-change:opes:v0",
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	resolver := CodexLaunchSpecResolverV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}

	got, err := resolver.agentTaskV0(context.Background(), orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   task.RunID,
		PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef: task.TaskID,
	}, "programacion")
	if err != nil {
		t.Fatalf("agentTaskV0: %v", err)
	}
	if !strings.Contains(got.Objective, "unidad de trabajo externa") ||
		strings.Contains(got.Objective, "microtarea") {
		t.Fatalf("objective=%s", got.Objective)
	}
}

func TestProgrammingTaskV0IncluyeContextoDeReworkSinRehacerTodo(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             "task-programacion-web-001",
		RunID:              "run-programacion-web-001",
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Crear web",
		Summary:            "Implementar agenda.",
		WriteSet:           []string{"web"},
		AcceptanceCriteria: []string{"web completada"},
	}
	resolver := CodexLaunchSpecResolverV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}

	got, err := resolver.agentTaskV0(context.Background(), orquestaruntime.LaunchRuntimeAgentRequestV0{
		RunID:   task.RunID,
		PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskRef: task.TaskID,
		Summary: "Corregir entrega tras revision; conservar lo valido y completar faltantes: web.",
		EvidenceRefs: []string{
			"review-rework-missing-web",
		},
	}, "programacion")
	if err != nil {
		t.Fatalf("agentTaskV0: %v", err)
	}
	for _, want := range []string{
		"Contexto de revision/rework",
		"completar faltantes: web",
		"conserva lo valido",
	} {
		if !strings.Contains(got.Objective, want) {
			t.Fatalf("objective no contiene %q:\n%s", want, got.Objective)
		}
	}
}

func TestProgrammingTaskV0EspecializaObjetivoPorWorkProfile(t *testing.T) {
	for _, tc := range []struct {
		name          string
		kind          orquestacoreworkflow.WorkProfileKindV0
		wantObjective string
		reject        string
	}{
		{
			name:          "code_study",
			kind:          orquestacoreworkflow.WorkProfileCodeStudyV0,
			wantObjective: "Estudia el codigo",
			reject:        "Implementa solo",
		},
		{
			name:          "implementation",
			kind:          orquestacoreworkflow.WorkProfileImplementationV0,
			wantObjective: "Implementa solo",
		},
		{
			name:          "refactor",
			kind:          orquestacoreworkflow.WorkProfileRefactorV0,
			wantObjective: "Refactoriza solo",
			reject:        "Implementa solo",
		},
		{
			name:          "required_tests",
			kind:          orquestacoreworkflow.WorkProfileRequiredTestsV0,
			wantObjective: "pruebas requeridas",
			reject:        "Implementa solo",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := orquestacoreworkflow.WorkProfileV0{
				SchemaVersion: orquestacoreworkflow.WorkProfileSchemaVersionV0,
				ProfileRef:    "profile-ref-" + tc.name,
				ProfileKind:   tc.kind,
				TaskRef:       "task-ref-" + tc.name,
				RunRef:        "run-ref-" + tc.name,
				PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
				Title:         "Trabajo " + tc.name,
				Objective:     "Resolver perfil " + tc.name + ".",
				ScopeRefs:     []string{"internal/" + tc.name},
				FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
					ContractRef:  "contract:function:" + tc.name + ":v0",
					FunctionName: "Do" + strings.ReplaceAll(tc.name, "_", ""),
				}},
			}
			if tc.kind == orquestacoreworkflow.WorkProfileImplementationV0 ||
				tc.kind == orquestacoreworkflow.WorkProfileRefactorV0 ||
				tc.kind == orquestacoreworkflow.WorkProfileRequiredTestsV0 {
				profile.RequiredTests = []string{"go test ./..."}
			}
			task, err := orquestacoreworkflow.WorkflowTaskFromWorkProfileV0(profile)
			if err != nil {
				t.Fatalf("WorkflowTaskFromWorkProfileV0: %v", err)
			}
			resolver := CodexLaunchSpecResolverV0{
				TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
			}

			got, err := resolver.agentTaskV0(context.Background(), orquestaruntime.LaunchRuntimeAgentRequestV0{
				RunID:   task.RunID,
				PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskRef: task.TaskID,
			}, "programacion")
			if err != nil {
				t.Fatalf("agentTaskV0: %v", err)
			}
			if got.TaskRef != task.TaskID ||
				len(got.WriteSet) != 1 ||
				got.WriteSet[0] != profile.ScopeRefs[0] ||
				!strings.Contains(got.Objective, tc.wantObjective) {
				t.Fatalf("perfil no propagado: task=%+v objective=%s", got, got.Objective)
			}
			if tc.reject != "" && strings.Contains(got.Objective, tc.reject) {
				t.Fatalf("objective demasiado generico para %s: %s", tc.name, got.Objective)
			}
			if len(profile.RequiredTests) > 0 &&
				(len(got.RequiredTests) != 1 || got.RequiredTests[0] != "go test ./...") {
				t.Fatalf("required_tests no propagados: %+v", got.RequiredTests)
			}
		})
	}
}

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
	cases := []struct {
		name          string
		unit          orquestaappplanner.AppWorkUnitV0
		wantKind      orquestacoreworkflow.WorkProfileKindV0
		wantObjective string
		reject        string
	}{
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
		})
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
