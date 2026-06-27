package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexStackAutoprogrammingExecutorV0MaterializaWorktreePorTask(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	executor := NewCodexStackAutoprogrammingExecutorV0(&stack)

	result, err := executor.Execute(context.Background(), AutoprogrammingBridgeRequestV0{
		Request:               autoprogrammingBridgeRequestForTestV0(),
		OccurredAt:            "2026-05-22T11:00:00Z",
		CorrelationID:         "corr-autoprogramming-worktree-isolation-001",
		RequestedBy:           "orquesta-stack-executor-test",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Accepted || len(result.Tasks) != 1 {
		t.Fatalf("result=%+v", result)
	}
	storedTasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(
		context.Background(),
		result.Run.RunID,
		result.Run.Tasks,
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(storedTasks) != 1 ||
		!autoprogrammingBridgeStringContainsForTestV0(storedTasks[0].ContextRefs, autoprogrammingWorktreeIsolationRefPrefixV0) ||
		!autoprogrammingBridgeStringContainsForTestV0(storedTasks[0].ContextRefs, autoprogrammingWorktreeBaselineRefPrefixV0) ||
		!autoprogrammingBridgeStringContainsForTestV0(storedTasks[0].ContextRefs, autoprogrammingWorktreeIsolationEvidencePrefixV0) {
		t.Fatalf("worktree no materializada en task: %+v", storedTasks)
	}
}

func TestCodexStackAutoprogrammingExecutorV0BloqueaBranchRefNoOpacaAntesDeEncolar(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	executor := NewCodexStackAutoprogrammingExecutorV0(&stack)
	request := autoprogrammingBridgeRequestForTestV0()
	request.BranchRef = "trabajo/plataforma-agentes"

	result, err := executor.Execute(context.Background(), AutoprogrammingBridgeRequestV0{
		Request:               request,
		OccurredAt:            "2026-05-22T11:05:00Z",
		CorrelationID:         "corr-autoprogramming-stack-executor-branch-001",
		RequestedBy:           "orquesta-stack-executor-test",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Accepted || len(result.Issues) == 0 {
		t.Fatalf("result=%+v", result)
	}
	if result.Issues[0].Code != "worktree_isolation_invalid" ||
		result.Issues[0].Field != "branch_ref" {
		t.Fatalf("issues=%+v", result.Issues)
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: stack.RunQueue.QueueRef},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("run encolada pese a bloqueo publico: %+v", candidates)
	}
}

func TestCodexLaunchSpecResolverV0BloqueaWorktreeNoMaterializableAntesDeRuntime(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	request := autoprogrammingBridgeRequestForTestV0()
	request.BranchRef = "trabajo/plataforma-agentes"
	work := orquestaautoprogramming.BuildAutoprogrammingProgrammableWorkV0(request)
	if !work.Accepted || len(work.Work.Tasks) != 1 {
		t.Fatalf("work=%+v", work)
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(context.Background(), work.Work.Tasks[0]); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	resolver := CodexLaunchSpecResolverV0{
		Config:    stack.Codex,
		TaskStore: stack.Stores.TaskStore,
	}
	_, err := resolver.ResolveExternalAgentLaunchSpecV0(
		context.Background(),
		orquestaruntime.AgentLauncherInboundV0{
			CorrelationID: "corr-worktree-guard",
			Payload: &orquestaruntime.LaunchRuntimeAgentRequestV0{
				RunID:          request.RequestRef,
				PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskRef:        work.Work.Tasks[0].TaskID,
				AgentRequestID: "agent-ref-worktree-guard-001",
				Role:           "implementacion",
			},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "worktree_isolation_invalid") {
		t.Fatalf("err=%v", err)
	}
}
