package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestProgrammingTaskV0PreservaWorktreeAisladaYRamaOpaca(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             "task-autoprogramming-scope-001",
		RunID:              "run-autoprogramming-scope-001",
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Cambio acotado",
		Summary:            "Autoprogramacion acotada.",
		WriteSet:           []string{"modulos/orquesta-app-codex-stack"},
		AcceptanceCriteria: []string{"refs de tarea origen preservadas"},
		ContextRefs: []string{
			"request_ref:run-autoprogramming-scope-001",
			"worktree_ref:worktree-ref-isolated-scope-001",
			"branch_ref:branch-ref-opaque-scope-001",
		},
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
	for _, want := range []string{
		"worktree aislada validada",
		"worktree_ref=worktree-ref-isolated-scope-001",
		"branch_ref=branch-ref-opaque-scope-001",
		"refs opacas",
	} {
		if !strings.Contains(got.Objective, want) {
			t.Fatalf("objective no conserva %q:\n%s", want, got.Objective)
		}
	}
	if !codexStackDoneCriteriaContainsForTestV0(got.DoneCriteria, "branch_ref opaco preservados") {
		t.Fatalf("done_criteria=%v", got.DoneCriteria)
	}
	for _, want := range []string{
		"worktree_ref=worktree-ref-isolated-scope-001",
		"branch_ref=branch-ref-opaque-scope-001",
		"no las conviertas en rutas ni nombres Git",
	} {
		if !codexStackDoneCriteriaContainsForTestV0(got.DoneCriteria, want) {
			t.Fatalf("done_criteria no conserva %q: %v", want, got.DoneCriteria)
		}
	}
}

func TestProgrammingTaskV0PreservaRefsOpacasDelPaqueteAutoprogramacion(t *testing.T) {
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        "task-autoprogramming-cb6278a91f63-g01",
		RunID:         "run-autoprog2-t01-cierre-autonomo-cola",
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Cambio acotado 01",
		Summary:       "Autoprogramacion acotada con worktree aislada.",
		WriteSet:      []string{"modulos/orquesta-app-codex-stack"},
		AcceptanceCriteria: []string{
			"worktree_ref y branch_ref preservadas como refs opacas",
		},
		ContextRefs: []string{
			"worktree_ref:worktree-ref-run-autoprog2-t01-cierre-autonomo-cola",
			"branch_ref:branch-ref-run-autoprog2-t01-cierre-autonomo-cola",
		},
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

	for _, want := range []string{
		"worktree_ref=worktree-ref-run-autoprog2-t01-cierre-autonomo-cola",
		"branch_ref=branch-ref-run-autoprog2-t01-cierre-autonomo-cola",
		"refs opacas",
		"sin convertirlas en rutas ni nombres Git",
	} {
		if !strings.Contains(got.Objective, want) {
			t.Fatalf("objective no conserva %q:\n%s", want, got.Objective)
		}
	}
}

func TestPrepareAutoprogrammingRunV0PreservaRefsOpacasDelPaquete(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	request := AutoprogrammingBridgeRequestV0{
		Request: orquestaautoprogramming.AutoprogrammingRequestV0{
			RequestRef:       "run-autoprog2-t07-smoke-desatendido-real",
			ProjectRef:       "project-ref-orquesta-autoprogramming",
			WorktreeRef:      "worktree-ref-run-autoprog2-t07-smoke-desatendido-real",
			WorktreeIsolated: true,
			BranchRef:        "branch-ref-run-autoprog2-t07-smoke-desatendido-real",
			Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{{
				TaskRef:            "task-autoprogramming-71254a25836a-g01",
				Area:               "orquesta-app-stack-programacion",
				Objective:          "Ejecutar contrato explicito sin inferir por task_ref.",
				Context:            []string{"paquete de agente conserva acceptance criteria y reglas compactas"},
				AcceptanceCriteria: []string{"objetivo explicito llega al agente"},
				CompactRules:       []string{"worktree_ref y branch_ref son refs opacas"},
			}},
			WriteSet: []string{
				"modulos/orquesta-app-codex-stack",
				"cmd/orquesta-server",
				"modulos/orquesta-run-file",
				"modulos/orquesta-state-file",
				"docs",
			},
			RequiredTests: []string{
				"go test -count=1 ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server ./modulos/orquesta-run-file ./modulos/orquesta-state-file",
			},
		},
		OccurredAt:    "2026-05-23T00:00:00Z",
		CorrelationID: "corr-run-autoprog2-t07-smoke-desatendido-real-burst-002",
		RequestedBy:   "orquesta-v2",
	}
	result, err := PrepareAutoprogrammingRunV0(context.Background(), request, orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:          runStore,
		DirectorTaskStore: taskStore,
	})
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if !result.Accepted || len(result.Tasks) != 1 {
		t.Fatalf("result=%+v", result)
	}
	task := result.Tasks[0]
	for _, want := range []string{
		"worktree_ref:worktree-ref-run-autoprog2-t07-smoke-desatendido-real",
		"branch_ref:branch-ref-run-autoprog2-t07-smoke-desatendido-real",
	} {
		if !autoprogrammingBridgeStringInSetForTestV0(task.ContextRefs, want) {
			t.Fatalf("context_refs=%v want=%s", task.ContextRefs, want)
		}
	}
	agentTask, err := (CodexLaunchSpecResolverV0{TaskStore: taskStore}).agentTaskV0(
		context.Background(),
		orquestaruntime.LaunchRuntimeAgentRequestV0{
			RunID:   result.Run.RunID,
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef: task.TaskID,
		},
		"programacion",
	)
	if err != nil {
		t.Fatalf("agentTaskV0: %v", err)
	}
	for _, want := range []string{
		"Ejecutar contrato explicito",
		"paquete de agente conserva acceptance criteria",
		"worktree_ref=worktree-ref-run-autoprog2-t07-smoke-desatendido-real",
		"branch_ref=branch-ref-run-autoprog2-t07-smoke-desatendido-real",
		"refs opacas",
	} {
		if !strings.Contains(agentTask.Objective, want) {
			t.Fatalf("objective no contiene %q:\n%s", want, agentTask.Objective)
		}
	}
	if len(agentTask.WriteSet) != len(request.Request.WriteSet) ||
		agentTask.RequiredTests[0] != request.Request.RequiredTests[0] {
		t.Fatalf("agent_task=%+v", agentTask)
	}
	for _, want := range []string{
		"objetivo explicito llega al agente",
		"worktree_ref y branch_ref son refs opacas",
	} {
		if !codexStackDoneCriteriaContainsForTestV0(agentTask.DoneCriteria, want) {
			t.Fatalf("done_criteria no contiene %q: %v", want, agentTask.DoneCriteria)
		}
	}
}
