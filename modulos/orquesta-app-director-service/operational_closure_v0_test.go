package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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

func TestMaybeCloseOperationalDirectorV0PasaRequiredTestEvidenceRefsDelPlanStateAlSource(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-test-evidence"
	planRef := "plan-ref-service-operational-closure-plan-state-test-evidence"
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
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-test-evidence",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-test-evidence",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "state-ref-service-operational-closure-plan-state-test-evidence",
		PlanRef:             planRef,
		RequestRef:          "request-ref-service-operational-closure-plan-state-test-evidence",
		RunRef:              runRef,
		ProjectRef:          "orquesta",
		Mode:                orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        "step-replan-or-close",
		RequiredTestRefs:    []string{"go test ./..."},
		ObservedAt:          "2026-05-17T14:36:00Z",
		ActiveWaveRef:       "wave-service-operational-closure-plan-state",
		ActiveCohortRef:     "cohort-service-operational-closure-plan-state",
		ActiveParentTaskRef: "parent-task-service-operational-closure-plan-state",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:             "step-review-deliveries",
				Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				DeliveryRefs:       []string{"delivery-ref-service-operational-closure-001"},
				ReviewResultRefs:   []string{"review-result-ref-service-operational-closure-001"},
				AcceptedReviewRefs: []string{"accepted-review-ref-service-operational-closure-001"},
			},
			{
				StepID:                   "step-run-required-tests",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			},
			{
				StepID:                   "step-replan-or-close",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:                  "wave-service-operational-closure-plan-state",
				CohortRef:                "cohort-service-operational-closure-plan-state",
				ParentTaskRef:            "parent-task-service-operational-closure-plan-state",
				RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			},
		},
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)

	_, _, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-17T14:36:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   store,
			EventSink:                  sink,
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
			OperationalClosureSource:   source,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil {
		t.Fatalf("maybeCloseOperationalDirectorV0: %v", err)
	}
	if !source.Called ||
		!serviceStringInSetV0(source.LastRequest.RequiredTestEvidenceRefs, "test-evidence-ref-service-operational-closure-001") {
		t.Fatalf("source request sin required_test_evidence_refs: called=%v request=%+v", source.Called, source.LastRequest)
	}
}

func TestMaybeCloseOperationalDirectorV0CierraPlanStateTrasCierreOperativoExitoso(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-closed"
	planRef := "plan-ref-service-operational-closure-plan-state-closed"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-closed",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-closed",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-17T16:10:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
			OperationalClosureSource:   source,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
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
	if loop.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no cerrado: %+v", loop.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		t.Fatalf("state sin active step: %+v", state)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		step.Reason != "operational-closure-succeeded" ||
		!serviceStringInSetV0(state.EvidenceRefs, "closure-ref-service-operational-closure-plan-state-closed") {
		t.Fatalf("plan state no cerrado durablemente: state=%+v step=%+v", state, step)
	}
}

func TestMaybeCloseOperationalDirectorV0BloqueaPlanStateConIssuesDeCierre(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-issues"
	planRef := "plan-ref-service-operational-closure-plan-state-issues"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-issues",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-issues",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-17T16:20:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			OperationalClosureSource:   source,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil {
		t.Fatalf("maybeCloseOperationalDirectorV0: %v", err)
	}
	if len(issues) == 0 || issues[0].Field != "required_test_evidence_store" {
		t.Fatalf("issues=%+v", issues)
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar con issues: %+v", loop.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		t.Fatalf("state sin active step: %+v", state)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-issues" ||
		!serviceStringInSetV0(state.BlockerRefs, "operational-closure-issues") ||
		!serviceStringInSetV0(state.BlockerRefs, "required_test_evidence_store") ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-issues" {
		t.Fatalf("plan state no bloqueado por issues: state=%+v step=%+v", state, step)
	}
}

func TestMaybeCloseOperationalDirectorV0BloqueaPlanStateSiSourceNoConstruyeRequest(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-source-missing"
	planRef := "plan-ref-service-operational-closure-plan-state-source-missing"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{Reject: true}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-17T16:30:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			OperationalClosureSource:   source,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
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
	if !source.Called {
		t.Fatalf("closure source no invocado")
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar sin request de cierre: %+v", loop.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step, ok := operationalDirectorPlanStateActiveStepV0(state)
	if !ok {
		t.Fatalf("state sin active step: %+v", state)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-source-unavailable" ||
		!serviceStringInSetV0(state.BlockerRefs, "operational-closure-source-unavailable") ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-source-unavailable" {
		t.Fatalf("plan state no bloqueado por source: state=%+v step=%+v", state, step)
	}
}

func TestMaybeCloseOperationalDirectorV0NoCierraConPlanStatePostWaitActivo(t *testing.T) {
	for _, stepKind := range []orquestadirectoroperativo.OperationalDirectorStepKindV0{
		orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
		orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
	} {
		t.Run(string(stepKind), func(t *testing.T) {
			runRef := "run-service-operational-closure-plan-state-" + string(stepKind)
			planRef := "plan-ref-service-operational-closure-plan-state-" + string(stepKind)
			run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
			run.Tasks = []string{"task-ref-service-operational-closure-001"}
			run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
			run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
			waitRefs := []string(nil)
			pendingAgentRefs := []string(nil)
			if stepKind == orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0 {
				waitRefs = []string{"wait-ref-service-operational-closure-plan-state"}
				pendingAgentRefs = []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")}
			}
			source := &serviceOperationalClosureSourceForTestV0{
				Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
					TaskID:                   "task-ref-service-operational-closure-001",
					DeliveryRef:              "delivery-ref-service-operational-closure-001",
					AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
					ValidationRef:            "validation-ref-service-operational-closure-plan-state",
					ClosureRef:               "closure-ref-service-operational-closure-plan-state",
					RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
					EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
				},
			}
			state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
				SchemaVersion: orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
				StateRef:      "state-ref-service-operational-closure-plan-state-" + string(stepKind),
				PlanRef:       planRef,
				RequestRef:    "request-ref-service-operational-closure-plan-state",
				RunRef:        runRef,
				ProjectRef:    "orquesta",
				Mode:          orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
				Status:        orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
				ActiveStepID:  "step-active-post-wait",
				ObservedAt:    "2026-05-17T14:35:00Z",
				Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
					StepID:           "step-active-post-wait",
					Kind:             stepKind,
					Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
					TaskRefs:         []string{"task-ref-service-operational-closure-001"},
					AgentRefs:        []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")},
					PendingAgentRefs: pendingAgentRefs,
					WaitRefs:         waitRefs,
				}},
			}
			planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
			loop, issues, err := maybeCloseOperationalDirectorV0(
				context.Background(),
				ContinueAppDirectorRequestV0{
					RunRef:                     runRef,
					OccurredAt:                 "2026-05-17T14:35:00Z",
					OperationalDirectorPlanRef: planRef,
				},
				StartAppDirectorPortsV0{
					RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
					EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
					EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
					DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
					RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
					OperationalClosureSource:   source,
					OperationalPlanStateStore:  planStateStore,
					OperationalPlanStateWriter: planStateStore,
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
			if source.Called {
				t.Fatalf("closure source no debe invocarse con plan state activo en %s", stepKind)
			}
			if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
				t.Fatalf("run cerrado con plan state activo: %+v", loop.Run)
			}
		})
	}
}

type serviceOperationalClosureSourceForTestV0 struct {
	Request     orquestacionnucleoapp.OperationalDirectorClosureRequestV0
	LastRequest AppDirectorOperationalClosureRequestV0
	Called      bool
	Reject      bool
}

func (source *serviceOperationalClosureSourceForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.Called = true
	source.LastRequest = request
	if source.Reject {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
	}
	return source.Request, true, nil
}

func serviceOperationalClosurePlanStateReadyForCloseV0(
	runRef string,
	planRef string,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "state-ref-" + planRef,
		PlanRef:             planRef,
		RequestRef:          "request-ref-" + planRef,
		RunRef:              runRef,
		ProjectRef:          "orquesta",
		Mode:                orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        "step-replan-or-close",
		RequiredTestRefs:    []string{"go test ./..."},
		ObservedAt:          "2026-05-17T16:00:00Z",
		ActiveWaveRef:       "wave-service-operational-closure-plan-state",
		ActiveCohortRef:     "cohort-service-operational-closure-plan-state",
		ActiveParentTaskRef: "parent-task-service-operational-closure-plan-state",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:             "step-review-deliveries",
				Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				DeliveryRefs:       []string{"delivery-ref-service-operational-closure-001"},
				ReviewResultRefs:   []string{"review-result-ref-service-operational-closure-001"},
				AcceptedReviewRefs: []string{"accepted-review-ref-service-operational-closure-001"},
			},
			{
				StepID:                   "step-run-required-tests",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			},
			{
				StepID:                   "step-replan-or-close",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:                  "wave-service-operational-closure-plan-state",
				CohortRef:                "cohort-service-operational-closure-plan-state",
				ParentTaskRef:            "parent-task-service-operational-closure-plan-state",
				TaskRefs:                 []string{"task-ref-service-operational-closure-001"},
				AgentRefs:                []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")},
				DeliveryRefs:             []string{"delivery-ref-service-operational-closure-001"},
				ReviewResultRefs:         []string{"review-result-ref-service-operational-closure-001"},
				RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			},
		},
	}
}

type serviceOperationalClosureEventReaderForTestV0 struct {
	Events []orquestacoreworkflow.OrchestrationEventV0
}

func (reader serviceOperationalClosureEventReaderForTestV0) LoadRunEventsV0(
	_ context.Context,
	_ string,
) ([]orquestacoreworkflow.OrchestrationEventV0, error) {
	return append([]orquestacoreworkflow.OrchestrationEventV0(nil), reader.Events...), nil
}

func newServiceOperationalClosureEventReaderForTestV0(
	t *testing.T,
	runRef string,
) serviceOperationalClosureEventReaderForTestV0 {
	t.Helper()
	return serviceOperationalClosureEventReaderForTestV0{Events: []orquestacoreworkflow.OrchestrationEventV0{
		serviceOperationalClosureEventForTestV0(t, runRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  "delivery-ref-service-operational-closure-001",
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       "task-ref-service-operational-closure-001",
			AgentRef:     orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001"),
			Summary:      "Entrega para cierre operativo.",
			EvidenceRefs: []string{"delivery-ref-service-operational-closure-001"},
		}),
		serviceOperationalClosureEventForTestV0(t, runRef, 2, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: "review-request-ref-service-operational-closure-001",
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     "delivery-ref-service-operational-closure-001",
			Summary:         "Review para cierre operativo.",
		}),
		serviceOperationalClosureEventForTestV0(t, runRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: "review-result-ref-service-operational-closure-001",
			ReviewRequestID: "review-request-ref-service-operational-closure-001",
			DeliveryRef:     "delivery-ref-service-operational-closure-001",
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Review aceptada.",
			EvidenceRefs:    []string{"test-evidence-ref-service-operational-closure-001"},
		}),
		serviceOperationalClosureEventForTestV0(t, runRef, 4, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   "review-request-ref-service-operational-closure-001",
			DeliveryRef:       "delivery-ref-service-operational-closure-001",
			Summary:           "Review aceptada.",
		}),
	}}
}

func serviceOperationalClosureEventForTestV0(
	t *testing.T,
	runRef string,
	sequence int64,
	eventType string,
	payload any,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return orquestacoreworkflow.OrchestrationEventV0{
		EventID:        "evt-service-operational-closure-" + eventType,
		EventType:      eventType,
		RunID:          runRef,
		Sequence:       sequence,
		PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
		Payload:        raw,
	}
}

func serviceOperationalClosureTaskForTestV0(runRef string) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             "task-ref-service-operational-closure-001",
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Task cierre operativo service",
		WriteSet:           []string{"internal/module"},
		AcceptanceCriteria: []string{"entrega revisada"},
		RequiredTests:      []string{"go test ./..."},
	}
}

func serviceOperationalClosureRequiredTestEvidenceForTestV0(
	runRef string,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       "test-evidence-ref-service-operational-closure-001",
		RunRef:            runRef,
		TaskRef:           "task-ref-service-operational-closure-001",
		TestCommand:       "go test ./...",
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       "delivery-ref-service-operational-closure-001",
		ReviewRequestID:   "review-request-ref-service-operational-closure-001",
		ReviewResultRef:   "review-result-ref-service-operational-closure-001",
		AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
		OccurredAt:        "2026-05-17T15:45:00Z",
		EvidenceRefs:      []string{"evidence-ref-service-operational-closure-001"},
	}
}

func mustServiceActiveProgrammingRunForClosureV0(
	t *testing.T,
	runRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.Tasks = nil
	return run
}
