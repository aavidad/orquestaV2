package orquestacionnucleoapp

import (
	"context"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestOperationalDirectorPlanMaterializerV0CreaMicrotareaParaCandidatos(t *testing.T) {
	runRef := "run-operational-director-materializer-001"
	contractRef := "contract:function:operational-director:v0"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.FunctionContracts = []string{contractRef}
	store := NewInMemoryRunStoreV0(run)
	taskStore := NewInMemoryWorkflowTaskStoreV0()
	sink := NewInMemoryEventSinkV0()
	plan := operationalDirectorProgrammingPlanForMaterializerTestV0(runRef)

	materialized, err := (OperationalDirectorPlanMaterializerV0{
		RunStore:    store,
		EventSink:   sink,
		TaskWriter:  taskStore,
		RequestedBy: "orquesta-nucleo-test",
	}).MaterializeOperationalDirectorPlanV0(context.Background(), OperationalDirectorPlanMaterializeRequestV0{
		Plan:       plan,
		OccurredAt: "2026-05-17T10:30:00Z",
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
		CorrelationID: "corr-operational-director-materializer-001",
	})
	if err != nil {
		t.Fatalf("MaterializeOperationalDirectorPlanV0: %v", err)
	}
	if len(materialized.Issues) != 0 || materialized.EventsCount != 1 || len(materialized.Tasks) != 1 {
		t.Fatalf("materialized=%+v", materialized)
	}
	taskRef := materialized.Tasks[0].TaskID
	run, err = store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if !stringInSetV0(taskRef, run.Tasks) {
		t.Fatalf("run tasks=%v missing=%s", run.Tasks, taskRef)
	}
	storedTasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(storedTasks) != 1 ||
		!stringInSetV0("go test -count=1 ./modulos/orquesta-director-operativo", storedTasks[0].RequiredTests) ||
		!containsFragmentInValuesV0(storedTasks[0].AcceptanceCriteria, "objetivo_actual") {
		t.Fatalf("stored tasks=%+v", storedTasks)
	}

	ledger := NewInMemoryOutboxLedgerV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: WorkflowTaskCandidateProviderV0{
			TaskStore:   taskStore,
			RequestedBy: "orquesta-nucleo-test",
		},
		OutboxLedger: ledger,
	}
	burst, err := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
		RunRef:        runRef,
		OccurredAt:    "2026-05-17T10:31:00Z",
		MaxSteps:      1,
		CorrelationID: "corr-operational-director-materializer-burst-001",
	})
	if err != nil {
		t.Fatalf("RunSupervisedBurstV0: %v", err)
	}
	if burst.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
		t.Fatalf("final action=%s", burst.Burst.FinalAction)
	}
	pending, issues := ledger.ListPending(context.Background(), orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{RunRef: runRef})
	if len(issues) > 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func containsFragmentInValuesV0(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func TestOperationalDirectorPlanMaterializerV0NoLanzaPlanBloqueado(t *testing.T) {
	runRef := "run-operational-director-materializer-blocked-001"
	planResult := orquestadirectoroperativo.BuildOperationalDirectorPlanV0(orquestadirectoroperativo.OperationalDirectorRequestV0{
		RequestRef:     "req-operational-director-blocked-001",
		RunRef:         runRef,
		ProjectRef:     "external-domain",
		Objective:      "Resolver trabajo documental externo.",
		Mode:           orquestadirectoroperativo.OperationalDirectorModeDomainWorkV0,
		ContextStatus:  orquestadirectoroperativo.OperationalDirectorContextInsufficientV0,
		MissingContext: []string{"topic_outline"},
	})
	result, err := (OperationalDirectorPlanMaterializerV0{
		RunStore:   NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef)),
		TaskWriter: NewInMemoryWorkflowTaskStoreV0(),
	}).MaterializeOperationalDirectorPlanV0(context.Background(), OperationalDirectorPlanMaterializeRequestV0{
		Plan:       planResult.Plan,
		OccurredAt: "2026-05-17T10:35:00Z",
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef: "contract:function:operational-director:v0",
		}},
	})
	if err != nil {
		t.Fatalf("MaterializeOperationalDirectorPlanV0: %v", err)
	}
	if len(result.Issues) == 0 || len(result.Tasks) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestOperationalDirectorPlanMaterializerV0NoGuardaTaskSiWorkflowRechaza(t *testing.T) {
	runRef := "run-operational-director-materializer-reject-001"
	contractRef := "contract:function:operational-director:v0"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{{
		ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
		RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
	}}
	run.FunctionContracts = []string{contractRef}
	taskStore := NewInMemoryWorkflowTaskStoreV0()

	_, err := (OperationalDirectorPlanMaterializerV0{
		RunStore:    NewInMemoryRunStoreV0(run),
		EventSink:   NewInMemoryEventSinkV0(),
		TaskWriter:  taskStore,
		RequestedBy: "orquesta-nucleo-test",
	}).MaterializeOperationalDirectorPlanV0(context.Background(), OperationalDirectorPlanMaterializeRequestV0{
		Plan:       operationalDirectorProgrammingPlanForMaterializerTestV0(runRef),
		OccurredAt: "2026-05-17T10:37:00Z",
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
		CorrelationID: "corr-operational-director-materializer-reject-001",
	})
	if err == nil {
		t.Fatal("err=nil, want workflow rejection")
	}
	if _, loadErr := taskStore.LoadWorkflowTasksV0(
		context.Background(),
		runRef,
		[]string{"task-operational-director-req-operational-director-materializer-001-step-launch-subagents"},
	); loadErr == nil {
		t.Fatal("task store contiene task rechazada por workflow")
	}
}

func TestOperationalDirectorPlanMaterializerV0PreservaRefsOperativasTipadas(t *testing.T) {
	runRef := "run-operational-director-materializer-refs-001"
	contractRef := "contract:function:operational-director:v0"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.FunctionContracts = []string{contractRef}
	plan := operationalDirectorProgrammingPlanForMaterializerTestV0(runRef)
	plan.RequestRef = "req-operational-director-materializer-refs-001"
	for index := range plan.Steps {
		if plan.Steps[index].Kind != orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0 {
			continue
		}
		plan.Steps[index].ParentStepID = "step-split-work"
		plan.Steps[index].DelegationDepth = 1
		plan.Steps[index].MaxChildAgents = 2
		plan.Steps[index].ChildStepIDs = []string{"step-wait-subagents", "step-review-deliveries"}
	}

	materialized, err := (OperationalDirectorPlanMaterializerV0{
		RunStore:    NewInMemoryRunStoreV0(run),
		EventSink:   NewInMemoryEventSinkV0(),
		TaskWriter:  NewInMemoryWorkflowTaskStoreV0(),
		RequestedBy: "orquesta-nucleo-test",
	}).MaterializeOperationalDirectorPlanV0(context.Background(), OperationalDirectorPlanMaterializeRequestV0{
		Plan:       plan,
		OccurredAt: "2026-05-17T10:40:00Z",
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
		CorrelationID: "corr-operational-director-materializer-refs-001",
	})
	if err != nil {
		t.Fatalf("MaterializeOperationalDirectorPlanV0: %v", err)
	}
	if len(materialized.Issues) != 0 || len(materialized.Tasks) != 1 {
		t.Fatalf("materialized=%+v", materialized)
	}
	task := materialized.Tasks[0]
	if task.TaskID != "task-operational-director-req-operational-director-materializer-refs-001-step-launch-subagents" {
		t.Fatalf("task id=%s", task.TaskID)
	}
	if task.WaveRef != "wave-03" ||
		task.CohortRef != "cohort-operational-director-req-operational-director-materializer-refs-001-wave-03" ||
		task.ParentTaskRef != "" ||
		len(task.ChildTaskRefs) != 0 ||
		task.DelegationDepth != 1 ||
		task.MaxChildAgents != 2 {
		t.Fatalf("lineage metadata=%+v", task)
	}
	for _, expected := range []string{
		"operational_director.plan_ref",
		"operational_director.request_ref: req-operational-director-materializer-refs-001",
		"operational_director.wave_ref: wave-03",
		"operational_director.source_step_ref: step-launch-subagents",
		"operational_director.source_item_ref: work-item-step-launch-subagents",
		"operational_director.parent_item_ref: work-item-step-split-work",
		"operational_director.child_item_refs: work-item-step-wait-subagents,work-item-step-review-deliveries",
	} {
		if !containsFragmentInValuesV0(task.AcceptanceCriteria, expected) {
			t.Fatalf("criteria=%v missing=%s", task.AcceptanceCriteria, expected)
		}
	}
	if containsFragmentInValuesV0(task.AcceptanceCriteria, "operational_director.parent_task_ref") {
		t.Fatalf("criteria should not invent parent_task_ref when parent item was not materialized: %v", task.AcceptanceCriteria)
	}
}

func TestOperationalDirectorPlanMaterializerV0UsaFaseObjetivo(t *testing.T) {
	runRef := "run-operational-director-materializer-target-phase-001"
	contractRef := "contract:function:operational-director:v0"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.FunctionContracts = []string{contractRef}

	materialized, err := (OperationalDirectorPlanMaterializerV0{
		RunStore:    NewInMemoryRunStoreV0(run),
		EventSink:   NewInMemoryEventSinkV0(),
		TaskWriter:  NewInMemoryWorkflowTaskStoreV0(),
		RequestedBy: "orquesta-nucleo-test",
	}).MaterializeOperationalDirectorPlanV0(context.Background(), OperationalDirectorPlanMaterializeRequestV0{
		Plan:          operationalDirectorProgrammingPlanForMaterializerTestV0(runRef),
		TargetPhaseID: orquestacoreworkflow.OrchestrationPhaseDocumentacionV0,
		OccurredAt:    "2026-05-17T10:45:00Z",
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
		CorrelationID: "corr-operational-director-materializer-target-phase-001",
	})
	if err != nil {
		t.Fatalf("MaterializeOperationalDirectorPlanV0: %v", err)
	}
	if len(materialized.Issues) != 0 || len(materialized.Tasks) != 1 {
		t.Fatalf("materialized=%+v", materialized)
	}
	if materialized.Tasks[0].PhaseID != orquestacoreworkflow.OrchestrationPhaseDocumentacionV0 {
		t.Fatalf("phase_id=%s", materialized.Tasks[0].PhaseID)
	}
}

func operationalDirectorProgrammingPlanForMaterializerTestV0(
	runRef string,
) orquestadirectoroperativo.OperationalDirectorPlanV0 {
	result := orquestadirectoroperativo.BuildOperationalDirectorPlanV0(orquestadirectoroperativo.OperationalDirectorRequestV0{
		RequestRef:       "req-operational-director-materializer-001",
		RunRef:           runRef,
		ProjectRef:       "orquesta",
		Objective:        "Implementar corte pequeno del Director Operativo.",
		Mode:             orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		WorktreeRef:      "worktree-ref-operational-director",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-operational-director",
		WriteSet: []string{
			"modulos/orquesta-director-operativo/plan_v0.go",
			"modulos/orquesta-director-operativo/plan_v0_test.go",
		},
		RequiredTests: []string{"go test -count=1 ./modulos/orquesta-director-operativo"},
	})
	if !result.Accepted || !result.ReadyToLaunch {
		panic("invalid materializer test plan")
	}
	return result.Plan
}
