package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
)

func TestCodexStackAutoprogrammingPrepareRunV0NoReintentaBacklogScannerFallido(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         "request-ref-autoprogramming-backlog-scanner-7e8a6ae9",
		ProjectRef:    "project-ref-autoprogramming-bridge-001",
		AppSpecRef:    "app-spec-ref-autoprogramming-backlog-scanner",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-autoprogramming-4f550c019e0d-g01"},
		FailedAgents:  []string{"agent-ref-autoprogramming-backlog-scanner"},
	}
	if AutoprogrammingRunNeedsFreshAttemptV0(run) ||
		AutoprogrammingRunNeedsFreshAttemptWithRuntimeV0(run, t.TempDir()) {
		t.Fatalf("scanner documental no debe generar retry automatico: %+v", run)
	}
}

func TestCodexStackAutoprogrammingPrepareRunV0ReintentaScannerParadoPorControlV0(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	queue := orquestarunmemory.NewRunMemoryStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "request-ref-autoprogramming-backlog-scanner-15eeecb9"
	stale := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         request.RequestRef,
		ProjectRef:    request.ProjectRef,
		AppSpecRef:    "app-spec-ref-autoprogramming-backlog-scanner",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-autoprogramming-backlog-scanner-g01"},
		StartedAgents: []string{"agent-ref-autoprogramming-backlog-scanner"},
		LostAgents:    []string{"agent-ref-autoprogramming-backlog-scanner"},
	}
	if err := runStore.SaveRunV0(ctx, stale); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	if _, err := queue.PutRunControlStateV0(ctx, orquestaruncontrol.RunControlStateV0{
		RunRef: stale.RunID,
		Status: orquestaruncontrol.RunControlStatusStoppedV0,
	}); err != nil {
		t.Fatalf("PutRunControlStateV0: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:          runStore,
			DirectorTaskStore: taskStore,
		},
		Stores: StoresV0{
			RunStore:   runStore,
			TaskStore:  taskStore,
			RunQueue:   queue,
			RunControl: queue,
		},
		Codex:                         CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()},
		RunQueue:                      RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
		AllowLegacyAutoprogrammingRun: true,
	}
	executor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		stack,
		"2026-05-27T07:30:00Z",
		"orquesta-test",
		queue,
		stack.RunQueue,
		nil,
		"",
	)

	result, err := executor.Execute(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-" + request.RequestRef,
		OccurredAt:             "2026-05-27T07:30:00Z",
		RequestedBy:            "orquesta-test",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: request,
		MaxBursts:              4,
		MaxCommands:            8,
		MaxOutboxPerCycle:      8,
		MaxDispatchesPerWait:   4,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Accepted ||
		result.RunRef == stale.RunID ||
		!strings.Contains(result.RunRef, "-retry-") {
		t.Fatalf("result=%+v stale=%s", result, stale.RunID)
	}
}

func TestCodexStackAutoprogrammingPrepareRunV0ReencuadraRetryRepetido(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	queue := orquestarunmemory.NewRunMemoryStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-stale-002-retry-deadbeefdead"
	stale := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         request.RequestRef,
		ProjectRef:    request.ProjectRef,
		AppSpecRef:    "app-spec-ref-autoprogramming-stale-002",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-ref-autoprogramming-stale-002"},
		StartedAgents: []string{"agent-ref-autoprogramming-stale-002"},
		FailedAgents:  []string{"agent-ref-autoprogramming-stale-002"},
	}
	if err := runStore.SaveRunV0(ctx, stale); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:          runStore,
			DirectorTaskStore: taskStore,
		},
		Stores: StoresV0{
			RunStore:  runStore,
			TaskStore: taskStore,
			RunQueue:  queue,
		},
		Codex:                         CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()},
		RunQueue:                      RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
		AllowLegacyAutoprogrammingRun: true,
	}
	executor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		stack,
		"2026-05-24T01:50:00Z",
		"orquesta-test",
		queue,
		stack.RunQueue,
		nil,
		"",
	)

	result, err := executor.Execute(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-" + request.RequestRef,
		OccurredAt:             "2026-05-24T01:50:00Z",
		RequestedBy:            "orquesta-test",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: request,
		MaxBursts:              4,
		MaxCommands:            8,
		MaxOutboxPerCycle:      8,
		MaxDispatchesPerWait:   4,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Accepted || result.RunRef == stale.RunID {
		t.Fatalf("result=%+v stale=%s", result, stale.RunID)
	}
	run, err := runStore.LoadRunV0(ctx, result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 retry: %v", err)
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0 retry: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks=%+v", tasks)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(tasks[0].ContextRefs, "retry_strategy:reframe-02") ||
		!autoprogrammingBridgeStringInSetForTestV0(tasks[0].ContextRefs, "previous_run_ref:"+stale.RunID) ||
		!autoprogrammingBridgeStringContainsForTestV0(tasks[0].AcceptanceCriteria, "No repetir literalmente el intento fallido") ||
		!autoprogrammingBridgeStringInSetForTestV0(tasks[0].AcceptanceCriteria, "Regla compacta source-task-ref-autoprogramming-bridge-001: retry_reframed_attempt:02") {
		t.Fatalf("retry repetido no reencuadrado: task=%+v", tasks[0])
	}
}
