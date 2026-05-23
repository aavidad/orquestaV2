package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

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
}
