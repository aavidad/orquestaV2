package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
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

func TestMaybeCloseOperationalDirectorV0MarcaWaitStateClearedAlCerrarPlanState(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-wait-cleared"
	planRef := "plan-ref-service-operational-closure-plan-state-wait-cleared"
	waitRef := "wait-ref-service-operational-closure-plan-state-wait-cleared"
	taskRef := "task-ref-service-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{taskRef}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	waitStep := orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		StepID:           "step-wait-subagents",
		Kind:             orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		Status:           orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		WaveRef:          state.ActiveWaveRef,
		CohortRef:        state.ActiveCohortRef,
		ParentTaskRef:    state.ActiveParentTaskRef,
		TaskRefs:         []string{taskRef},
		WaitRefs:         []string{waitRef},
		AgentRefs:        []string{agentRef},
		PendingAgentRefs: []string{agentRef},
	}
	state.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{waitStep}, state.Steps...)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	waitStateStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0(
		orquestacionnucleoapp.WorkflowTaskWaitStateV0{
			SchemaVersion:    orquestacionnucleoapp.WorkflowTaskWaitStateSchemaVersionV0,
			WaitRef:          waitRef,
			RunRef:           runRef,
			ReasonCode:       orquestacionnucleoapp.WorkflowTaskWaitReasonCohortInProgressV0,
			CohortRef:        state.ActiveCohortRef,
			WaveRef:          state.ActiveWaveRef,
			ParentTaskRef:    state.ActiveParentTaskRef,
			TaskRefs:         []string{taskRef},
			AgentRefs:        []string{agentRef},
			PendingAgentRefs: []string{agentRef},
			Status:           orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0,
			Attempt:          1,
			ObservedAt:       "2026-05-22T13:00:00Z",
		},
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   taskRef,
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-wait-cleared",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-wait-cleared",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T13:10:00Z",
		OperationalDirectorPlanRef: planRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
		RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		WaitStateStore:             waitStateStore,
		WaitStateWriter:            waitStateStore,
	}
	first, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first err=%v issues=%+v", err, issues)
	}
	waitState, err := waitStateStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 first: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusClearedV0 ||
		len(waitState.PendingAgentRefs) != 0 ||
		waitState.ObservedAt != "2026-05-22T13:10:00Z" ||
		serviceCountStringV0(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-cleared-v0") != 1 {
		t.Fatalf("waitState no cleared: %+v", waitState)
	}
	request.OccurredAt = "2026-05-22T13:10:01Z"
	_, issues, err = maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    first.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 replay err=%v issues=%+v", err, issues)
	}
	replayedWaitState, err := waitStateStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 replay: %v", err)
	}
	if replayedWaitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusClearedV0 ||
		serviceCountStringV0(replayedWaitState.EvidenceRefs, "evidence-ref-app-director-wait-state-cleared-v0") != 1 {
		t.Fatalf("waitState replay no idempotente: %+v", replayedWaitState)
	}
}

func TestMaybeCloseOperationalDirectorV0ReplayNoDuplicaCierreNiPlanState(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-closed-replay"
	planRef := "plan-ref-service-operational-closure-plan-state-closed-replay"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   "task-ref-service-operational-closure-001",
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-closed-replay",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-closed-replay",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T10:10:00Z",
		OperationalDirectorPlanRef: planRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
		RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    run,
	}

	first, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		loop,
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first err=%v issues=%+v", err, issues)
	}
	request.OccurredAt = "2026-05-22T10:10:01Z"
	replay, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    first.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 replay err=%v issues=%+v", err, issues)
	}
	if replay.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(replay.Run.Closures) != 1 ||
		replay.Run.Closures[0] != "closure-ref-service-operational-closure-plan-state-closed-replay" ||
		len(replay.Run.Validations) != 1 ||
		len(replay.Run.ClosedTasks) != 1 {
		t.Fatalf("run replay no idempotente: %+v", replay.Run)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0); got != 1 {
		t.Fatalf("RunClosed duplicado: got=%d events=%+v", got, eventSink.EventsV0())
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" ||
		len(state.BlockerRefs) != 0 ||
		serviceCountStringV0(state.EvidenceRefs, "closure-ref-service-operational-closure-plan-state-closed-replay") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-succeeded-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		serviceCountStringV0(step.RequiredTestEvidenceRefs, "test-evidence-ref-service-operational-closure-001") != 1 {
		t.Fatalf("plan state replay no idempotente: state=%+v step=%+v", state, step)
	}
}

func TestMaybeCloseOperationalDirectorV0CierraOlaMultitareaPorReentradasSinBloquear(t *testing.T) {
	runRef := "run-service-operational-closure-multitask-wave"
	planRef := "plan-ref-service-operational-closure-multitask-wave"
	parentTaskRef := "parent-task-service-operational-closure-wave"
	childA := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-service-operational-closure-child-a",
		DeliveryRef:     "delivery-ref-service-operational-closure-child-a",
		ReviewRequestID: "review-request-ref-service-operational-closure-child-a",
		ReviewResultRef: "review-result-ref-service-operational-closure-child-a",
		AcceptedRef:     "accepted-review-ref-service-operational-closure-child-a",
		TestEvidenceRef: "test-evidence-ref-service-operational-closure-child-a",
		ValidationRef:   "validation-ref-service-operational-closure-child-a",
		ClosureRef:      "closure-ref-service-operational-closure-child-a",
	}
	childB := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-service-operational-closure-child-b",
		DeliveryRef:     "delivery-ref-service-operational-closure-child-b",
		ReviewRequestID: "review-request-ref-service-operational-closure-child-b",
		ReviewResultRef: "review-result-ref-service-operational-closure-child-b",
		AcceptedRef:     "accepted-review-ref-service-operational-closure-child-b",
		TestEvidenceRef: "test-evidence-ref-service-operational-closure-child-b",
		ValidationRef:   "validation-ref-service-operational-closure-child-b",
		ClosureRef:      "closure-ref-service-operational-closure-child-b",
	}
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{childA.TaskRef, childB.TaskRef}
	run.Deliveries = []string{childA.DeliveryRef, childB.DeliveryRef}
	run.DeliveredTasks = []string{childA.TaskRef, childB.TaskRef}
	run.Reviews = []string{childA.ReviewRequestID, childB.ReviewRequestID}
	run.ReviewResults = []string{
		serviceOperationalClosureReviewResultProjectionForTestV0(childA),
		serviceOperationalClosureReviewResultProjectionForTestV0(childB),
	}
	run.AcceptedReviews = []string{childA.AcceptedRef, childB.AcceptedRef}
	run.Agents = []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childA.TaskRef),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskRef),
	}
	run.StartedAgents = append([]string(nil), run.Agents...)
	run.DeliveredAgents = append([]string(nil), run.Agents...)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseTasksV0(runRef, planRef, parentTaskRef, childA, childB),
	)
	source := &serviceOperationalClosureOpenTaskSourceForTestV0{
		Requests: map[string]orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			childA.TaskRef: serviceOperationalClosureRequestForTaskRefsV0(childA),
			childB.TaskRef: serviceOperationalClosureRequestForTaskRefsV0(childB),
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T12:00:00Z",
		OperationalDirectorPlanRef: planRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:  runStore,
		EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		EventReader: serviceOperationalClosureEventReaderForTestV0{
			Events: serviceOperationalClosureEventsForTasksForTestV0(t, runRef, childA, childB),
		},
		DirectorTaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
			serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childA),
			serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childB),
		),
		RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
			serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childA),
			serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childB),
		),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}

	first, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first err=%v issues=%+v", err, issues)
	}
	if first.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		!serviceStringInSetV0(first.Run.ClosedTasks, childA.TaskRef) ||
		serviceStringInSetV0(first.Run.ClosedTasks, childB.TaskRef) {
		t.Fatalf("primer cierre parcial incorrecto: %+v", first.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 first: %v", err)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 {
		t.Fatalf("plan state no debe bloquearse tras cierre parcial: %+v", state)
	}

	request.OccurredAt = "2026-05-22T12:00:01Z"
	second, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    first.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 second err=%v issues=%+v", err, issues)
	}
	if second.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(second.Run.ClosedTasks, childA.TaskRef) ||
		!serviceStringInSetV0(second.Run.ClosedTasks, childB.TaskRef) ||
		!serviceStringInSetV0(second.Run.Closures, childB.ClosureRef) {
		t.Fatalf("segundo cierre no completo la ola: %+v", second.Run)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 second: %v", err)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" {
		t.Fatalf("plan state no cerrado al final de la ola: %+v", state)
	}
	if len(source.TaskCalls) != 2 ||
		source.TaskCalls[0] != childA.TaskRef ||
		source.TaskCalls[1] != childB.TaskRef {
		t.Fatalf("source no avanzo por tareas abiertas: %+v", source.TaskCalls)
	}
}

func TestMaybeCloseOperationalDirectorV0ReplanCausalCierreInsuficienteEnOlaMultitarea(t *testing.T) {
	runRef := "run-service-operational-closure-multitask-replan"
	planRef := "plan-ref-service-operational-closure-multitask-replan"
	parentTaskRef := "parent-task-service-operational-closure-multitask-replan"
	childA := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-service-operational-closure-multitask-replan-a",
		DeliveryRef:     "delivery-ref-service-operational-closure-multitask-replan-a",
		ReviewRequestID: "review-request-ref-service-operational-closure-multitask-replan-a",
		ReviewResultRef: "review-result-ref-service-operational-closure-multitask-replan-a",
		AcceptedRef:     "accepted-review-ref-service-operational-closure-multitask-replan-a",
		TestEvidenceRef: "test-evidence-ref-service-operational-closure-multitask-replan-a",
		ValidationRef:   "validation-ref-service-operational-closure-multitask-replan-a",
		ClosureRef:      "closure-ref-service-operational-closure-multitask-replan-a",
	}
	childB := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-service-operational-closure-multitask-replan-b",
		DeliveryRef:     "delivery-ref-service-operational-closure-multitask-replan-b",
		ReviewRequestID: "review-request-ref-service-operational-closure-multitask-replan-b",
		ReviewResultRef: "review-result-ref-service-operational-closure-multitask-replan-b",
		AcceptedRef:     "accepted-review-ref-service-operational-closure-multitask-replan-b",
		TestEvidenceRef: "test-evidence-ref-service-operational-closure-multitask-replan-b",
		ValidationRef:   "validation-ref-service-operational-closure-multitask-replan-b",
	}
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{childA.TaskRef, childB.TaskRef}
	run.ClosedTasks = []string{childA.TaskRef}
	run.Deliveries = []string{childA.DeliveryRef, childB.DeliveryRef}
	run.DeliveredTasks = []string{childA.TaskRef, childB.TaskRef}
	run.Reviews = []string{childA.ReviewRequestID, childB.ReviewRequestID}
	run.ReviewResults = []string{
		serviceOperationalClosureReviewResultProjectionForTestV0(childA),
		serviceOperationalClosureReviewResultProjectionForTestV0(childB),
	}
	run.AcceptedReviews = []string{childA.AcceptedRef, childB.AcceptedRef}
	run.Agents = []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childA.TaskRef),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskRef),
	}
	run.StartedAgents = append([]string(nil), run.Agents...)
	run.DeliveredAgents = append([]string(nil), run.Agents...)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	closureEvents := serviceOperationalClosureEventsForTasksForTestV0(t, runRef, childA, childB)
	planState := serviceOperationalClosurePlanStateReadyForCloseTasksV0(runRef, planRef, parentTaskRef, childA, childB)
	planState.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
		StepID:        "step-wait-subagents",
		Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		WaveRef:       planState.ActiveWaveRef,
		CohortRef:     planState.ActiveCohortRef,
		ParentTaskRef: parentTaskRef,
		TaskRefs:      []string{childA.TaskRef, childB.TaskRef},
		AgentRefs: []string{
			orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childA.TaskRef),
			orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskRef),
		},
		WaitRefs: []string{"wait-ref-service-operational-closure-multitask-replan"},
	}}, planState.Steps...)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(planState)
	source := &serviceOperationalClosureOpenTaskSourceForTestV0{
		Requests: map[string]orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			childB.TaskRef: {
				TaskID:                   childB.TaskRef,
				DeliveryRef:              childB.DeliveryRef,
				AcceptedReviewRef:        childB.AcceptedRef,
				ValidationRef:            childB.ValidationRef,
				RequiredTestEvidenceRefs: []string{childB.TestEvidenceRef},
				EvidenceRefs:             []string{"evidence-ref-service-operational-closure-multitask-replan-b"},
			},
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T20:10:00Z",
		CorrelationID:              "corr-service-operational-closure-multitask-replan",
		RequestedBy:                "test",
		OperationalDirectorPlanRef: planRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:  runStore,
		EventSink: eventSink,
		EventReader: serviceOperationalClosureEventReaderForTestV0{
			Events: closureEvents,
		},
		DirectorTaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
			serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childA),
			serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childB),
		),
		RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
			serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childA),
			serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childB),
		),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}

	_, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil {
		t.Fatalf("maybeCloseOperationalDirectorV0: %v", err)
	}
	if len(issues) == 0 || issues[0].Field != "closure_ref" {
		t.Fatalf("issues=%+v", issues)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 1 {
		t.Fatalf("QualityGateRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	updatedRun, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(updatedRun.QualityGates) != 1 ||
		!strings.Contains(updatedRun.QualityGates[0], "#subject:"+childB.TaskRef) ||
		len(updatedRun.ReplanDecisions) != 1 ||
		!strings.Contains(updatedRun.ReplanDecisions[0], "#task:"+childB.TaskRef) {
		t.Fatalf("run sin replan causal de childB: %+v", updatedRun)
	}
	followupAgentRef := serviceFirstAgentFollowupFromReplanProjectionForTestV0(t, updatedRun.ReplanDecisions[0])
	updatedRun.Agents = append(updatedRun.Agents, followupAgentRef)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T20:10:01Z",
		CorrelationID:              "corr-service-operational-closure-multitask-replan",
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(updatedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: append(closureEvents, eventSink.EventsV0()...)},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	oldAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskRef)
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, oldAgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, oldAgentRef)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reentered: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	closureStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		closureStep.Reason != "operational-closure-issues-replan-recorded" {
		t.Fatalf("state=%+v waitStep=%+v closureStep=%+v", state, waitStep, closureStep)
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

func TestMaybeCloseOperationalDirectorV0ReplanCausalSiFaltaEvidenciaDeTestEnReplanOrClose(t *testing.T) {
	runRef := "run-service-operational-closure-missing-test-replan"
	planRef := "plan-ref-service-operational-closure-missing-test-replan"
	taskRef := "task-ref-service-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{taskRef}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.DeliveredTasks = []string{taskRef}
	run.Reviews = []string{"review-request-ref-service-operational-closure-001"}
	run.ReviewResults = []string{
		"review-result-ref-service-operational-closure-001#review_result:accepted#review_request:review-request-ref-service-operational-closure-001#delivery:delivery-ref-service-operational-closure-001",
	}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	planState := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	for i := range planState.Steps {
		planState.Steps[i].RequiredTestEvidenceRefs = nil
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(planState)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:            taskRef,
			DeliveryRef:       "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
			ValidationRef:     "validation-ref-service-operational-closure-missing-test-replan",
			ClosureRef:        "closure-ref-service-operational-closure-missing-test-replan",
			EvidenceRefs:      []string{"evidence-ref-service-operational-closure-missing-test-replan"},
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T13:10:00Z",
			CorrelationID:              "corr-service-operational-closure-missing-test-replan",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   runStore,
			EventSink:                  eventSink,
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(),
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
	if len(issues) == 0 || issues[0].Field != "required_test_evidence_refs" {
		t.Fatalf("issues=%+v", issues)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 1 {
		t.Fatalf("QualityGateRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar sin evidencia de tests: %+v", loop.Run)
	}
	updatedRun, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(updatedRun.QualityGates) != 1 ||
		len(updatedRun.ReplanDecisions) != 1 ||
		!strings.Contains(updatedRun.QualityGates[0], "#decision:blocked#subject:"+taskRef) ||
		!strings.Contains(updatedRun.ReplanDecisions[0], "#action:retry_task#followups:") {
		t.Fatalf("run sin quality gate/replan causal: %+v", updatedRun)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 1 {
		t.Fatalf("QualityGateRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-issues" ||
		!serviceStringInSetV0(state.BlockerRefs, "required_test_evidence_refs") ||
		!serviceStringInSetV0(step.BlockerRefs, "required_test_evidence_refs") ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") {
		t.Fatalf("plan state no bloqueo causalmente: state=%+v step=%+v", state, step)
	}
}

func TestMaybeCloseOperationalDirectorV0ReplanCausalSiCierreInsuficienteConReviewAceptada(t *testing.T) {
	runRef := "run-service-operational-closure-insufficient-replan"
	planRef := "plan-ref-service-operational-closure-insufficient-replan"
	taskRef := "task-ref-service-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{taskRef}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.DeliveredTasks = []string{taskRef}
	run.Reviews = []string{"review-request-ref-service-operational-closure-001"}
	run.ReviewResults = []string{
		"review-result-ref-service-operational-closure-001#review_result:accepted#review_request:review-request-ref-service-operational-closure-001#delivery:delivery-ref-service-operational-closure-001",
	}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	closureEvents := newServiceOperationalClosureEventReaderForTestV0(t, runRef).Events
	planState := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	planState.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		{
			StepID:        "step-wait-subagents",
			Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
			Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			WaveRef:       planState.ActiveWaveRef,
			CohortRef:     planState.ActiveCohortRef,
			ParentTaskRef: planState.ActiveParentTaskRef,
			TaskRefs:      []string{taskRef},
			AgentRefs:     []string{agentRef},
			WaitRefs:      []string{"wait-ref-service-operational-closure-insufficient-replan"},
		},
	}, planState.Steps...)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(planState)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   taskRef,
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-insufficient-replan",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-insufficient-replan"},
		},
	}

	_, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T19:10:00Z",
			CorrelationID:              "corr-service-operational-closure-insufficient-replan",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   runStore,
			EventSink:                  eventSink,
			EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: closureEvents},
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
	if len(issues) == 0 || issues[0].Field != "closure_ref" {
		t.Fatalf("issues=%+v", issues)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 1 {
		t.Fatalf("QualityGateRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0); got != 0 {
		t.Fatalf("RunClosed=%d events=%+v", got, eventSink.EventsV0())
	}
	_, replayIssues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T19:10:01Z",
			CorrelationID:              "corr-service-operational-closure-insufficient-replan",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   runStore,
			EventSink:                  eventSink,
			EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: append(closureEvents, eventSink.EventsV0()...)},
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
		t.Fatalf("maybeCloseOperationalDirectorV0 replay: %v", err)
	}
	if len(replayIssues) != 0 {
		t.Fatalf("replayIssues=%+v", replayIssues)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != 1 {
		t.Fatalf("QualityGateRecorded replay=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded replay=%d events=%+v", got, eventSink.EventsV0())
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0); got != 0 {
		t.Fatalf("RunClosed replay=%d events=%+v", got, eventSink.EventsV0())
	}
	blockedState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	closureStep := serviceOperationalDirectorPlanStateStepForTestV0(t, blockedState, "step-replan-or-close")
	if blockedState.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		blockedState.ClosureReason != "operational-closure-issues" ||
		!serviceStringInSetV0(blockedState.BlockerRefs, "closure_ref") ||
		!serviceStringInSetV0(blockedState.BlockerRefs, "operational_closure_insufficient") ||
		closureStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		closureStep.Reason != "operational-closure-issues" {
		t.Fatalf("blockedState=%+v closureStep=%+v", blockedState, closureStep)
	}

	updatedRun, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	followupAgentRef := serviceFirstAgentFollowupFromReplanProjectionForTestV0(t, updatedRun.ReplanDecisions[0])
	updatedRun.Agents = append(updatedRun.Agents, followupAgentRef)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T19:10:02Z",
		CorrelationID:              "corr-service-operational-closure-insufficient-replan",
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(updatedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: append(closureEvents, eventSink.EventsV0()...)},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, agentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, agentRef)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reentered: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, agentRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) {
		t.Fatalf("state=%+v waitStep=%+v", state, waitStep)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0MantieneClosureIssuesBloqueadoHastaFollowup(t *testing.T) {
	runRef := "run-service-operational-closure-insufficient-replan-wait-followup"
	planRef := "plan-ref-service-operational-closure-insufficient-replan-wait-followup"
	taskRef := "task-ref-service-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{taskRef}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.DeliveredTasks = []string{taskRef}
	run.Reviews = []string{"review-request-ref-service-operational-closure-001"}
	run.ReviewResults = []string{
		"review-result-ref-service-operational-closure-001#review_result:accepted#review_request:review-request-ref-service-operational-closure-001#delivery:delivery-ref-service-operational-closure-001",
	}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	closureEvents := newServiceOperationalClosureEventReaderForTestV0(t, runRef).Events
	planState := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	planState.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
		StepID:        "step-wait-subagents",
		Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
		Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
		WaveRef:       planState.ActiveWaveRef,
		CohortRef:     planState.ActiveCohortRef,
		ParentTaskRef: planState.ActiveParentTaskRef,
		TaskRefs:      []string{taskRef},
		AgentRefs:     []string{agentRef},
		WaitRefs:      []string{"wait-ref-service-operational-closure-insufficient-replan-wait-followup"},
	}}, planState.Steps...)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(planState)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   taskRef,
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-insufficient-replan-wait-followup",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-insufficient-replan-wait-followup"},
		},
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: closureEvents},
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
		RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}

	_, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T23:10:00Z",
			CorrelationID:              "corr-service-operational-closure-insufficient-replan-wait-followup",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil {
		t.Fatalf("maybeCloseOperationalDirectorV0: %v", err)
	}
	if len(issues) == 0 || issues[0].Field != "closure_ref" {
		t.Fatalf("issues=%+v", issues)
	}
	updatedRun, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T23:10:01Z",
		CorrelationID:              "corr-service-operational-closure-insufficient-replan-wait-followup",
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(updatedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: append(closureEvents, eventSink.EventsV0()...)},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 no debe fallar esperando followup: %v", err)
	}
	if len(reentered.WaitAgentRefs) != 0 {
		t.Fatalf("reentered no debe esperar hasta que followup exista: %+v", reentered)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reentered: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-issues" ||
		state.ReplanAttempts != 0 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-issues" {
		t.Fatalf("state debe seguir bloqueado hasta followup: state=%+v step=%+v", state, step)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReabreClosureIssuesConFollowupTardio(t *testing.T) {
	runRef := "run-service-operational-closure-late-followup"
	planRef := "plan-ref-service-operational-closure-late-followup"
	taskRef := "task-ref-service-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{taskRef}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.DeliveredTasks = []string{taskRef}
	run.Reviews = []string{"review-request-ref-service-operational-closure-001"}
	run.ReviewResults = []string{
		"review-result-ref-service-operational-closure-001#review_result:accepted#review_request:review-request-ref-service-operational-closure-001#delivery:delivery-ref-service-operational-closure-001",
	}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	closureEvents := newServiceOperationalClosureEventReaderForTestV0(t, runRef).Events
	planState := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	planState.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		{
			StepID:        "step-wait-subagents",
			Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
			Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			WaveRef:       planState.ActiveWaveRef,
			CohortRef:     planState.ActiveCohortRef,
			ParentTaskRef: planState.ActiveParentTaskRef,
			TaskRefs:      []string{taskRef},
			AgentRefs:     []string{agentRef},
			WaitRefs:      []string{"wait-ref-service-operational-closure-late-followup"},
		},
	}, planState.Steps...)
	for i := range planState.Steps {
		planState.Steps[i].RequiredTestEvidenceRefs = nil
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(planState)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:            taskRef,
			DeliveryRef:       "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
			ValidationRef:     "validation-ref-service-operational-closure-late-followup",
			ClosureRef:        "closure-ref-service-operational-closure-late-followup",
			EvidenceRefs:      []string{"evidence-ref-service-operational-closure-late-followup"},
		},
	}

	_, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T18:30:00Z",
			CorrelationID:              "corr-service-operational-closure-late-followup",
			RequestedBy:                "test",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   runStore,
			EventSink:                  eventSink,
			EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: closureEvents},
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(),
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
	if len(issues) == 0 || issues[0].Field != "required_test_evidence_refs" {
		t.Fatalf("issues=%+v", issues)
	}

	blockedState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	closureStep := serviceOperationalDirectorPlanStateStepForTestV0(t, blockedState, "step-replan-or-close")
	if blockedState.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		closureStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		closureStep.Reason != "operational-closure-issues" ||
		!serviceStringInSetV0(closureStep.BlockerRefs, "required_test_evidence_refs") ||
		blockedState.ReplanAttempts != 0 {
		t.Fatalf("blockedState=%+v closureStep=%+v", blockedState, closureStep)
	}

	updatedRun, err := runStore.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	followupAgentRef := serviceFirstAgentFollowupFromReplanProjectionForTestV0(t, updatedRun.ReplanDecisions[0])
	updatedRun.Agents = append(updatedRun.Agents, followupAgentRef)
	events := append(closureEvents, eventSink.EventsV0()...)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T18:30:01Z",
		CorrelationID:              "corr-service-operational-closure-late-followup",
		OperationalDirectorPlanRef: planRef,
		WaitAgentRefs:              []string{agentRef},
		WaitWaveRef:                planState.ActiveWaveRef,
		WaitCohortRef:              planState.ActiveCohortRef,
		WaitParentTaskRef:          planState.ActiveParentTaskRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(updatedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if reentered.WaitWaveRef != "" ||
		reentered.WaitCohortRef != "" ||
		reentered.WaitParentTaskRef != "" ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, agentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, agentRef)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reentered: %v", err)
	}
	closureStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ClosureReason != "" ||
		len(state.BlockerRefs) != 0 ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, agentRef) {
		t.Fatalf("state=%+v", state)
	}
	if closureStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		closureStep.Reason != "operational-closure-issues-replan-recorded" ||
		!serviceStringInSetV0(closureStep.BlockerRefs, "required_test_evidence_refs") ||
		len(closureStep.ReplanDecisionRefs) != 1 {
		t.Fatalf("closureStep=%+v", closureStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "operational-closure-issues-replan-followup-agents-waiting" ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, agentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}

	replayed, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T18:30:02Z",
		CorrelationID:              "corr-service-operational-closure-late-followup",
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(updatedRun),
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 replay: %v", err)
	}
	if !serviceStringInSetV0(replayed.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(replayed.WaitAgentRefs, agentRef) {
		t.Fatalf("replayed=%+v followup=%s old=%s", replayed, followupAgentRef, agentRef)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 replay: %v", err)
	}
	if state.ReplanAttempts != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-issues-replan-v0") != 1 {
		t.Fatalf("state replay=%+v", state)
	}
}

func TestMaybeCloseOperationalDirectorV0ReplayNoDuplicaBloqueoDeCierre(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-issues-replay"
	planRef := "plan-ref-service-operational-closure-plan-state-issues-replay"
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
			ValidationRef:            "validation-ref-service-operational-closure-plan-state-issues-replay",
			ClosureRef:               "closure-ref-service-operational-closure-plan-state-issues-replay",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T10:20:00Z",
		OperationalDirectorPlanRef: planRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
		OperationalClosureSource:   source,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    run,
	}

	if _, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		loop,
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	); err != nil || len(issues) == 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first err=%v issues=%+v", err, issues)
	}
	request.OccurredAt = "2026-05-22T10:20:01Z"
	if _, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		loop,
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	); err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 replay err=%v issues=%+v", err, issues)
	}
	if !source.Called {
		t.Fatalf("closure source no invocado en primer intento")
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-issues" ||
		serviceCountStringV0(state.BlockerRefs, "operational-closure-issues") != 1 ||
		serviceCountStringV0(state.BlockerRefs, "required_test_evidence_store") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		serviceCountStringV0(step.BlockerRefs, "required_test_evidence_store") != 1 {
		t.Fatalf("plan state bloqueado no idempotente: state=%+v step=%+v", state, step)
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

func TestMaybeCloseOperationalDirectorV0BloqueaPlanStateSinClosureSourceEnReplanOrClose(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-no-source"
	planRef := "plan-ref-service-operational-closure-plan-state-no-source"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T11:00:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
			DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(runRef)),
			RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)),
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
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar sin source de cierre: %+v", loop.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-source-unavailable" ||
		serviceCountStringV0(state.BlockerRefs, "operational-closure-source-unavailable") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-source-unavailable" ||
		serviceCountStringV0(step.BlockerRefs, "operational-closure-source-unavailable") != 1 {
		t.Fatalf("plan state no bloqueado por falta de source: state=%+v step=%+v", state, step)
	}
}

func TestMaybeCloseOperationalDirectorV0BloqueaPlanStateSinTaskStoreEnReplanOrClose(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-no-task-store"
	planRef := "plan-ref-service-operational-closure-plan-state-no-task-store"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:            "task-ref-service-operational-closure-001",
			DeliveryRef:       "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
			ValidationRef:     "validation-ref-service-operational-closure-plan-state-no-task-store",
			ClosureRef:        "closure-ref-service-operational-closure-plan-state-no-task-store",
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T11:05:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
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
		t.Fatalf("closure source no debe invocarse sin task store")
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar sin task store: %+v", loop.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-task-store-unavailable" ||
		serviceCountStringV0(state.BlockerRefs, "operational-closure-task-store-unavailable") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-task-store-unavailable" ||
		serviceCountStringV0(step.BlockerRefs, "operational-closure-task-store-unavailable") != 1 {
		t.Fatalf("plan state no bloqueado por falta de task store: state=%+v step=%+v", state, step)
	}
}

func TestMaybeCloseOperationalDirectorV0BloqueaPlanStateConOutboxPendienteEnReplanOrClose(t *testing.T) {
	runRef := "run-service-operational-closure-plan-state-outbox-pending"
	planRef := "plan-ref-service-operational-closure-plan-state-outbox-pending"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef),
	)
	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:            "task-ref-service-operational-closure-001",
			DeliveryRef:       "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef: "accepted-review-ref-service-operational-closure-001",
			ValidationRef:     "validation-ref-service-operational-closure-plan-state-outbox-pending",
			ClosureRef:        "closure-ref-service-operational-closure-plan-state-outbox-pending",
		},
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     runRef,
			OccurredAt:                 "2026-05-22T19:00:00Z",
			OperationalDirectorPlanRef: planRef,
		},
		StartAppDirectorPortsV0{
			RunStore:                   orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			OperationalClosureSource:   source,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status:             orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			PendingOutboxCount: 2,
			Run:                run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
	}
	if source.Called {
		t.Fatalf("closure source no debe invocarse con outbox pendiente")
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar con outbox pendiente: %+v", loop.Run)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != "operational-closure-outbox-pending" ||
		serviceCountStringV0(state.BlockerRefs, "operational-closure-outbox-pending") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-blocked-v0") != 1 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != "operational-closure-outbox-pending" ||
		serviceCountStringV0(step.BlockerRefs, "operational-closure-outbox-pending") != 1 {
		t.Fatalf("plan state no bloqueado por outbox pendiente: state=%+v step=%+v", state, step)
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

func TestMaybeCloseOperationalDirectorV0NoCierraConPlanStateFueraDeReplanOrCloseRunning(t *testing.T) {
	type closureBlockedStep struct {
		Name   string
		Kind   orquestadirectoroperativo.OperationalDirectorStepKindV0
		Status orquestadirectoroperativo.OperationalDirectorStepStatusV0
	}
	for _, tc := range []closureBlockedStep{
		{Name: "launch-running", Kind: orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0, Status: orquestadirectoroperativo.OperationalDirectorStepRunningV0},
		{Name: "gather-context-running", Kind: orquestadirectoroperativo.OperationalDirectorStepGatherContextV0, Status: orquestadirectoroperativo.OperationalDirectorStepRunningV0},
		{Name: "split-work-running", Kind: orquestadirectoroperativo.OperationalDirectorStepSplitWorkV0, Status: orquestadirectoroperativo.OperationalDirectorStepRunningV0},
		{Name: "govern-delegation-running", Kind: orquestadirectoroperativo.OperationalDirectorStepGovernDelegationV0, Status: orquestadirectoroperativo.OperationalDirectorStepRunningV0},
		{Name: "replan-or-close-pending", Kind: orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0, Status: orquestadirectoroperativo.OperationalDirectorStepPendingV0},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			runRef := "run-service-operational-closure-plan-state-" + tc.Name
			planRef := "plan-ref-service-operational-closure-plan-state-" + tc.Name
			run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
			run.Tasks = []string{"task-ref-service-operational-closure-001"}
			run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
			run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
			source := &serviceOperationalClosureSourceForTestV0{
				Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
					TaskID:                   "task-ref-service-operational-closure-001",
					DeliveryRef:              "delivery-ref-service-operational-closure-001",
					AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
					ValidationRef:            "validation-ref-service-operational-closure-plan-state-" + tc.Name,
					ClosureRef:               "closure-ref-service-operational-closure-plan-state-" + tc.Name,
					RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
					EvidenceRefs:             []string{"evidence-ref-service-operational-closure-001"},
				},
			}
			state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
			state.ActiveStepID = "step-active-before-replan"
			state.Steps = []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
				StepID:    "step-active-before-replan",
				Kind:      tc.Kind,
				Status:    tc.Status,
				TaskRefs:  []string{"task-ref-service-operational-closure-001"},
				AgentRefs: []string{orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")},
			}}
			planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
			loop, issues, err := maybeCloseOperationalDirectorV0(
				context.Background(),
				ContinueAppDirectorRequestV0{
					RunRef:                     runRef,
					OccurredAt:                 "2026-05-22T11:10:00Z",
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
				t.Fatalf("closure source no debe invocarse con active step %s/%s", tc.Kind, tc.Status)
			}
			if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
				t.Fatalf("run cerrado con active step %s/%s: %+v", tc.Kind, tc.Status, loop.Run)
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

type serviceOperationalClosureOpenTaskSourceForTestV0 struct {
	Requests  map[string]orquestacionnucleoapp.OperationalDirectorClosureRequestV0
	LastInput AppDirectorOperationalClosureRequestV0
	TaskCalls []string
}

func (source *serviceOperationalClosureOpenTaskSourceForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.LastInput = request
	for _, taskRef := range request.Run.Tasks {
		if serviceStringInSetV0(request.Run.ClosedTasks, taskRef) {
			continue
		}
		closureRequest, ok := source.Requests[taskRef]
		if !ok {
			return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
		}
		source.TaskCalls = append(source.TaskCalls, taskRef)
		return closureRequest, true, nil
	}
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, nil
}

type serviceOperationalClosureTaskRefsForTestV0 struct {
	TaskRef         string
	DeliveryRef     string
	ReviewRequestID string
	ReviewResultRef string
	AcceptedRef     string
	TestEvidenceRef string
	ValidationRef   string
	ClosureRef      string
}

func serviceOperationalClosureRequestForTaskRefsV0(
	refs serviceOperationalClosureTaskRefsForTestV0,
) orquestacionnucleoapp.OperationalDirectorClosureRequestV0 {
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
		TaskID:                   refs.TaskRef,
		DeliveryRef:              refs.DeliveryRef,
		AcceptedReviewRef:        refs.AcceptedRef,
		ValidationRef:            refs.ValidationRef,
		ClosureRef:               refs.ClosureRef,
		RequiredTestEvidenceRefs: []string{refs.TestEvidenceRef},
		EvidenceRefs:             []string{refs.DeliveryRef, refs.TestEvidenceRef},
	}
}

func serviceOperationalClosureReviewResultProjectionForTestV0(
	refs serviceOperationalClosureTaskRefsForTestV0,
) string {
	return refs.ReviewResultRef +
		"#review_result:" + string(orquestacoreworkflow.ReviewResultStatusAcceptedV0) +
		"#review_request:" + refs.ReviewRequestID +
		"#delivery:" + refs.DeliveryRef
}

func serviceCountEventsByTypeV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func serviceCountStringV0(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}

func serviceFirstAgentFollowupFromReplanProjectionForTestV0(t *testing.T, projection string) string {
	t.Helper()
	for _, part := range strings.Split(projection, "#") {
		if !strings.HasPrefix(part, "followups:") {
			continue
		}
		for _, ref := range strings.Split(strings.TrimPrefix(part, "followups:"), "+") {
			if strings.HasPrefix(ref, "agent-ref-") {
				return ref
			}
		}
	}
	t.Fatalf("proyeccion sin followup agent: %s", projection)
	return ""
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

func serviceOperationalClosurePlanStateReadyForCloseTasksV0(
	runRef string,
	planRef string,
	parentTaskRef string,
	tasks ...serviceOperationalClosureTaskRefsForTestV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	state.ActiveParentTaskRef = parentTaskRef
	taskRefs := make([]string, 0, len(tasks))
	agentRefs := make([]string, 0, len(tasks))
	deliveryRefs := make([]string, 0, len(tasks))
	reviewResultRefs := make([]string, 0, len(tasks))
	acceptedRefs := make([]string, 0, len(tasks))
	testEvidenceRefs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskRefs = append(taskRefs, task.TaskRef)
		agentRefs = append(agentRefs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskRef))
		deliveryRefs = append(deliveryRefs, task.DeliveryRef)
		reviewResultRefs = append(reviewResultRefs, task.ReviewResultRef)
		acceptedRefs = append(acceptedRefs, task.AcceptedRef)
		testEvidenceRefs = append(testEvidenceRefs, task.TestEvidenceRef)
	}
	state.Steps = []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		{
			StepID:             "step-review-deliveries",
			Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
			Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			TaskRefs:           taskRefs,
			DeliveryRefs:       deliveryRefs,
			ReviewResultRefs:   reviewResultRefs,
			AcceptedReviewRefs: acceptedRefs,
		},
		{
			StepID:                   "step-run-required-tests",
			Kind:                     orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
			Status:                   orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			TaskRefs:                 taskRefs,
			RequiredTestEvidenceRefs: testEvidenceRefs,
		},
		{
			StepID:                   "step-replan-or-close",
			Kind:                     orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
			Status:                   orquestadirectoroperativo.OperationalDirectorStepRunningV0,
			WaveRef:                  "wave-service-operational-closure-plan-state",
			CohortRef:                "cohort-service-operational-closure-plan-state",
			ParentTaskRef:            parentTaskRef,
			TaskRefs:                 taskRefs,
			AgentRefs:                agentRefs,
			DeliveryRefs:             deliveryRefs,
			ReviewResultRefs:         reviewResultRefs,
			RequiredTestEvidenceRefs: testEvidenceRefs,
		},
	}
	return state
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

func serviceOperationalClosureEventsForTasksForTestV0(
	t *testing.T,
	runRef string,
	tasks ...serviceOperationalClosureTaskRefsForTestV0,
) []orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	events := make([]orquestacoreworkflow.OrchestrationEventV0, 0, len(tasks)*4)
	sequence := int64(1)
	for _, task := range tasks {
		events = append(events,
			serviceOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
				DeliveryRef:  task.DeliveryRef,
				PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				TaskID:       task.TaskRef,
				AgentRef:     orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskRef),
				Summary:      "Entrega para cierre operativo multitarea.",
				EvidenceRefs: []string{task.DeliveryRef},
			}),
		)
		sequence++
		events = append(events,
			serviceOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
				ReviewRequestID: task.ReviewRequestID,
				PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				DeliveryRef:     task.DeliveryRef,
				Summary:         "Review para cierre operativo multitarea.",
			}),
		)
		sequence++
		events = append(events,
			serviceOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
				ReviewResultRef: task.ReviewResultRef,
				ReviewRequestID: task.ReviewRequestID,
				DeliveryRef:     task.DeliveryRef,
				Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
				Summary:         "Review aceptada.",
				EvidenceRefs:    []string{task.TestEvidenceRef},
			}),
		)
		sequence++
		events = append(events,
			serviceOperationalClosureEventForTestV0(t, runRef, sequence, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
				AcceptedReviewRef: task.AcceptedRef,
				PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
				ReviewRequestID:   task.ReviewRequestID,
				DeliveryRef:       task.DeliveryRef,
				Summary:           "Review aceptada.",
			}),
		)
		sequence++
	}
	return events
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
		EventID:        "evt-service-operational-closure-" + eventType + "-" + strconv.FormatInt(sequence, 10),
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

func serviceOperationalClosureTaskWithRefsForTestV0(
	runRef string,
	parentTaskRef string,
	refs serviceOperationalClosureTaskRefsForTestV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             refs.TaskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:              "Task cierre operativo multitarea",
		WriteSet:           []string{"internal/module"},
		AcceptanceCriteria: []string{"entrega revisada"},
		RequiredTests:      []string{"go test ./..."},
		ParentTaskRef:      parentTaskRef,
		CohortRef:          "cohort-service-operational-closure-plan-state",
		WaveRef:            "wave-service-operational-closure-plan-state",
		DelegationDepth:    1,
		MaxChildAgents:     6,
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

func serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(
	runRef string,
	refs serviceOperationalClosureTaskRefsForTestV0,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       refs.TestEvidenceRef,
		RunRef:            runRef,
		TaskRef:           refs.TaskRef,
		TestCommand:       "go test ./...",
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       refs.DeliveryRef,
		ReviewRequestID:   refs.ReviewRequestID,
		ReviewResultRef:   refs.ReviewResultRef,
		AcceptedReviewRef: refs.AcceptedRef,
		OccurredAt:        "2026-05-22T12:00:00Z",
		EvidenceRefs:      []string{refs.DeliveryRef, refs.TestEvidenceRef},
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
