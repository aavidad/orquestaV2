package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestCodexSupervisorStackLifecycleV0SupervisaRunExistenteSinCanalParaleloV0(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != len(director.StartedAgents) {
		t.Fatalf("launches iniciales=%d director=%+v", runtime.launchCountV0(), director)
	}

	lifecycle := CodexSupervisorStackLifecycleV0{
		Stack:  stack,
		RunRef: director.RunRef,
		DrainRequest: DrainRunRequestV0{
			CorrelationID:        "corr-codex-supervisor-stack-lifecycle-001",
			OccurredAt:           "2026-05-18T12:00:00Z",
			MaxBursts:            2,
			MaxStepsPerBurst:     4,
			MaxDispatchesPerWait: 2,
			MaxCommands:          8,
			MaxOutboxPerCycle:    4,
			MaxDecisionCycles:    1,
			MaxExternalWaits:     1,
		},
	}

	result, err := SuperviseCodexV0(ctx, CodexSupervisorDepsV0{AgentLifecycle: lifecycle}, CodexSupervisorCommandV0{
		MaxTicks: 2,
	})
	if err != nil {
		t.Fatalf("SuperviseCodexV0: %v result=%+v", err, result)
	}
	if result.StopReason != CodexSupervisorStopMaxTicksV0 || result.Ticks != 2 {
		t.Fatalf("result=%+v", result)
	}
	if len(result.History) != 2 ||
		result.History[0].Action != "launch" ||
		result.History[1].Action != "continue" {
		t.Fatalf("history=%+v", result.History)
	}
	if result.Last.SessionRef != director.RunRef ||
		result.Last.Status != CodexSupervisorRuntimeRunningLiveV0 ||
		result.Last.AgentRef == "" ||
		result.Last.ProcessRef == "" ||
		!codexStackRefsContainPartV0(result.Last.EvidenceRefs, "evidence-ref-codex-supervisor-stack-drain") {
		t.Fatalf("last=%+v director=%+v", result.Last, director)
	}
	if runtime.launchCountV0() != len(director.StartedAgents) {
		t.Fatalf("el supervisor relanzo agentes: launches=%d director=%+v", runtime.launchCountV0(), director)
	}
}

func TestCodexSupervisorStackLifecycleV0UsaSupervisorGlobalExistenteV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	low := postDirectorAPIWithNameV0(t, stack, "app-codex-supervisor-low", "Agenda Supervisor Low")
	high := postDirectorAPIWithNameV0(t, stack, "app-codex-supervisor-high", "Agenda Supervisor High")
	setStackRunPriorityForTestV0(t, stack, low.RunRef, low.AppSpec.Slug, 10)
	setStackRunPriorityForTestV0(t, stack, high.RunRef, high.AppSpec.Slug, 90)

	lifecycle := CodexSupervisorStackLifecycleV0{
		Stack:             stack,
		SupervisorCommand: supervisorCommandForStackTestV0(2),
	}

	snapshot, err := lifecycle.LaunchV0(ctx)
	if err != nil {
		t.Fatalf("LaunchV0: %v snapshot=%+v", err, snapshot)
	}
	if snapshot.Status == CodexSupervisorRuntimeFailedV0 ||
		!codexStackRefsContainPartV0(snapshot.EvidenceRefs, "evidence-ref-codex-supervisor-stack-global") {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	if snapshot.SessionRef != low.RunRef {
		t.Fatalf("session=%s want last execution %s snapshot=%+v", snapshot.SessionRef, low.RunRef, snapshot)
	}
}

func TestCodexStackRunSupervisorAPIV0EmpujaRunExistenteSinRelanzarAgentes(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:     "request-ref-run-supervisor-api-001",
		CorrelationID: "corr-run-supervisor-api-001",
		RunRef:        director.RunRef,
		MaxTicks:      2,
		MaxBursts:     2,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != director.RunRef ||
		result.Last.SessionRef != director.RunRef ||
		result.Last.Status != string(CodexSupervisorRuntimeRunningLiveV0) ||
		result.Last.AgentRef == "" ||
		result.Last.ProcessRef == "" ||
		!codexStackRefsContainPartV0(result.Last.EvidenceRefs, "evidence-ref-codex-supervisor-stack-drain") {
		t.Fatalf("result=%+v director=%+v", result, director)
	}
	if runtime.launchCountV0() != len(director.StartedAgents) {
		t.Fatalf("el endpoint relanzo agentes: launches=%d director=%+v", runtime.launchCountV0(), director)
	}
}

func TestCodexSupervisorStackLifecycleV0SincronizaColaTerminalEnDrainDirectoV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	director := postDirectorAPIV0(t, stack)
	runRef := director.RunRef
	taskRef := "task-ref-supervisor-direct-terminal-queue-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	run.AppSpecRef = "app-ref-supervisor-direct-terminal-queue-001"
	run = codexStackRunWithActivePhaseForTestV0(run, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	for index := range run.Phases {
		if run.Phases[index].RecommendedCapacity == "" {
			run.Phases[index].RecommendedCapacity = orquestacoreworkflow.OrchestrationCapacityHighV0
		}
	}
	run.Brainstorms = nil
	run.Votes = nil
	run.Tasks = []string{taskRef}
	run.FunctionContracts = nil
	run.Decisions = nil
	run.CapacityRequests = nil
	run.CapacityDecisions = nil
	run.Agents = []string{agentRef}
	run.AgentPhaseRefs = nil
	run.StartedAgents = []string{agentRef}
	run.FailedAgents = nil
	run.LostAgents = nil
	run.StoppedAgents = nil
	run.AgentStopRequests = nil
	run.ConfirmedStoppedAgents = nil
	run.AgentAssessments = nil
	run.AgentLeaseExpirations = nil
	run.ConcurrencyGates = nil
	run.QualityGates = nil
	run.PhaseArtifacts = nil
	run.DeliveredAgents = []string{agentRef}
	run.Deliveries = []string{"delivery-ref-" + taskRef}
	run.DeliveredTasks = []string{taskRef}
	run.Reviews = nil
	run.ReviewResults = nil
	run.ReworkRequests = nil
	run.ReplanDecisions = nil
	run.AcceptedReviews = nil
	run.ClosedTasks = []string{taskRef}
	run.Validations = nil
	run.Closures = nil
	run.DirectorQuestions = nil
	run.DirectorAnswers = nil
	run.DirectorAnsweredQuestions = nil
	run.Blockers = nil
	run.CommandEffects = nil
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Entrega terminal para sincronizar cola",
		WriteSet:           []string{"modulos/orquesta-app-codex-stack"},
		AcceptanceCriteria: []string{"cola sincronizada como terminal tras supervise directo"},
	}); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        run.AppSpecRef,
		Status:        orquestarunqueue.RunStatusRunningV0,
		PriorityScore: 90,
		EvidenceRefs:  []string{"evidence-ref-test-direct-drain-running"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 running: %v", err)
	}
	lifecycle := CodexSupervisorStackLifecycleV0{
		Stack:  stack,
		RunRef: runRef,
		DrainRequest: DrainRunRequestV0{
			CorrelationID:        "corr-supervisor-direct-terminal-queue-001",
			OccurredAt:           "2026-06-11T12:00:00Z",
			MaxBursts:            1,
			MaxStepsPerBurst:     4,
			MaxDispatchesPerWait: 1,
			MaxCommands:          4,
			MaxOutboxPerCycle:    4,
			MaxDecisionCycles:    1,
			MaxExternalWaits:     1,
		},
	}

	snapshot, err := lifecycle.LaunchV0(ctx)
	if err != nil {
		t.Fatalf("LaunchV0: %v snapshot=%+v", err, snapshot)
	}
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             DefaultRunQueueRefV0,
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0: %v", err)
	}
	if snapshot.Status != CodexSupervisorRuntimeDoneV0 ||
		len(candidates) != 1 ||
		candidates[0].Status != orquestarunqueue.RunStatusDeliveredV0 ||
		orquestarunqueue.IsExecutableRunStatusV0(candidates[0].Status) ||
		!codexStackRefsContainPartV0(candidates[0].EvidenceRefs, "direct-drain-terminal-queue-sync") {
		t.Fatalf("snapshot=%+v candidates=%+v", snapshot, candidates)
	}
}

type codexSupervisorRequiredTestRunnerForTestV0 struct {
	writer orquestacionnucleoapp.RequiredTestEvidenceWriterPortV0
	calls  int
}

func (runner *codexSupervisorRequiredTestRunnerForTestV0) RunRequiredTestsV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, error) {
	runner.calls++
	result := orquestacionnucleoapp.RequiredTestExecutionResultV0{}
	for _, command := range compactStringsV0(request.TestCommands) {
		evidenceRef := codexStackDeterministicRefV0(
			"test-evidence-ref-supervisor-direct-required-tests-",
			request.RunRef,
			request.TaskRef,
			command,
			request.DeliveryRef,
			request.ReviewResultRef,
			request.AcceptedReviewRef,
		)
		if runner.writer != nil {
			if err := runner.writer.SaveRequiredTestEvidenceV0(ctx, orquestacionnucleoapp.RequiredTestEvidenceV0{
				SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
				EvidenceRef:       evidenceRef,
				RunRef:            request.RunRef,
				TaskRef:           request.TaskRef,
				TestCommand:       command,
				Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				DeliveryRef:       request.DeliveryRef,
				ReviewRequestID:   request.ReviewRequestID,
				ReviewResultRef:   request.ReviewResultRef,
				AcceptedReviewRef: request.AcceptedReviewRef,
				OccurredAt:        request.OccurredAt,
				EvidenceRefs:      []string{"artifact-ref-supervisor-direct-required-test"},
			}); err != nil {
				return result, err
			}
		}
		result.EvidenceRefs = append(result.EvidenceRefs, evidenceRef)
		result.PassedEvidenceRefs = append(result.PassedEvidenceRefs, evidenceRef)
	}
	result.EvidenceRefs = compactStringsV0(result.EvidenceRefs)
	result.PassedEvidenceRefs = compactStringsV0(result.PassedEvidenceRefs)
	return result, nil
}

func TestCodexSupervisorSnapshotFromDrainV0WaitUnhandledOutboxNoEsRunningV0(t *testing.T) {
	ctx := context.Background()
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:  "run-ref-supervisor-waiting-outbox-001",
		Status: orquestacoreworkflow.OrchestrationRunStatusActiveV0,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status:             orquestacionnucleoapp.ProgressiveLoopStatusWaitUnhandledOutboxV0,
		Run:                run,
		PendingOutboxRefs:  []string{"outbox-ref-supervisor-waiting-001"},
		PendingOutboxCount: 1,
	}
	snapshot := (CodexSupervisorStackLifecycleV0{}).codexSupervisorSnapshotFromDrainV0(ctx, run.RunID, "", orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: loop.Status,
		Final:  loop,
	})
	if snapshot.Status != CodexSupervisorRuntimeWaitingOutboxV0 ||
		snapshot.ProcessRef != "" ||
		!codexStackRefsContainPartV0(snapshot.EvidenceRefs, "wait_unhandled_outbox") {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestCodexSupervisorSnapshotFromDrainV0ProcesoParadoEsStalledV0(t *testing.T) {
	ctx := context.Background()
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	agentRef := codexSupervisorLastRefV0(director.StartedAgents)
	record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(ctx, director.RunRef, agentRef)
	if err != nil {
		t.Fatalf("ResolveAgentProcessV0: %v", err)
	}
	runtime.markStoppedForTestV0(record.ProcessRef)
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
		Run:    run,
	}
	lifecycle := CodexSupervisorStackLifecycleV0{Stack: stack}
	snapshot := lifecycle.codexSupervisorSnapshotFromDrainV0(ctx, director.RunRef, "", orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: loop.Status,
		Final:  loop,
	})
	if snapshot.Status != CodexSupervisorRuntimeStalledV0 ||
		snapshot.AgentRef != agentRef ||
		snapshot.ProcessRef != record.ProcessRef ||
		!codexStackRefsContainPartV0(snapshot.EvidenceRefs, "evidence-ref-codex-supervisor-process-not-live") {
		t.Fatalf("snapshot=%+v agent=%s process=%s", snapshot, agentRef, record.ProcessRef)
	}
}

func TestCodexStackRunSupervisorAPIV0BloqueoRequiredTestsEvidenceMissingDevuelveOKV0(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	stack.Stores.OperationalPlanStateWriter = planStore
	stack.Stores.OperationalPlanStateStore = planStore
	stack.Stores.RequiredTestEvidenceStore = evidenceStore
	stack.Ports.OperationalPlanStateWriter = planStore
	stack.Ports.OperationalPlanStateStore = planStore
	stack.Ports.RequiredTestEvidenceStore = evidenceStore
	stack.Ports.RequiredTestRunner = nil

	refs := (codexStackRequiredTestRefsV0{
		RunRef:  "run-ref-supervisor-required-tests-evidence-missing-001",
		PlanRef: "plan-ref-supervisor-required-tests-evidence-missing-001",
	}).withDefaultsV0()
	if err := stack.Stores.RunStore.SaveRunV0(ctx, refs.runV0()); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, refs.workflowTaskV0()); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := planStore.SaveOperationalDirectorPlanStateV0(ctx, refs.planStateV0()); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, refs.RunRef, refs.eventsV0(t)); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:                  "request-ref-run-supervisor-required-tests-missing-001",
		CorrelationID:              "corr-run-supervisor-required-tests-missing-001",
		RunRef:                     refs.RunRef,
		OperationalDirectorPlanRef: refs.PlanRef,
		MaxTicks:                   1,
		MaxBursts:                  2,
		MaxStepsPerBurst:           4,
		MaxDispatchesPerWait:       2,
		MaxCommands:                8,
		MaxOutboxPerCycle:          4,
		MaxDecisionCycles:          1,
		MaxExternalWaits:           1,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	orquestamcp.NewMCPRunSupervisorHTTPHandlerV0(NewCodexStackRunSupervisorExecutorV0(&stack)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	state, err := planStore.LoadOperationalDirectorPlanStateV0(ctx, refs.RunRef, refs.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != refs.RunRef ||
		result.Last.Status != string(CodexSupervisorRuntimeStoppedV0) ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		!codexStackRefsContainPartV0(result.Last.EvidenceRefs, "operational-director-plan-state:blocked") ||
		!codexStackRefsContainPartV0(result.Last.EvidenceRefs, "required-tests-evidence-missing") {
		t.Fatalf("result=%+v state=%+v", result, state)
	}
}

func TestCodexStackRunSupervisorAPIV0RecuperaPlanRefYReintentaRequiredTestsBloqueadosV0(t *testing.T) {
	ctx := context.Background()
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	runner := &codexSupervisorRequiredTestRunnerForTestV0{writer: evidenceStore}
	stack.Stores.OperationalPlanStateWriter = planStore
	stack.Stores.OperationalPlanStateStore = planStore
	stack.Stores.RequiredTestEvidenceStore = evidenceStore
	stack.Ports.OperationalPlanStateWriter = planStore
	stack.Ports.OperationalPlanStateStore = planStore
	stack.Ports.RequiredTestEvidenceStore = evidenceStore
	stack.Ports.RequiredTestRunner = runner

	refs := (codexStackRequiredTestRefsV0{
		RunRef:  "run-ref-supervisor-required-tests-retry-without-plan-ref-001",
		PlanRef: "operational-director-plan-director-decisions-run-ref-supervisor-required-tests-retry-without-plan-ref-001",
	}).withDefaultsV0()
	if err := stack.Stores.RunStore.SaveRunV0(ctx, refs.runV0()); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	taskWriter, ok := stack.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWriterPortV0)
	if !ok {
		t.Fatalf("TaskStore no escribe WorkflowTaskV0: %T", stack.Stores.TaskStore)
	}
	if err := taskWriter.SaveWorkflowTaskV0(ctx, refs.workflowTaskV0()); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	state := refs.planStateV0()
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.BlockerRefs = []string{"required-tests-evidence-missing"}
	for index := range state.Steps {
		if state.Steps[index].StepID != "step-run-required-tests" {
			continue
		}
		state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
		state.Steps[index].Reason = "required-tests-evidence-missing"
		state.Steps[index].BlockerRefs = []string{"required-tests-evidence-missing"}
	}
	if err := planStore.SaveOperationalDirectorPlanStateV0(ctx, state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	if err := stack.Stores.EventSink.AppendRunEventsV0(ctx, refs.RunRef, refs.eventsV0(t)); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-run-supervisor-required-tests-retry-without-plan-ref-001",
		CorrelationID:        "corr-run-supervisor-required-tests-retry-without-plan-ref-001",
		RunRef:               refs.RunRef,
		MaxTicks:             1,
		MaxBursts:            2,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 2,
		MaxCommands:          8,
		MaxOutboxPerCycle:    4,
		MaxDecisionCycles:    1,
		MaxExternalWaits:     1,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	orquestamcp.NewMCPRunSupervisorHTTPHandlerV0(NewCodexStackRunSupervisorExecutorV0(&stack)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	state, err := planStore.LoadOperationalDirectorPlanStateV0(ctx, refs.RunRef, refs.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := codexStackPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := codexStackPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != refs.RunRef ||
		runner.calls != 1 ||
		state.ActiveStepID != "step-replan-or-close" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		len(testsStep.RequiredTestEvidenceRefs) != 1 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 ||
		codexStackRefsContainPartV0(state.BlockerRefs, "required-tests-evidence-missing") {
		t.Fatalf("result=%+v state=%+v testsStep=%+v replanStep=%+v calls=%d", result, state, testsStep, replanStep, runner.calls)
	}
}

func TestCodexStackRunSupervisorAPIV0ConsumeDecisionFileYArrancaProgramacion(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	codexStackWriteDelayedDirectorDecisionsForTestV0(t, stack, director.RunRef)

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-run-supervisor-api-decisions-001",
		CorrelationID:        "corr-run-supervisor-api-decisions-001",
		RunRef:               director.RunRef,
		MaxTicks:             4,
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     4,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	taskRef := "task-ref-stack-agenda-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != director.RunRef ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackStringInSetForTestV0(run.Tasks, taskRef) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, agentRef) ||
		runtime.launchCountV0() < 5 {
		t.Fatalf("result=%+v run_phase=%s tasks=%v started=%v launches=%d",
			result,
			run.CurrentPhase,
			run.Tasks,
			run.StartedAgents,
			runtime.launchCountV0(),
		)
	}
}

func TestCodexStackRunSupervisorAPIV0ColaGlobalConsumeDecisionFileYArrancaProgramacion(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	codexStackWriteDelayedDirectorDecisionsForTestV0(t, stack, director.RunRef)

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-run-supervisor-api-queue-decisions-001",
		CorrelationID:        "corr-run-supervisor-api-queue-decisions-001",
		QueueRef:             DefaultRunQueueRefV0,
		MaxTicks:             4,
		MaxRunsPerTick:       1,
		MaxExecutions:        4,
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     4,
		AllowRepeatedRuns:    true,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	taskRef := "task-ref-stack-agenda-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if result.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		result.RunRef != director.RunRef ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackStringInSetForTestV0(run.Tasks, taskRef) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, agentRef) ||
		runtime.launchCountV0() < 5 {
		t.Fatalf("result=%+v run_phase=%s tasks=%v started=%v launches=%d",
			result,
			run.CurrentPhase,
			run.Tasks,
			run.StartedAgents,
			runtime.launchCountV0(),
		)
	}
}

func TestCodexStackRunSupervisorAPIV0ColaGlobalConsumeDecisionFileAparecidoTrasACK(t *testing.T) {
	runtime := newPendingAckCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	for _, descriptor := range codexStackDescriptorsForTestV0(t, stack) {
		if err := writeCodexStackCompletedAckForDescriptorV0(descriptor); err != nil {
			t.Fatalf("write ACK %s: %v", descriptor.AgentRef, err)
		}
	}

	first := postRunSupervisorQueueStackV0(t, stack, "late-decision-first")
	run := mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	if first.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		len(run.PhaseArtifacts) < len(director.StartedAgents) ||
		len(run.Tasks) != 0 {
		t.Fatalf("first=%+v artifacts=%v tasks=%v", first, run.PhaseArtifacts, run.Tasks)
	}

	codexStackWriteDelayedDirectorDecisionsForTestV0(t, stack, director.RunRef)
	second := postRunSupervisorQueueStackV0(t, stack, "late-decision-second")
	run = mustLoadCodexStackRunForTestV0(t, stack, director.RunRef)
	taskRef := "task-ref-stack-agenda-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if second.Estado != orquestamcp.MCPRunSupervisorEstadoOKV0 ||
		second.RunRef != director.RunRef ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		!codexStackStringInSetForTestV0(run.Tasks, taskRef) ||
		!codexStackStringInSetForTestV0(run.StartedAgents, agentRef) ||
		runtime.launchCountV0() < 5 {
		t.Fatalf("second=%+v run_phase=%s tasks=%v started=%v launches=%d",
			second,
			run.CurrentPhase,
			run.Tasks,
			run.StartedAgents,
			runtime.launchCountV0(),
		)
	}
}

func postRunSupervisorQueueStackV0(
	t *testing.T,
	stack StackV0,
	ref string,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-run-supervisor-api-" + ref,
		CorrelationID:        "corr-run-supervisor-api-" + ref,
		QueueRef:             DefaultRunQueueRefV0,
		MaxTicks:             4,
		MaxRunsPerTick:       1,
		MaxExecutions:        4,
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
		MaxDecisionCycles:    4,
		MaxExternalWaits:     4,
		AllowRepeatedRuns:    true,
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/supervise", body)
	req.Header.Set("Content-Type", "application/json")

	stack.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunSupervisorToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return result
}

func TestCodexSupervisorRuntimeStateFromLoopV0MapeaEstadosDelNucleoV0(t *testing.T) {
	tests := []struct {
		name      string
		status    orquestacionnucleoapp.ProgressiveLoopStatusV0
		runStatus orquestacoreworkflow.OrchestrationRunStatusV0
		want      CodexSupervisorRuntimeStateV0
	}{
		{
			name:   "espera_externa_sigue_vivo",
			status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			want:   CodexSupervisorRuntimeRunningV0,
		},
		{
			name:   "outbox_sin_worker_no_es_running",
			status: orquestacionnucleoapp.ProgressiveLoopStatusWaitUnhandledOutboxV0,
			want:   CodexSupervisorRuntimeWaitingOutboxV0,
		},
		{
			name:   "quiescente_termina",
			status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			want:   CodexSupervisorRuntimeDoneV0,
		},
		{
			name:   "necesita_director_sigue_vivo",
			status: orquestacionnucleoapp.ProgressiveLoopStatusNeedsDirectorV0,
			want:   CodexSupervisorRuntimeRunningV0,
		},
		{
			name:   "error_falla",
			status: orquestacionnucleoapp.ProgressiveLoopStatusStopErrorV0,
			want:   CodexSupervisorRuntimeFailedV0,
		},
		{
			name:      "run_cerrada_termina",
			status:    orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			runStatus: orquestacoreworkflow.OrchestrationRunStatusClosedV0,
			want:      CodexSupervisorRuntimeDoneV0,
		},
		{
			name:      "run_terminal_por_control_no_cierra_run_activa",
			status:    orquestacionnucleoapp.ProgressiveLoopStatusRunTerminalV0,
			runStatus: orquestacoreworkflow.OrchestrationRunStatusActiveV0,
			want:      CodexSupervisorRuntimeStoppedV0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codexSupervisorRuntimeStateFromLoopV0(tt.status, tt.runStatus); got != tt.want {
				t.Fatalf("got=%s want=%s", got, tt.want)
			}
		})
	}
}

func TestCodexSupervisorSnapshotFromDrainV0NoCierraAutomejoraActivaConEntregaAbiertaV0(t *testing.T) {
	ctx := context.Background()
	taskRef := "task-autoprogramming-open-delivered-001"
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              "run-ref-autoprogramming-open-delivered-001",
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Automejora abierta",
		WriteSet:           []string{"modulos/orquesta-app-codex-stack/codex_supervisor_stack_lifecycle_v0.go"},
		AcceptanceCriteria: []string{"no cerrar run activa con entrega pendiente de cierre"},
		RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestCodexSupervisorSnapshot"},
	})
	lifecycle := CodexSupervisorStackLifecycleV0{
		Stack: StackV0{
			Stores: StoresV0{TaskStore: taskStore},
		},
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:          "run-ref-autoprogramming-open-delivered-001",
			AppSpecRef:     "app-spec-ref-autoprogramming-open-delivered-001",
			Status:         orquestacoreworkflow.OrchestrationRunStatusActiveV0,
			Tasks:          []string{taskRef},
			DeliveredTasks: []string{taskRef},
		},
	}

	snapshot := lifecycle.codexSupervisorSnapshotFromDrainV0(ctx, loop.Run.RunID, "", orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: loop.Status,
		Final:  loop,
	})
	if snapshot.Status != CodexSupervisorRuntimeRunningV0 {
		t.Fatalf("snapshot=%+v want status=%s", snapshot, CodexSupervisorRuntimeRunningV0)
	}
	loop.Run.ClosedTasks = []string{taskRef}
	snapshot = lifecycle.codexSupervisorSnapshotFromDrainV0(ctx, loop.Run.RunID, "", orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: loop.Status,
		Final:  loop,
	})
	if snapshot.Status != CodexSupervisorRuntimeDoneV0 {
		t.Fatalf("snapshot=%+v want status=%s", snapshot, CodexSupervisorRuntimeDoneV0)
	}
}

func TestCodexSupervisorGlobalV0NoTrataDeliveredOpenComoIdleV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-global-delivered-open-001"
	taskRef := "task-ref-global-delivered-open-001"
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(orquestacoreworkflow.OrchestrationRunV0{
		RunID:          runRef,
		Status:         orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		Tasks:          []string{taskRef},
		DeliveredTasks: []string{taskRef},
	})
	queue := orquestarunmemory.NewRunMemoryStoreV0()
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "app-ref-global-delivered-open-001",
		Status:        orquestarunqueue.RunStatusDeliveredV0,
		PriorityScore: 10,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	lifecycle := CodexSupervisorStackLifecycleV0{
		Stack: StackV0{
			Stores: StoresV0{
				RunStore: runStore,
				RunQueue: queue,
			},
			RunQueue: RunQueueConfigV0{QueueRef: DefaultRunQueueRefV0},
		},
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			QueueRef:          DefaultRunQueueRefV0,
			MaxTicks:          1,
			StopOnNoExecution: true,
		},
	}

	snapshot, err := lifecycle.LaunchV0(ctx)
	if err != nil {
		t.Fatalf("LaunchV0: %v", err)
	}
	if snapshot.Status != CodexSupervisorRuntimeRunningV0 {
		t.Fatalf("snapshot=%+v want running para delivered-open", snapshot)
	}
	if !codexSupervisorEvidenceRefsContainPartV0(snapshot.EvidenceRefs, "delivered-open") {
		t.Fatalf("evidence_refs=%v", snapshot.EvidenceRefs)
	}
}

func TestCodexSupervisorRunOperationalBlockerRefsV0SoloBloqueosVivosV0(t *testing.T) {
	taskRef := "task-autoprogramming-blockers-live-001"
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID: "run-ref-autoprogramming-blockers-live-001",
		Tasks: []string{taskRef},
		QualityGates: []string{
			"quality-gate-ref-required-tests-failed-old#decision:blocked#subject:" + taskRef,
			"quality-gate-ref-required-tests-accepted#decision:accepted#subject:" + taskRef,
		},
	}

	if got := codexSupervisorRunOperationalBlockerRefsV0(run); len(got) != 0 {
		t.Fatalf("blockers=%+v", got)
	}

	run.QualityGates = append(run.QualityGates,
		"quality-gate-ref-required-tests-failed-new#decision:blocked#subject:"+taskRef,
	)
	got := codexSupervisorRunOperationalBlockerRefsV0(run)
	if len(got) != 1 || got[0] != "quality-gate-ref-required-tests-failed-new" {
		t.Fatalf("blockers=%+v", got)
	}
}
