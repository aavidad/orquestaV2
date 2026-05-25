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

func TestWorkflowTaskCandidateProviderV0UsaFaseActualDelRun(t *testing.T) {
	runRef := "run-nucleo-workflow-task-current-phase-001"
	task := workflowTaskForCandidateProviderTestV0(runRef, "task-workitem-docs", []string{"docs/uso.md"})
	task.PhaseID = orquestacoreworkflow.OrchestrationPhaseDocumentacionV0
	run := mustActiveProgrammingRunV0(t, runRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseDocumentacionV0
	run.Tasks = []string{task.TaskID}

	candidates, err := (WorkflowTaskCandidateProviderV0{
		TaskStore: NewInMemoryWorkflowTaskStoreV0(task),
	}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-17T11:00:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("work candidates=%+v", candidates.WorkCandidates)
	}
	payload := candidates.WorkCandidates[0].AgentCandidate.Payload
	if payload.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseDocumentacionV0) {
		t.Fatalf("phase_id=%s", payload.PhaseID)
	}
}

func TestWorkflowTaskCandidateProviderV0UsesWorkProfileKindForRoleAndCapacity(t *testing.T) {
	runRef := "run-nucleo-workflow-task-profile-001"
	task := workflowTaskFromProfileForCandidateProviderTestV0(
		t,
		runRef,
		orquestacoreworkflow.WorkProfileReviewV0,
	)
	run := mustActiveProgrammingRunV0(t, runRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	run.Tasks = []string{task.TaskID}

	candidates, err := (WorkflowTaskCandidateProviderV0{
		TaskStore: NewInMemoryWorkflowTaskStoreV0(task),
	}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-22T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("work candidates=%+v", candidates.WorkCandidates)
	}
	candidate := candidates.WorkCandidates[0]
	if candidate.AgentCandidate.Payload.Role != "revision" {
		t.Fatalf("role=%s", candidate.AgentCandidate.Payload.Role)
	}
	if candidate.CapacityCandidate.Payload.MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityHighV0 {
		t.Fatalf("capacity=%s", candidate.CapacityCandidate.Payload.MinimumRecommendedCapacity)
	}
	if candidate.CapacityCandidate.Payload.ReasonCode != "work_profile_review" {
		t.Fatalf("reason=%s", candidate.CapacityCandidate.Payload.ReasonCode)
	}
}

func TestWorkflowTaskCandidateProviderV0UsesInjectedProfileResolver(t *testing.T) {
	runRef := "run-nucleo-workflow-task-profile-resolver-001"
	task := workflowTaskForCandidateProviderTestV0(runRef, "task-workitem-profile-resolver", []string{"app/profile"})
	task.RequiredTests = []string{"go test -count=1 ./..."}
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{task.TaskID}
	profileEvidenceRef := "evidence-ref-profile-resolver-001"

	candidates, err := (WorkflowTaskCandidateProviderV0{
		TaskStore: NewInMemoryWorkflowTaskStoreV0(task),
		ProfileResolver: staticWorkflowTaskProfileResolverForTestV0{
			resolution: WorkflowTaskProfileResolutionV0{
				ProfileKind:                orquestacoreworkflow.WorkProfileRefactorV0,
				Role:                       "refactor",
				ReasonCode:                 "work_profile_refactor",
				CapacitySummary:            "Capacidad para refactor acotado.",
				AgentSummary:               "Refactor acotado listo.",
				MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityXHighV0,
				EvidenceRefs:               []string{profileEvidenceRef},
			},
		},
	}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-22T10:05:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("work candidates=%+v", candidates.WorkCandidates)
	}
	candidate := candidates.WorkCandidates[0]
	if candidate.AgentCandidate.Payload.Role != "refactor" ||
		candidate.CapacityCandidate.Payload.MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 ||
		!stringInSetV0(profileEvidenceRef, candidate.EvidenceRefs) {
		t.Fatalf("candidate=%+v", candidate)
	}
}

func TestWorkflowTaskCandidateProviderV0IgnoraTareaDeOtraFase(t *testing.T) {
	runRef := "run-nucleo-workflow-task-other-phase-001"
	task := workflowTaskForCandidateProviderTestV0(runRef, "task-workitem-other-phase", []string{"docs/uso.md"})
	task.PhaseID = orquestacoreworkflow.OrchestrationPhaseDocumentacionV0
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{task.TaskID}

	candidates, err := (WorkflowTaskCandidateProviderV0{
		TaskStore: NewInMemoryWorkflowTaskStoreV0(task),
	}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-17T11:05:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.WorkCandidates) != 0 {
		t.Fatalf("work candidates=%+v", candidates.WorkCandidates)
	}
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

func TestWorkflowTaskCandidateProviderV0NoBloqueaFollowupPorAgentePadreEntregado(t *testing.T) {
	runRef := "run-nucleo-workflow-task-delivered-parent-001"
	parent := workflowTaskForCandidateProviderTestV0(runRef, "task-parent", []string{"internal/api"})
	followup := workflowTaskForCandidateProviderTestV0(runRef, "task-followup", []string{"internal/api"})
	followup.DependsOn = []string{parent.TaskID}
	parentAgent := WorkflowTaskAgentRequestRefV0(parent.TaskID)
	run := mustActiveProgrammingRunV0(t, runRef)
	run.Tasks = []string{parent.TaskID, followup.TaskID}
	run.Agents = []string{parentAgent}
	run.StartedAgents = []string{parentAgent}
	run.DeliveredAgents = []string{parentAgent}
	run.DeliveredTasks = []string{parent.TaskID}

	candidates, err := (WorkflowTaskCandidateProviderV0{
		TaskStore: NewInMemoryWorkflowTaskStoreV0(parent, followup),
	}).BuildSchedulerCandidatesV0(context.Background(), SchedulerCandidateRequestV0{
		Run:        run,
		OccurredAt: "2026-05-25T18:20:00Z",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	got := workflowCandidateTaskRefsForTestV0(candidates.WorkCandidates)
	if !sameStringsForTestV0(got, []string{followup.TaskID}) {
		t.Fatalf("frontier=%v want followup tras entrega del padre", got)
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

func TestInMemoryWorkflowTaskStoreV0CompletaFunctionContractsFaltantes(t *testing.T) {
	store := NewInMemoryWorkflowTaskStoreV0()
	task := workflowTaskForCandidateProviderTestV0(
		"run-store-contract-repair",
		"task-store-contract-repair",
		[]string{"internal/a"},
	)
	task.FunctionContractRefs = nil
	if err := store.SaveWorkflowTaskV0(context.Background(), task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 inicial: %v", err)
	}
	task.FunctionContractRefs = []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
		ContractRef: "contract:function:workflow-task:v0",
	}}
	if err := store.SaveWorkflowTaskV0(context.Background(), task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 debe reconciliar contratos faltantes: %v", err)
	}
	loaded, err := store.LoadWorkflowTasksV0(context.Background(), task.RunID, []string{task.TaskID})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(loaded) != 1 || len(loaded[0].FunctionContractRefs) != 1 {
		t.Fatalf("task no reconciliada: %+v", loaded)
	}
}

func TestInMemoryWorkflowTaskStoreV0LoadsTasksByParent(t *testing.T) {
	runRef := "run-store-by-parent"
	parent := workflowTaskForCandidateProviderTestV0(runRef, "task-parent", []string{"internal/parent"})
	childA := workflowTaskForCandidateProviderTestV0(runRef, "task-child-a", []string{"internal/child-a"})
	childA.ParentTaskRef = parent.TaskID
	childB := workflowTaskForCandidateProviderTestV0(runRef, "task-child-b", []string{"internal/child-b"})
	childB.ParentTaskRef = " " + parent.TaskID + " "
	sibling := workflowTaskForCandidateProviderTestV0(runRef, "task-sibling", []string{"internal/sibling"})
	sibling.ParentTaskRef = "task-parent-other"
	otherRunChild := workflowTaskForCandidateProviderTestV0("run-store-by-parent-other", "task-child-other-run", []string{"internal/other"})
	otherRunChild.ParentTaskRef = parent.TaskID
	store := NewInMemoryWorkflowTaskStoreV0(parent, childB, sibling, otherRunChild, childA)

	got, err := store.LoadWorkflowTasksByParentV0(context.Background(), runRef, " "+parent.TaskID+" ")
	if err != nil {
		t.Fatalf("LoadWorkflowTasksByParentV0: %v", err)
	}
	if len(got) != 2 || got[0].TaskID != childA.TaskID || got[1].TaskID != childB.TaskID {
		t.Fatalf("children=%+v", got)
	}
	if _, err := store.LoadWorkflowTasksByParentV0(context.Background(), runRef, " "); err == nil {
		t.Fatal("expected parent_task_ref requerido")
	}
}

type staticWorkflowTaskProfileResolverForTestV0 struct {
	resolution WorkflowTaskProfileResolutionV0
}

func (resolver staticWorkflowTaskProfileResolverForTestV0) ResolveWorkflowTaskProfileV0(
	_ context.Context,
	_ WorkflowTaskProfileRequestV0,
) (WorkflowTaskProfileResolutionV0, error) {
	return resolver.resolution, nil
}

func workflowTaskFromProfileForCandidateProviderTestV0(
	t *testing.T,
	runRef string,
	kind orquestacoreworkflow.WorkProfileKindV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	t.Helper()
	task, err := orquestacoreworkflow.WorkflowTaskFromWorkProfileV0(orquestacoreworkflow.WorkProfileV0{
		SchemaVersion: orquestacoreworkflow.WorkProfileSchemaVersionV0,
		ProfileRef:    "work-profile-ref-candidate-001",
		ProfileKind:   kind,
		TaskRef:       "task-workitem-profile-001",
		RunRef:        runRef,
		Title:         "Revisar entrega",
		Objective:     "Registrar revision acotada con evidencias.",
		ScopeRefs:     []string{"docs/revision"},
		AcceptanceCriteria: []string{
			"Revision registrada con refs compactas.",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:review-work:v0",
			FunctionName: "ReviewWorkV0",
		}},
	})
	if err != nil {
		t.Fatalf("WorkflowTaskFromWorkProfileV0: %v", err)
	}
	return task
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
