package orquestaappcodexstack

import (
	"context"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestCodexStackAutoprogrammingPrepareRunV0CreaRetrySiRunPrevioEstaAtascado(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	queue := orquestarunmemory.NewRunMemoryStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-stale-001"
	stale := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         request.RequestRef,
		ProjectRef:    request.ProjectRef,
		AppSpecRef:    "app-spec-ref-autoprogramming-stale-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-ref-autoprogramming-stale-001"},
		StartedAgents: []string{"agent-ref-autoprogramming-stale-001"},
		LostAgents:    []string{"agent-ref-autoprogramming-stale-001"},
	}
	if err := runStore.SaveRunV0(ctx, stale); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        stale.RunID,
		QueueRef:      "queue-main",
		AppRef:        stale.ProjectRef,
		PriorityScore: 10,
		RequestedBy:   "test",
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 stale: %v", err)
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
		RunQueue: RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
	}
	executor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		stack,
		"2026-05-24T01:40:00Z",
		"orquesta-test",
		queue,
		stack.RunQueue,
		nil,
		"",
	)

	result, err := executor.Execute(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-" + request.RequestRef,
		OccurredAt:             "2026-05-24T01:40:00Z",
		RequestedBy:            "orquesta-test",
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
	if _, err := runStore.LoadRunV0(ctx, stale.RunID); err != nil {
		t.Fatalf("run viejo debe conservarse: %v", err)
	}
	candidates, err := queue.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "queue-main"})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(candidates) != 1 ||
		candidates[0].RunRef != result.RunRef ||
		!autoprogrammingBridgeStringInSetForTestV0(candidates[0].EvidenceRefs, "evidence-ref-autoprogramming-prepare-run-enqueued") {
		t.Fatalf("candidates=%+v result=%+v", candidates, result)
	}
}

func TestCodexStackAutoprogrammingPrepareRunV0CreaRetrySiAssessmentTerminalQuedoObsoleto(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	queue := orquestarunmemory.NewRunMemoryStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-assessment-stale-001"
	agentRef := "agent-ref-autoprogramming-assessment-stale-001"
	ghostAgentRef := "agent-ref-autoprogramming-assessment-stale-ghost-001"
	runtimeWorkDir := t.TempDir()
	stale := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         request.RequestRef,
		ProjectRef:    request.ProjectRef,
		AppSpecRef:    "app-spec-ref-autoprogramming-assessment-stale-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-ref-autoprogramming-assessment-stale-001"},
		StartedAgents: []string{agentRef, ghostAgentRef},
		StoppedAgents: []string{agentRef},
		AgentAssessments: []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-autoprogramming-assessment-stale-001",
			PhaseID:        string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			AgentRequestID: agentRef,
			TaskRef:        "task-ref-autoprogramming-assessment-stale-001",
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionStopAgentV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		})},
	}
	if err := runStore.SaveRunV0(ctx, stale); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        stale.RunID,
		QueueRef:      "queue-main",
		AppRef:        stale.ProjectRef,
		PriorityScore: 10,
		RequestedBy:   "test",
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 stale: %v", err)
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
		RunQueue: RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
	}
	executor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		stack,
		"2026-05-24T02:20:00Z",
		"orquesta-test",
		queue,
		stack.RunQueue,
		nil,
		runtimeWorkDir,
	)

	result, err := executor.Execute(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-" + request.RequestRef,
		OccurredAt:             "2026-05-24T02:20:00Z",
		RequestedBy:            "orquesta-test",
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
	candidates, err := queue.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "queue-main"})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(candidates) != 1 || candidates[0].RunRef != result.RunRef {
		t.Fatalf("candidates=%+v result=%+v", candidates, result)
	}
}

func TestCodexStackAutoprogrammingPrepareRunV0CreaRetrySiLaunchFalloSinStartedAgent(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	queue := orquestarunmemory.NewRunMemoryStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-launch-blocked-001"
	stale := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         request.RequestRef,
		ProjectRef:    request.ProjectRef,
		AppSpecRef:    "app-spec-ref-autoprogramming-launch-blocked-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-ref-autoprogramming-launch-blocked-001"},
		FailedAgents:  []string{"agent-ref-autoprogramming-launch-blocked-001"},
	}
	if err := runStore.SaveRunV0(ctx, stale); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        stale.RunID,
		QueueRef:      "queue-main",
		AppRef:        stale.ProjectRef,
		PriorityScore: 10,
		RequestedBy:   "test",
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 stale: %v", err)
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
		RunQueue: RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
	}
	executor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		stack,
		"2026-05-24T02:10:00Z",
		"orquesta-test",
		queue,
		stack.RunQueue,
		nil,
		"",
	)

	result, err := executor.Execute(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-" + request.RequestRef,
		OccurredAt:             "2026-05-24T02:10:00Z",
		RequestedBy:            "orquesta-test",
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
	candidates, err := queue.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "queue-main"})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(candidates) != 1 || candidates[0].RunRef != result.RunRef {
		t.Fatalf("candidates=%+v result=%+v", candidates, result)
	}
}
