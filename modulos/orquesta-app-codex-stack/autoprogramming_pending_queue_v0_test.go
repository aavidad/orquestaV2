package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCodexStackAutoprogrammingPrepareRunAPIV0NoRelanzaAgentePendienteEnTicksGlobalesV0(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-pending-queue-api-001",
		CorrelationID:          "corr-autoprogramming-pending-queue-api-001",
		OccurredAt:             "2026-05-22T11:35:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: autoprogrammingBridgeRequestForTestV0(),
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if !prepared.Accepted || prepared.RunRef == "" || len(prepared.WaitAgentRefs) != 1 {
		t.Fatalf("prepared=%+v", prepared)
	}

	first, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0 first: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(first); len(got) != 1 ||
		got[0] != prepared.RunRef || runtime.launchCountV0() != 1 {
		t.Fatalf("first executions=%v launches=%d result=%+v", got, runtime.launchCountV0(), first)
	}

	second, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0 second: %v", err)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if runtime.launchCountV0() != 1 ||
		len(compactStringsV0(run.StartedAgents)) != 1 ||
		!autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, prepared.WaitAgentRefs[0]) ||
		!drainRunHasPendingExternalAgentsV0(run, nil) {
		t.Fatalf("second=%+v run=%+v launches=%d wait=%v", second, run, runtime.launchCountV0(), prepared.WaitAgentRefs)
	}
}

func TestCodexStackAutoprogrammingRunGlobalTickV0RelanzaFronteraDependienteTrasACKV0(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	request := autoprogrammingBridgeDependentRequestForTestV0()

	prepared := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-dependent-queue-api-001",
		CorrelationID:          "corr-autoprogramming-dependent-queue-api-001",
		OccurredAt:             "2026-06-19T10:15:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if !prepared.Accepted || prepared.RunRef == "" || len(prepared.WaitAgentRefs) != 1 || len(prepared.WorkflowTaskRefs) != 2 {
		t.Fatalf("prepared=%+v", prepared)
	}
	tasks, err := stack.Ports.DirectorTaskStore.LoadWorkflowTasksV0(
		context.Background(),
		prepared.RunRef,
		prepared.WorkflowTaskRefs,
	)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	baseTaskRef := tasks[0].TaskID
	dependentTaskRef := tasks[1].TaskID
	baseAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(baseTaskRef)
	dependentAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(dependentTaskRef)
	if prepared.WaitAgentRefs[0] != baseAgentRef {
		t.Fatalf("wait_agent_refs=%v base_agent=%s tasks=%+v", prepared.WaitAgentRefs, baseAgentRef, tasks)
	}

	first, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0 first: %v", err)
	}
	if got := executionRefsForStackCoordinatorTestV0(first); len(got) != 1 ||
		got[0] != prepared.RunRef || runtime.launchCountV0() != 1 {
		t.Fatalf("first executions=%v launches=%d result=%+v", got, runtime.launchCountV0(), first)
	}
	baseDescriptor := mustCodexStackDescriptorByTaskRefV0(t, stack, baseTaskRef)
	if err := writeCodexStackAckForDescriptorV0(t, baseDescriptor); err != nil {
		t.Fatalf("write base ack: %v", err)
	}

	second, err := stack.RunGlobalTickV0(context.Background(), globalTickCommandForTestV0())
	if err != nil {
		t.Fatalf("RunGlobalTickV0 second: %v", err)
	}
	run, err := stack.Ports.RunStore.LoadRunV0(context.Background(), prepared.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.DeliveredTasks, baseTaskRef) ||
		!autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, dependentAgentRef) ||
		runtime.launchCountV0() != 2 {
		t.Fatalf("frontera no relanzada: second=%+v delivered=%v started=%v launches=%d base=%s dep_agent=%s",
			second,
			run.DeliveredTasks,
			run.StartedAgents,
			runtime.launchCountV0(),
			baseTaskRef,
			dependentAgentRef,
		)
	}

	replayed := postAutoprogrammingPrepareRunStackV0(t, stack, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              "request-autoprogramming-dependent-queue-api-001",
		CorrelationID:          "corr-autoprogramming-dependent-queue-api-001",
		OccurredAt:             "2026-06-19T10:16:00Z",
		RequestedBy:            "orquesta-stack-api-test",
		AutoprogrammingRequest: request,
		MaxBursts:              3,
		MaxStepsPerBurst:       3,
		MaxDispatchesPerWait:   3,
		MaxCommands:            5,
		MaxOutboxPerCycle:      5,
	})
	if len(replayed.WaitAgentRefs) != 1 || replayed.WaitAgentRefs[0] != dependentAgentRef {
		t.Fatalf("replay wait_agent_refs=%v want %s", replayed.WaitAgentRefs, dependentAgentRef)
	}
}

func autoprogrammingBridgeDependentRequestForTestV0() orquestaautoprogramming.AutoprogrammingRequestV0 {
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-dependent-bridge-001"
	request.ProjectRef = "project-ref-autoprogramming-dependent-bridge-001"
	request.WorktreeRef = "worktree-ref-autoprogramming-dependent-bridge-001"
	request.BranchRef = "branch-ref-autoprogramming-dependent-bridge-001"
	request.Tasks = []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{
		{
			TaskRef:  "source-task-ref-autoprogramming-dependent-base",
			Area:     "domain",
			Title:    "Base dominio",
			WriteSet: []string{"internal/domain/model.go"},
		},
		{
			TaskRef:   "source-task-ref-autoprogramming-dependent-adapter",
			Area:      "adapter",
			Title:     "Adaptador",
			WriteSet:  []string{"internal/adapter/repository.go"},
			DependsOn: []string{"source-task-ref-autoprogramming-dependent-base"},
		},
	}
	request.WriteSet = []string{
		"internal/domain/model.go",
		"internal/adapter/repository.go",
	}
	return request
}
