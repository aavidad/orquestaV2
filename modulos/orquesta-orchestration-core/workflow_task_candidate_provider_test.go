package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestWorkflowTaskCandidateProviderV0BuildsWorkCandidatesFromTaskStore(t *testing.T) {
	runRef := "run-nucleo-workflow-task-candidate-001"
	task := workflowTaskForCandidateProviderTestV0(runRef, "task-workitem-api", []string{"app/main.go"})
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{task.TaskID}
	provider := WorkflowTaskCandidateProviderV0{
		TaskStore:       NewInMemoryWorkflowTaskStoreV0(task),
		RequestedBy:     "orquesta-nucleo-test",
		DefaultCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
	}

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:           run,
		OccurredAt:    "2026-05-09T13:00:00Z",
		CorrelationID: "corr-workflow-task-candidate-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("work candidates=%+v", candidates.WorkCandidates)
	}
	candidate := candidates.WorkCandidates[0]
	if candidate.AgentCandidate.Payload.AgentRequestID != WorkflowTaskAgentRequestRefV0(task.TaskID) {
		t.Fatalf("agent ref=%s", candidate.AgentCandidate.Payload.AgentRequestID)
	}
	if candidate.CapacityCandidate.Payload.MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityHighV0 {
		t.Fatalf("capacity=%s", candidate.CapacityCandidate.Payload.MinimumRecommendedCapacity)
	}
	if got := candidate.Claims[0].WriteSet[0].Ref; got != "app/main.go" {
		t.Fatalf("write set=%s", got)
	}
}

func TestWorkflowTaskCandidateProviderV0FeedsSchedulerWithoutManualCandidates(t *testing.T) {
	runRef := "run-nucleo-workflow-task-service-001"
	task := workflowTaskForCandidateProviderTestV0(runRef, "task-workitem-service", []string{"app/service.go"})
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{task.TaskID}
	ledger := &memoryOutboxLedgerV0{}
	sink := &recordingEventSinkV0{}
	service := ServiceV0{
		RunStore:  newMemoryRunStoreV0(run),
		EventSink: sink,
		CandidateProvider: WorkflowTaskCandidateProviderV0{
			TaskStore:   NewInMemoryWorkflowTaskStoreV0(task),
			RequestedBy: "orquesta-nucleo-test",
		},
		OutboxLedger: ledger,
	}

	result, err := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
		RunRef:        runRef,
		OccurredAt:    "2026-05-09T13:05:00Z",
		MaxSteps:      1,
		CorrelationID: "corr-workflow-task-service-001",
	})
	if err != nil {
		t.Fatalf("RunSupervisedBurstV0: %v", err)
	}
	if result.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
		t.Fatalf("final action=%s", result.Burst.FinalAction)
	}
	wantCapacity := WorkflowTaskCapacityRequestRefV0(task.TaskID)
	if len(result.Run.CapacityRequests) != 1 || result.Run.CapacityRequests[0] != wantCapacity {
		t.Fatalf("capacity requests=%v want %s", result.Run.CapacityRequests, wantCapacity)
	}
	pending, issues := ledger.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef})
	if len(issues) > 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
	if len(sink.events) != 1 || sink.events[0].EventType != orquestacoreworkflow.OrchestrationEventCapacityRequestedV0 {
		t.Fatalf("events=%+v", sink.events)
	}
}

func TestWorkflowTaskCandidateProviderV0RequiresTaskStoreForProgramacion(t *testing.T) {
	runRef := "run-nucleo-workflow-task-store-missing-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{"task-workitem-store-missing"}

	_, err := (WorkflowTaskCandidateProviderV0{}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T13:10:00Z",
	})
	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "workflow_task_store")
}

func TestWorkflowTaskCandidateProviderV0SkipsClosedTasks(t *testing.T) {
	runRef := "run-nucleo-workflow-task-closed-001"
	task := workflowTaskForCandidateProviderTestV0(runRef, "task-workitem-closed", []string{"app/closed.go"})
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{task.TaskID}
	run.ClosedTasks = []string{task.TaskID}

	candidates, err := (WorkflowTaskCandidateProviderV0{
		TaskStore: NewInMemoryWorkflowTaskStoreV0(task),
	}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T13:15:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.WorkCandidates) != 0 {
		t.Fatalf("closed task generated candidates=%+v", candidates.WorkCandidates)
	}
}

func TestWorkflowTaskCandidateProviderV0WaitsForDeliveredDependencies(t *testing.T) {
	runRef := "run-nucleo-workflow-task-dependency-001"
	bootstrap := workflowTaskForCandidateProviderTestV0(runRef, "task-bootstrap", []string{"go.mod"})
	api := workflowTaskForCandidateProviderTestV0(runRef, "task-api", []string{"internal/api"})
	api.DependsOn = []string{bootstrap.TaskID}
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{bootstrap.TaskID, api.TaskID}

	provider := WorkflowTaskCandidateProviderV0{
		TaskStore: NewInMemoryWorkflowTaskStoreV0(bootstrap, api),
	}
	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-11T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	got := workflowCandidateTaskRefsForTestV0(candidates.WorkCandidates)
	if !sameStringsForTestV0(got, []string{bootstrap.TaskID}) {
		t.Fatalf("frontier=%v want bootstrap solo", got)
	}

	run.DeliveredTasks = []string{bootstrap.TaskID}
	run.Agents = []string{WorkflowTaskAgentRequestRefV0(bootstrap.TaskID)}
	run.StartedAgents = []string{WorkflowTaskAgentRequestRefV0(bootstrap.TaskID)}
	candidates, err = provider.BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-11T10:05:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0 tras entrega: %v", err)
	}
	got = workflowCandidateTaskRefsForTestV0(candidates.WorkCandidates)
	if !sameStringsForTestV0(got, []string{api.TaskID}) {
		t.Fatalf("frontier=%v want api tras entrega", got)
	}
}

func TestWorkflowTaskCandidateProviderV0BuildsCompactConflictAwareFrontier(t *testing.T) {
	runRef := "run-nucleo-workflow-task-frontier-001"
	domain := workflowTaskForCandidateProviderTestV0(runRef, "task-domain", []string{"internal/agenda/ports"})
	persistence := workflowTaskForCandidateProviderTestV0(runRef, "task-persistence", []string{"internal/agenda/ports"})
	docs := workflowTaskForCandidateProviderTestV0(runRef, "task-docs", []string{"docs/agenda"})
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{domain.TaskID, persistence.TaskID, docs.TaskID}

	candidates, err := (WorkflowTaskCandidateProviderV0{
		TaskStore: NewInMemoryWorkflowTaskStoreV0(domain, persistence, docs),
	}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T13:20:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	got := workflowCandidateTaskRefsForTestV0(candidates.WorkCandidates)
	want := []string{domain.TaskID, docs.TaskID}
	if !sameStringsForTestV0(got, want) {
		t.Fatalf("frontier=%v want %v", got, want)
	}
	if len(candidates.WorkClaims) != 2 {
		t.Fatalf("work_claims=%d want 2", len(candidates.WorkClaims))
	}
}

func TestWorkflowTaskCandidateProviderV0SkipsWorkBlockedByActiveAgent(t *testing.T) {
	runRef := "run-nucleo-workflow-task-active-001"
	domain := workflowTaskForCandidateProviderTestV0(runRef, "task-active-domain", []string{"internal/agenda/ports"})
	persistence := workflowTaskForCandidateProviderTestV0(runRef, "task-active-persistence", []string{"internal/agenda/ports"})
	docs := workflowTaskForCandidateProviderTestV0(runRef, "task-active-docs", []string{"docs/agenda"})
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{domain.TaskID, persistence.TaskID, docs.TaskID}
	run.Agents = []string{WorkflowTaskAgentRequestRefV0(domain.TaskID)}
	run.StartedAgents = []string{WorkflowTaskAgentRequestRefV0(domain.TaskID)}

	candidates, err := (WorkflowTaskCandidateProviderV0{
		TaskStore: NewInMemoryWorkflowTaskStoreV0(domain, persistence, docs),
	}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T13:25:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	got := workflowCandidateTaskRefsForTestV0(candidates.WorkCandidates)
	want := []string{docs.TaskID}
	if !sameStringsForTestV0(got, want) {
		t.Fatalf("frontier=%v want %v", got, want)
	}
	if len(candidates.WorkClaims) != 2 {
		t.Fatalf("work_claims=%d want active+frontier", len(candidates.WorkClaims))
	}
}

func TestWorkflowTaskCandidateProviderV0LimitsFrontier(t *testing.T) {
	runRef := "run-nucleo-workflow-task-frontier-limit-001"
	tasks := []orquestacoreworkflow.WorkflowTaskV0{
		workflowTaskForCandidateProviderTestV0(runRef, "task-limit-a", []string{"app/a"}),
		workflowTaskForCandidateProviderTestV0(runRef, "task-limit-b", []string{"app/b"}),
		workflowTaskForCandidateProviderTestV0(runRef, "task-limit-c", []string{"app/c"}),
	}
	run := mustActiveProgrammingRunV0(t, runRef)
	for _, task := range tasks {
		run.Tasks = append(run.Tasks, task.TaskID)
	}

	candidates, err := (WorkflowTaskCandidateProviderV0{
		TaskStore:   NewInMemoryWorkflowTaskStoreV0(tasks...),
		MaxFrontier: 2,
	}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-09T13:30:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.WorkCandidates) != 2 || len(candidates.WorkClaims) != 2 {
		t.Fatalf("frontier=%d claims=%d", len(candidates.WorkCandidates), len(candidates.WorkClaims))
	}
}

func TestInMemoryWorkflowTaskStoreV0RejectsSameTaskWithDifferentContract(t *testing.T) {
	store := NewInMemoryWorkflowTaskStoreV0()
	task := workflowTaskForCandidateProviderTestV0(
		"run-store-conflict",
		"task-store-conflict",
		[]string{"internal/a"},
	)
	if err := store.SaveWorkflowTaskV0(context.Background(), task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 inicial: %v", err)
	}
	task.RequiredTests = []string{"go test ./..."}

	err := store.SaveWorkflowTaskV0(context.Background(), task)
	if err == nil {
		t.Fatalf("expected workflow task conflict")
	}
}

func workflowTaskForCandidateProviderTestV0(
	runRef string,
	taskRef string,
	writeSet []string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Microtarea de codigo",
		Summary:       "Trabajo compacto de programacion.",
		WriteSet:      writeSet,
		AcceptanceCriteria: []string{
			"Solo se modifica el write-set declarado.",
			"Se registran evidencias compactas.",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
			{ContractRef: "contract:function:workitem:v0", FunctionName: "NewWorkflowTaskV0"},
		},
	}
}

func workflowCandidateTaskRefsForTestV0(
	candidates []orquestadirectorscheduler.SchedulableWorkCandidateV0,
) []string {
	refs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.AgentCandidate != nil {
			refs = append(refs, candidate.AgentCandidate.Payload.TaskRef)
		}
	}
	return refs
}

func sameStringsForTestV0(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
