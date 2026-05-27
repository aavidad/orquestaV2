package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueAppDirectorV0InvocaCierreOperativoTrasLoopQuiescent(t *testing.T) {
	runRef := "run-service-operational-closure-001"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef))
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-001",
			ClosureRef:               "closure-ref-service-operational-closure-001",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}

	result, err := ContinueAppDirectorV0(
		context.Background(),
		serviceContinueClosureRequestForTestV0(runRef),
		StartAppDirectorPortsV0{
			RunStore:                  store,
			EventSink:                 sink,
			EventReader:               newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			OutboxLedger:              ledger,
			DirectorTaskStore:         taskStore,
			RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
			OperationalClosureSource:  source,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !source.Called {
		t.Fatalf("closure source no invocado")
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(result.Run.Closures, "closure-ref-service-operational-closure-001") {
		t.Fatalf("run no cerrado: %+v", result.Run)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0) {
		t.Fatalf("eventos sin RunClosed: %+v", sink.EventsV0())
	}
}

func TestContinueAppDirectorV0CierreOperativoNormalizaRevisionTrasReabrirProgramacion(t *testing.T) {
	runRef := "run-service-operational-closure-reopened-programming"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef))
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-reopened-programming",
			ClosureRef:               "closure-ref-service-operational-closure-reopened-programming",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-reopened-programming"},
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		serviceContinueClosureRequestForTestV0(runRef),
		StartAppDirectorPortsV0{
			RunStore:                  store,
			EventSink:                 sink,
			EventReader:               newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:         taskStore,
			RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
			OperationalClosureSource:  source,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
	}
	if loop.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(loop.Run.ClosedTasks, "task-ref-service-operational-closure-001") ||
		!serviceStringInSetV0(loop.Run.Closures, "closure-ref-service-operational-closure-reopened-programming") {
		t.Fatalf("run no cerrado desde programacion reabierta: %+v", loop.Run)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventPhaseOpenedV0) ||
		!serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventTaskClosedV0) {
		t.Fatalf("eventos sin normalizacion revision/cierre task: %+v", sink.EventsV0())
	}
}

func TestMaybeCloseOperationalDirectorV0CierraNeedsDirectorSiPlanEstaEnReplanOrClose(t *testing.T) {
	runRef := "run-service-operational-closure-needs-director-replan"
	planRef := defaultOperationalDirectorDecisionPlanRefV0(runRef)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef))
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-needs-director",
			ClosureRef:               "closure-ref-service-operational-closure-needs-director",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-needs-director"},
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		func() ContinueAppDirectorRequestV0 {
			request := serviceContinueClosureRequestForTestV0(runRef)
			request.OperationalDirectorPlanRef = planRef
			return request
		}(),
		StartAppDirectorPortsV0{
			RunStore:                   store,
			EventSink:                  sink,
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:          taskStore,
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
			OperationalClosureSource:   source,
			OperationalPlanStateStore:  planStore,
			OperationalPlanStateWriter: planStore,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusNeedsDirectorV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
	}
	if !source.Called ||
		loop.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(loop.Run.Closures, "closure-ref-service-operational-closure-needs-director") {
		t.Fatalf("closure no ejecutado desde needs_director: called=%v loop=%+v", source.Called, loop)
	}
}

func TestContinueAppDirectorV0NoInvocaCierreOperativoMientrasEsperaAgente(t *testing.T) {
	runRef := "run-service-operational-closure-wait-001"
	run := mustServiceActiveProgrammingRunForClosureV0(t, runRef)
	run.Agents = []string{"agent-ref-service-wait-001"}
	run.StartedAgents = []string{"agent-ref-service-wait-001"}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	source := &serviceOperationalClosureSourceForTestV0{}

	result, err := ContinueAppDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:               runRef,
			OccurredAt:           "2026-05-17T13:20:00Z",
			CorrelationID:        "corr-service-operational-closure-wait-001",
			WaitAgentRefs:        []string{"agent-ref-service-wait-001"},
			MaxBursts:            1,
			MaxStepsPerBurst:     1,
			MaxDispatchesPerWait: 1,
		},
		StartAppDirectorPortsV0{
			RunStore:                 store,
			EventSink:                sink,
			OutboxLedger:             ledger,
			DirectorTaskStore:        orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(),
			OperationalClosureSource: source,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("loop_status=%s result=%+v", result.LoopStatus, result)
	}
	if source.Called {
		t.Fatalf("closure source no debe invocarse en wait_external")
	}
}

func TestContinueAppDirectorV0ExponeIssuesDeCierreOperativo(t *testing.T) {
	runRef := "run-service-operational-closure-issues-001"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-issues-001",
			ClosureRef:               "closure-ref-service-operational-closure-issues-001",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-issues-001"},
		},
	}

	result, err := ContinueAppDirectorV0(
		context.Background(),
		serviceContinueClosureRequestForTestV0(runRef),
		StartAppDirectorPortsV0{
			RunStore:                 store,
			EventSink:                sink,
			EventReader:              newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			OutboxLedger:             ledger,
			DirectorTaskStore:        orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			OperationalClosureSource: source,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !source.Called {
		t.Fatalf("closure source no invocado")
	}
	if len(result.OperationalClosureIssues) == 0 ||
		result.OperationalClosureIssues[0].Field != "required_test_evidence_store" {
		t.Fatalf("operational_closure_issues=%+v", result.OperationalClosureIssues)
	}
	if result.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("no debe cerrar con issues: %+v", result.Run)
	}
}

func TestMaybeCloseOperationalDirectorV0PasaScopeResueltoAlSource(t *testing.T) {
	runRef := "run-service-operational-closure-scope-001"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-scope-001",
			ClosureRef:               "closure-ref-service-operational-closure-scope-001",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}

	_, _, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		serviceContinueClosureRequestForTestV0(runRef),
		StartAppDirectorPortsV0{
			RunStore:                  store,
			EventSink:                 sink,
			EventReader:               newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:         orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
			OperationalClosureSource:  source,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{
			WaitAgentRefs:    []string{"agent-ref-resolved"},
			WaitScopeApplied: true,
		},
	)
	if err != nil {
		t.Fatalf("maybeCloseOperationalDirectorV0: %v", err)
	}
	if !source.LastRequest.WaitScopeApplied ||
		len(source.LastRequest.WaitAgentRefs) != 1 ||
		source.LastRequest.WaitAgentRefs[0] != "agent-ref-resolved" {
		t.Fatalf("source request sin scope resuelto: %+v", source.LastRequest)
	}
}
