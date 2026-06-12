package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueAppDirectorV0CierraPlanStateReplanOrCloseSinAgentRefs(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-no-agent-refs"
	planRef := "plan-ref-service-operational-closure-plan-state-no-agent-refs"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	for index := range state.Steps {
		if state.Steps[index].StepID == "step-replan-or-close" {
			state.Steps[index].AgentRefs = nil
		}
	}
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-no-agent-refs",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-no-agent-refs",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-plan-state-no-agent-refs"},
		},
	}
	request := serviceContinueClosureRequestForTestV0(runRef)
	request.OperationalDirectorPlanRef = planRef

	result, err := ContinueAppDirectorV0(context.Background(), request, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
		OutboxLedger:               ledger,
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
		RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStore,
		OperationalPlanStateWriter: planStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !source.Called ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(result.Run.Closures, "closure-ref-service-operational-closure-plan-state-no-agent-refs") {
		t.Fatalf("closure no ejecutado: called=%v run=%+v", source.Called, result.Run)
	}
	closed, err := planStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if closed.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		closed.ActiveStepID != "step-replan-or-close" {
		t.Fatalf("plan state no cerrado: %+v", closed)
	}
}

func TestContinueAppDirectorV0CierraPlanStateReplanOrCloseSinAgentRefsTrasRecarga(t *testing.T) {
	ctx := context.Background()
	runRef := "run-service-operational-closure-plan-state-no-agent-refs-reload"
	planRef := "plan-ref-service-operational-closure-plan-state-no-agent-refs-reload"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	state.ActiveWaveRef = ""
	state.ActiveCohortRef = ""
	state.ActiveParentTaskRef = ""
	for index := range state.Steps {
		if state.Steps[index].StepID == "step-replan-or-close" {
			state.Steps[index].WaveRef = ""
			state.Steps[index].CohortRef = ""
			state.Steps[index].ParentTaskRef = ""
			state.Steps[index].AgentRefs = nil
		}
	}
	rootDir := t.TempDir()
	store := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, runRef, newServiceOperationalClosureEventReaderForTestV0(t, runRef).Events); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	if err := store.SaveWorkflowTaskV0(ctx, serviceOperationalClosureTaskForTestV0(runRef)); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := store.SaveRequiredTestEvidenceV0(ctx, serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	recovered := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-no-agent-refs-reload",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-no-agent-refs-reload",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-plan-state-no-agent-refs-reload"},
		},
	}
	request := serviceContinueClosureRequestForTestV0(runRef)
	request.OperationalDirectorPlanRef = planRef
	result, err := ContinueAppDirectorV0(ctx, request, StartAppDirectorPortsV0{
		RunStore:                   recovered,
		EventSink:                  recovered,
		EventReader:                recovered,
		OutboxLedger:               orquestacionnucleoapp.NewInMemoryOutboxLedgerV0(),
		DirectorTaskStore:          recovered,
		RequiredTestEvidenceStore:  recovered,
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  recovered,
		OperationalPlanStateWriter: recovered,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(
				orquestacionnucleoapp.NewInMemoryRunStoreV0(),
				orquestacionnucleoapp.NewInMemoryEventSinkV0(),
				orquestacionnucleoapp.NewInMemoryOutboxLedgerV0(),
			),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 tras recarga: %v", err)
	}
	if !source.Called ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(result.Run.Closures, "closure-ref-service-operational-closure-plan-state-no-agent-refs-reload") {
		t.Fatalf("closure recargado no ejecutado: called=%v run=%+v", source.Called, result.Run)
	}
	closed, err := recovered.LoadOperationalDirectorPlanStateV0(ctx, runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if closed.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		closed.ActiveStepID != "step-replan-or-close" {
		t.Fatalf("plan state recargado no cerrado: %+v", closed)
	}
}
