package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type prepareRunAuthorityOrderingQueueWriterForTestV0 struct {
	delegate            orquestarunqueue.RunQueuePriorityWriterPortV0
	manifestStore       orquestaautoprogramming.AutoprogrammingIntentManifestStorePortV0
	expectedManifestRef string
	observedBeforeMark  bool
}

func (writer *prepareRunAuthorityOrderingQueueWriterForTestV0) SetRunPriorityV0(ctx context.Context, command orquestarunqueue.RunQueuePriorityCommandV0) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	if command.Reason == "autoprogramming_stale_run_retried" {
		if _, err := writer.manifestStore.LoadAutoprogrammingIntentManifestV0(ctx, writer.expectedManifestRef); err != nil {
			return orquestarunqueue.RunSchedulingCandidateV0{}, fmt.Errorf("stale mark before durable authority: %w", err)
		}
		writer.observedBeforeMark = true
	}
	return writer.delegate.SetRunPriorityV0(ctx, command)
}

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
	intentStore := orquestaautoprogramming.NewInMemoryAutoprogrammingIntentManifestStoreV0()
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:          runStore,
			DirectorTaskStore: taskStore,
		},
		Stores: StoresV0{
			RunStore:                           runStore,
			TaskStore:                          taskStore,
			RunQueue:                           queue,
			AutoprogrammingIntentManifestStore: intentStore,
		},
		Codex:                         CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()},
		RunQueue:                      RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
		AllowLegacyAutoprogrammingRun: true,
	}
	orderingWriter := &prepareRunAuthorityOrderingQueueWriterForTestV0{
		delegate:            queue,
		manifestStore:       intentStore,
		expectedManifestRef: autoprogrammingPrepareRetryRefV0(request.RequestRef, "2026-05-24T01:40:00Z", time.Time{}),
	}
	executor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		stack,
		"2026-05-24T01:40:00Z",
		"orquesta-test",
		orderingWriter,
		stack.RunQueue,
		nil,
		"",
	)

	result, err := executor.Execute(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-" + request.RequestRef,
		OccurredAt:             "2026-05-24T01:40:00Z",
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
	if !orderingWriter.observedBeforeMark {
		t.Fatal("stale candidate marked before claim/manifest authority")
	}
	retryManifest, err := intentStore.LoadAutoprogrammingIntentManifestV0(ctx, result.RunRef)
	if err != nil {
		t.Fatal(err)
	}
	retryContent, retryIssues := orquestaautoprogramming.ProjectAutoprogrammingIntentManifestContentV0(retryManifest)
	if len(retryIssues) != 0 || retryContent.PrepareRunEnvelope == nil ||
		!autoprogrammingBridgeStringInSetForTestV0(retryContent.Request.Tasks[0].ContextRefs, "previous_run_ref:"+stale.RunID) ||
		!autoprogrammingBridgeStringInSetForTestV0(retryContent.Request.Tasks[0].ContextRefs, "retry_attempt:01") {
		t.Fatalf("retry content=%+v issues=%+v", retryContent, retryIssues)
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

func TestAutoprogrammingPrepareRunRetryV0UsesStableNeutralPublicMessage(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-retry-message-001"
	if err := runStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         request.RequestRef,
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		FailedAgents:  []string{"agent-ref-autoprogramming-retry-message-001"},
	}); err != nil {
		t.Fatal(err)
	}
	executor := CodexStackAutoprogrammingPrepareRunExecutorV0{Stack: &StackV0{Stores: StoresV0{RunStore: runStore}}}
	plan := executor.planFreshAttemptForStaleAutoprogrammingRunV0(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID: request.RequestRef, AutoprogrammingRequest: request,
	})
	const key = "autoprogramming_prepare_run_retry_occurred_at_required"
	if len(plan.Issues) != 1 || plan.Issues[0].Code != key || plan.Issues[0].Message != key || plan.Issues[0].Field != "occurred_at" {
		t.Fatalf("issues=%+v", plan.Issues)
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
			RunStore:                           runStore,
			TaskStore:                          taskStore,
			RunQueue:                           queue,
			AutoprogrammingIntentManifestStore: orquestaautoprogramming.NewInMemoryAutoprogrammingIntentManifestStoreV0(),
		},
		Codex:                         CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()},
		RunQueue:                      RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
		AllowLegacyAutoprogrammingRun: true,
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
			RunStore:                           runStore,
			TaskStore:                          taskStore,
			RunQueue:                           queue,
			AutoprogrammingIntentManifestStore: orquestaautoprogramming.NewInMemoryAutoprogrammingIntentManifestStoreV0(),
		},
		Codex:                         CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()},
		RunQueue:                      RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
		AllowLegacyAutoprogrammingRun: true,
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
	candidates, err := queue.ListRunSchedulingCandidatesV0(ctx, orquestarunqueue.RunQueueReadRequestV0{QueueRef: "queue-main"})
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if len(candidates) != 1 || candidates[0].RunRef != result.RunRef {
		t.Fatalf("candidates=%+v result=%+v", candidates, result)
	}
}

func TestCodexStackAutoprogrammingPrepareRunV0NoCreaRetrySiHayProcesoVivoRegistrado(t *testing.T) {
	ctx := context.Background()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	queue := orquestarunmemory.NewRunMemoryStoreV0()
	processRegistry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	runtime := newFakeCodexStackRuntimeV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequestRef = "run-autoprogramming-live-process-guard-001"
	stack := &StackV0{
		Ports: orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:          runStore,
			DirectorTaskStore: taskStore,
		},
		Stores: StoresV0{
			RunStore:                           runStore,
			TaskStore:                          taskStore,
			RunQueue:                           queue,
			ProcessRegistry:                    processRegistry,
			AutoprogrammingIntentManifestStore: orquestaautoprogramming.NewInMemoryAutoprogrammingIntentManifestStoreV0(),
		},
		Codex:                         CodexRuntimeConfigV0{ProjectWorkDir: t.TempDir()},
		CodexSnapshotSource:           runtime,
		RunQueue:                      RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
		AllowLegacyAutoprogrammingRun: true,
	}
	executor := NewCodexStackAutoprogrammingPrepareRunExecutorV0(
		stack,
		"2026-05-24T02:25:00Z",
		"orquesta-test",
		queue,
		stack.RunQueue,
		nil,
		"",
	)
	first, err := executor.Execute(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-" + request.RequestRef,
		OccurredAt:             "2026-05-24T02:25:00Z",
		RequestedBy:            "orquesta-test",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: request,
		MaxBursts:              4,
		MaxCommands:            8,
		MaxOutboxPerCycle:      8,
		MaxDispatchesPerWait:   4,
	})
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	if !first.Accepted || first.RunRef != request.RequestRef || len(first.WaitAgentRefs) == 0 {
		t.Fatalf("first=%+v", first)
	}
	stale, err := runStore.LoadRunV0(ctx, first.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 first: %v", err)
	}
	agentRef := first.WaitAgentRefs[0]
	stale.StartedAgents = []string{agentRef}
	stale.LostAgents = []string{agentRef}
	if err := runStore.SaveRunV0(ctx, stale); err != nil {
		t.Fatalf("SaveRunV0 stale: %v", err)
	}
	processRef := "process-ref-autoprogramming-live-process-guard-001"
	if err := processRegistry.RecordAgentProcessV0(ctx, orquestacionnucleoapp.AgentProcessRegistryRecordV0{
		RunID:          stale.RunID,
		AgentRequestID: agentRef,
		ProcessRef:     processRef,
		SessionRef:     "session-ref-autoprogramming-live-process-guard-001",
		LaunchRef:      "launch-ref-autoprogramming-live-process-guard-001",
		ReadinessRef:   "readiness-ref-autoprogramming-live-process-guard-001",
		EvidenceRefs:   []string{"evidence-ref-autoprogramming-live-process-guard"},
	}); err != nil {
		t.Fatalf("RecordAgentProcessV0: %v", err)
	}
	runtime.snapshots[processRef] = orquestaruntime.ProcessRuntimeSnapshotV0{
		SchemaVersion: orquestaruntime.ProcessRuntimeConnectorVersionV0,
		ProcessRef:    processRef,
		SessionRef:    "session-ref-autoprogramming-live-process-guard-001",
		LaunchRef:     "launch-ref-autoprogramming-live-process-guard-001",
		Status:        orquestaruntime.ProcessRuntimeRunningV0,
	}

	result, err := executor.Execute(ctx, orquestamcp.MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              request.RequestRef,
		CorrelationID:          "corr-" + request.RequestRef,
		OccurredAt:             "2026-05-24T02:30:00Z",
		RequestedBy:            "orquesta-test",
		DirectorExecutionMode:  orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AutoprogrammingRequest: request,
		MaxBursts:              4,
		MaxCommands:            8,
		MaxOutboxPerCycle:      8,
		MaxDispatchesPerWait:   4,
	})
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if !result.Accepted ||
		result.RunRef != stale.RunID ||
		strings.Contains(result.RunRef, "-retry-") {
		t.Fatalf("result=%+v stale=%s", result, stale.RunID)
	}
}
