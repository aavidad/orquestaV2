package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestReviewReworkReplanCandidateProviderV0ProgressiveLoopSplitCreatesSchedulableTasks(t *testing.T) {
	runRef := "run-nucleo-review-rework-split-001"
	taskA := reviewReworkSplitWorkflowTaskV0(runRef, "task-ref-review-split-a", "app/rework_split_a.go")
	taskB := reviewReworkSplitWorkflowTaskV0(runRef, "task-ref-review-split-b", "app/rework_split_b.go")
	run := mustReviewGateReadyRunV0(t, runRef)
	run.FunctionContracts = []string{"contract:function:rework-split:v0"}
	taskStore := NewInMemoryWorkflowTaskStoreV0(
		reviewReworkSplitWorkflowTaskV0(runRef, "task-ref-nucleo-001", "app/rework_original.go"),
	)
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: WorkflowTaskCandidateProviderV0{
			Base: ReviewReworkReplanCandidateProviderV0{
				Base: ReviewGateCandidateProviderV0{
					GateSource:  reviewReworkLoopGateSourceV0{},
					RequestedBy: "orquesta-nucleo-test",
				},
				PlanSource: reviewReworkSplitPlanSourceV0{
					Tasks: []orquestacoreworkflow.WorkflowTaskV0{taskA, taskB},
				},
				TaskWriter:  taskStore,
				RequestedBy: "orquesta-nucleo-test",
			},
			TaskStore:   taskStore,
			RequestedBy: "orquesta-nucleo-test",
		},
		OutboxLedger:      ledger,
		MaxCommands:       12,
		MaxOutboxPerCycle: 4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-10T10:20:00Z",
		MaxBursts:            4,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 4,
		CorrelationID:        "corr-review-rework-split-001",
		EvidenceRefs:         []string{"evidence-ref-review-rework-split-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop split: %v result=%+v", err, result)
	}
	assertReviewReworkSplitRunV0(t, result.Run, taskA.TaskID, taskB.TaskID)
	if _, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskA.TaskID, taskB.TaskID}); err != nil {
		t.Fatalf("split tasks no materializadas en store: %v", err)
	}
	assertReviewReworkProgressiveEventsV0(t, sink, []string{
		orquestacoreworkflow.OrchestrationEventReviewRequestedV0,
		orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0,
		orquestacoreworkflow.OrchestrationEventReworkRequestedV0,
		orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0,
		orquestacoreworkflow.OrchestrationEventPhaseOpenedV0,
		orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0,
		orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0,
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventCapacityDecidedV0,
		orquestacoreworkflow.OrchestrationEventCapacityDecidedV0,
		orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
	})
}

type reviewReworkSplitPlanSourceV0 struct {
	Tasks []orquestacoreworkflow.WorkflowTaskV0
}

func (source reviewReworkSplitPlanSourceV0) BuildReviewReworkReplanPlansV0(
	_ context.Context,
	request ReviewReworkReplanPlanRequestV0,
) ([]ReviewReworkReplanPlanV0, error) {
	if !reviewReworkRunHasRefV0(request.Run.ReworkRequests, "rework-request-ref-nucleo-review-001", "#review_result:") ||
		reviewReworkRunContainsRefV0(request.Run.Agents, WorkflowTaskAgentRequestRefV0("task-ref-review-split-a")) {
		return nil, nil
	}
	return []ReviewReworkReplanPlanV0{reviewReworkSplitPlanV0(request.Run.RunID, source.Tasks)}, nil
}

func reviewReworkSplitPlanV0(
	runRef string,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) ReviewReworkReplanPlanV0 {
	plan := reviewReworkRetryPlanV0(runRef)
	plan.CandidateRef = "review-rework-split-candidate-ref-nucleo-review-001"
	plan.ReplanRef = "replan-ref-nucleo-review-split-001"
	plan.SignalRef = "review-rework-split-signal-ref-nucleo-review-001"
	plan.RequestedAction = orquestacorereplanner.ReplanActionSplitTaskV0
	plan.CapacityRequestRef = ""
	plan.AgentRequestID = ""
	plan.Summary = "Dividir tarea tras revision no aceptada."
	plan.SplitTasks = tasks
	return plan
}

func reviewReworkSplitWorkflowTaskV0(
	runRef string,
	taskRef string,
	writeSet string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Microtarea de retrabajo",
		Summary:       "Trabajo compacto tras revision no aceptada.",
		WriteSet:      []string{writeSet},
		AcceptanceCriteria: []string{
			"Solo se modifica el write-set declarado.",
			"La entrega queda lista para revision.",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
			{ContractRef: "contract:function:rework-split:v0", FunctionName: "NewWorkflowTaskV0"},
		},
	}
}

func assertReviewReworkSplitRunV0(t *testing.T, run orquestacoreworkflow.OrchestrationRunV0, taskA string, taskB string) {
	t.Helper()
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s run=%+v", run.CurrentPhase, run)
	}
	for _, taskRef := range []string{taskA, taskB} {
		if !reviewReworkRunContainsRefV0(run.Tasks, taskRef) {
			t.Fatalf("task %s no autorizada: %v", taskRef, run.Tasks)
		}
		if !reviewReworkRunContainsRefV0(run.CapacityRequests, WorkflowTaskCapacityRequestRefV0(taskRef)) ||
			!reviewReworkRunHasRefV0(run.CapacityDecisions, WorkflowTaskCapacityRequestRefV0(taskRef), "#capacity_decision:") {
			t.Fatalf("capacity para %s incompleta: requests=%v decisions=%v", taskRef, run.CapacityRequests, run.CapacityDecisions)
		}
		if !reviewReworkRunContainsRefV0(run.Agents, WorkflowTaskAgentRequestRefV0(taskRef)) {
			t.Fatalf("agent para %s no solicitado: %v", taskRef, run.Agents)
		}
	}
	if !reviewReworkRunHasRefV0(run.ReworkRequests, "rework-request-ref-nucleo-review-001", "#review_result:") ||
		!reviewReworkRunHasRefV0(run.ReplanDecisions, "replan-ref-nucleo-review-split-001", "#source:") {
		t.Fatalf("rework/replan incompleto: reworks=%v replans=%v", run.ReworkRequests, run.ReplanDecisions)
	}
}
