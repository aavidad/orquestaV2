package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestCodexStackSelfImprovementAPIV0AutoPrepareRunEncola(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPAutoprogrammingSelfImprovementToolInputV0{
		RequestID:      "request-ref-self-improvement-stack-001",
		AutoPrepareRun: true,
		Proposal: orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{
			ProjectRef:        "project-ref-autoprogramming-bridge-001",
			WorktreeRef:       "worktree-ref-autoprogramming-bridge-001",
			WorktreeIsolated:  true,
			BranchRef:         "branch-ref-autoprogramming-bridge-001",
			ObservedBy:        "director",
			FailureSummary:    "Mejora reutilizable detectada por el stack.",
			SuggestedArea:     "app-codex-stack",
			SuggestedWriteSet: []string{"modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go"},
			RequiredTests:     []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestPrepareAutoprogrammingRunV0"},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/autoprogramming/self-improvement", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPAutoprogrammingSelfImprovementToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.PreparedRun == nil || !result.PreparedRun.Accepted || result.PreparedRun.RunRef == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestPrepareAutoprogrammingRunV0PersisteWorkflowTasksYRunContinuable(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		OperationalPlanStateStore:  planStore,
		OperationalPlanStateWriter: planStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			autoprogrammingBridgeCapacityDispatcherForTestV0(runStore, eventSink, ledger),
			autoprogrammingBridgeAgentDispatcherForTestV0(runStore, eventSink, ledger),
		},
	}

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request:       autoprogrammingBridgeRequestForTestV0(),
		OccurredAt:    "2026-05-22T10:00:00Z",
		CorrelationID: "corr-autoprogramming-bridge-001",
		RequestedBy:   "orquesta-test",
		MaxBursts:     4,
		MaxCommands:   8,
	}, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if !bridged.Accepted || len(bridged.Tasks) != 1 || len(bridged.WaitAgentRefs) != 1 {
		t.Fatalf("bridged=%+v", bridged)
	}
	storedTasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), bridged.Run.RunID, bridged.Run.Tasks)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(storedTasks) != 1 ||
		storedTasks[0].TaskID != bridged.Tasks[0].TaskID ||
		storedTasks[0].RunID != bridged.Run.RunID ||
		storedTasks[0].RequiredTests[0] != "go test -count=1 ./modulos/orquesta-app-codex-stack -run TestPrepareAutoprogrammingRunV0" {
		t.Fatalf("stored_tasks=%+v bridged=%+v", storedTasks, bridged.Tasks)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(storedTasks[0].ContextRefs, autoprogrammingBridgeOperationalTaskSourceRefV0) {
		t.Fatalf("context_refs=%v", storedTasks[0].ContextRefs)
	}
	run, err := runStore.LoadRunV0(context.Background(), bridged.Run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		len(run.Tasks) != 1 ||
		run.Tasks[0] != bridged.Tasks[0].TaskID {
		t.Fatalf("run no continuable: %+v", run)
	}
	agentTask, err := (CodexLaunchSpecResolverV0{TaskStore: taskStore}).agentTaskV0(
		context.Background(),
		orquestaruntime.LaunchRuntimeAgentRequestV0{
			RunID:   bridged.Run.RunID,
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef: bridged.Tasks[0].TaskID,
		},
		"programacion",
	)
	if err != nil {
		t.Fatalf("agentTaskV0: %v", err)
	}
	if agentTask.TaskRef != bridged.Tasks[0].TaskID ||
		len(agentTask.WriteSet) != 1 ||
		agentTask.WriteSet[0] != "modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go" {
		t.Fatalf("agent_task=%+v", agentTask)
	}
	if bridged.Continue.RunRef != bridged.Run.RunID ||
		len(bridged.Continue.WaitAgentRefs) != 1 ||
		bridged.Continue.WaitAgentRefs[0] != bridged.WaitAgentRefs[0] {
		t.Fatalf("continue=%+v wait=%v", bridged.Continue, bridged.WaitAgentRefs)
	}

	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(
		context.Background(),
		bridged.Continue,
		ports,
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if continued.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		!autoprogrammingBridgeStringInSetForTestV0(continued.StartedAgents, bridged.WaitAgentRefs[0]) {
		t.Fatalf("continued=%+v wait_agent=%s", continued, bridged.WaitAgentRefs[0])
	}
	planRef := "operational-director-plan-director-decisions-" + bridged.Run.RunID
	planState, err := planStore.LoadOperationalDirectorPlanStateV0(context.Background(), bridged.Run.RunID, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := codexStackPlanStateStepForTestV0(t, planState, "step-wait-subagents")
	if planState.ActiveStepID != "step-wait-subagents" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!autoprogrammingBridgeStringInSetForTestV0(planState.RequiredTestRefs, storedTasks[0].RequiredTests[0]) {
		t.Fatalf("plan_state=%+v wait_step=%+v", planState, waitStep)
	}
}

func TestPrepareAutoprogrammingRunV0NoPersisteRequestInvalida(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	request := autoprogrammingBridgeRequestForTestV0()
	request.RequiredTests = nil

	bridged, err := PrepareAutoprogrammingRunV0(context.Background(), AutoprogrammingBridgeRequestV0{
		Request: request,
	}, orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:          runStore,
		DirectorTaskStore: taskStore,
	})
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0: %v", err)
	}
	if bridged.Accepted || len(bridged.Issues) == 0 {
		t.Fatalf("bridged=%+v", bridged)
	}
	if _, err := runStore.LoadRunV0(context.Background(), request.RequestRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run no debe persistirse, err=%v", err)
	}
}

func TestPrepareAutoprogrammingRunV0EsIdempotenteYNoSobrescribeRunViva(t *testing.T) {
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		OutboxLedger:      ledger,
		DirectorTaskStore: taskStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			autoprogrammingBridgeCapacityDispatcherForTestV0(runStore, eventSink, ledger),
			autoprogrammingBridgeAgentDispatcherForTestV0(runStore, eventSink, ledger),
		},
	}
	request := AutoprogrammingBridgeRequestV0{
		Request:       autoprogrammingBridgeRequestForTestV0(),
		OccurredAt:    "2026-05-22T10:10:00Z",
		CorrelationID: "corr-autoprogramming-idempotent-001",
		RequestedBy:   "orquesta-test",
		MaxBursts:     4,
		MaxCommands:   8,
	}

	first, err := PrepareAutoprogrammingRunV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0 first: %v", err)
	}
	continued, err := orquestaappdirectorservice.ContinueAppDirectorV0(
		context.Background(),
		first.Continue,
		ports,
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(continued.StartedAgents, first.WaitAgentRefs[0]) {
		t.Fatalf("continued=%+v wait_agent=%s", continued, first.WaitAgentRefs[0])
	}

	second, err := PrepareAutoprogrammingRunV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0 second: %v", err)
	}
	run, err := runStore.LoadRunV0(context.Background(), first.Run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !autoprogrammingBridgeStringInSetForTestV0(run.StartedAgents, first.WaitAgentRefs[0]) {
		t.Fatalf("prepare-run repetido sobrescribio started_agents: run=%+v", run)
	}
	if second.Run.RunID != first.Run.RunID ||
		second.Continue.RunRef != first.Run.RunID ||
		second.WaitAgentRefs[0] != first.WaitAgentRefs[0] {
		t.Fatalf("second=%+v first=%+v", second, first)
	}
	changedRequest := request
	changedRequest.Request.Tasks = append([]orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0(nil), request.Request.Tasks...)
	changedRequest.Request.Tasks[0].AcceptanceCriteria = append(
		append([]string(nil), request.Request.Tasks[0].AcceptanceCriteria...),
		"criterio nuevo reparable sin bloquear reentrada",
	)
	third, err := PrepareAutoprogrammingRunV0(context.Background(), changedRequest, ports)
	if err != nil {
		t.Fatalf("PrepareAutoprogrammingRunV0 tolerante: %v", err)
	}
	if third.Run.RunID != first.Run.RunID ||
		third.Tasks[0].TaskID != first.Tasks[0].TaskID ||
		autoprogrammingBridgeStringInSetForTestV0(third.Tasks[0].AcceptanceCriteria, "criterio nuevo reparable sin bloquear reentrada") {
		t.Fatalf("third no reutiliza tarea viva: third=%+v first=%+v", third, first)
	}
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
		RunQueue: RunQueueConfigV0{QueueRef: "queue-main", DefaultPriorityScore: 10},
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

func autoprogrammingBridgeRequestForTestV0() orquestaautoprogramming.AutoprogrammingRequestV0 {
	return orquestaautoprogramming.AutoprogrammingRequestV0{
		RequestRef:       "run-autoprogramming-bridge-001",
		ProjectRef:       "project-ref-autoprogramming-bridge-001",
		WorktreeRef:      "worktree-ref-autoprogramming-bridge-001",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-autoprogramming-bridge-001",
		Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "source-task-ref-autoprogramming-bridge-001",
			Area:    "app-codex-stack",
		}},
		WriteSet: []string{
			"modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go",
		},
		RequiredTests: []string{
			"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestPrepareAutoprogrammingRunV0",
		},
	}
}

func autoprogrammingBridgeCapacityDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			OccurredAt: "2026-05-22T10:01:00Z",
		},
		Acker: ledger,
	}
}

func autoprogrammingBridgeAgentDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Launcher:   orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt: "2026-05-22T10:02:00Z",
		},
		Acker: ledger,
	}
}

func autoprogrammingBridgeStringInSetForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func autoprogrammingBridgeStringContainsForTestV0(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func codexStackPlanStateStepForTestV0(
	t *testing.T,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	stepID string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	t.Helper()
	for _, step := range state.Steps {
		if step.StepID == stepID {
			return step
		}
	}
	t.Fatalf("step %s no existe en %+v", stepID, state)
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
}
