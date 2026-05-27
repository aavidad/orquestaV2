package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueAppDirectorV0DecisionPlanStateEjecutaRunnerYCierra(t *testing.T) {
	ctx := context.Background()
	runRef := "run-app-director-decision-plan-state-close-001"
	planRef := defaultOperationalDirectorDecisionPlanRefV0(runRef)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()

	request := validStartAppDirectorRequestForTestV0()
	request.RunRef = runRef
	request.MaxDecisionCycles = 2
	request.MaxBursts = 8

	started, err := StartAppDirectorV0(ctx, request, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorDecisionSource:     &servicePersistentDecisionSourceForTestV0{},
		DirectorTaskStore:          taskStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateWriter: planStateStore,
		OperationalPlanStateStore:  planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if len(started.Run.Tasks) != 1 {
		t.Fatalf("tasks=%v", started.Run.Tasks)
	}
	taskRef := started.Run.Tasks[0]
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	tasks, err := taskStore.LoadWorkflowTasksV0(ctx, runRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(tasks) != 1 || len(tasks[0].RequiredTests) != 1 {
		t.Fatalf("workflow task requerida invalida: %+v", tasks)
	}
	requiredTest := tasks[0].RequiredTests[0]

	deliveryRef := "delivery-ref-director-decision-close-001"
	reviewRequestID := "review-request-id-director-decision-close-001"
	reviewResultRef := "review-result-ref-director-decision-close-001"
	acceptedReviewRef := "accepted-review-ref-director-decision-close-001"
	serviceApplyDirectorDecisionCommandForTestV0(t, store, sink, serviceMustDirectorDecisionCommandForTestV0(t,
		func() (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewRegisterDeliveryCommandV0(
				serviceDirectorDecisionCommandMetaForTestV0(runRef, "register-delivery-close"),
				orquestacoreworkflow.RegisterDeliveryCommandPayloadV0{
					DeliveryRef:  deliveryRef,
					PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
					TaskID:       taskRef,
					AgentRef:     agentRef,
					Summary:      "Entrega de la microtarea creada por decision persistida del director.",
					EvidenceRefs: []string{"evidence-ref-director-decision-delivery-close-001"},
				},
			)
		},
	))
	serviceApplyDirectorDecisionCommandForTestV0(t, store, sink, serviceMustDirectorDecisionCommandForTestV0(t,
		func() (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewOpenPhaseCommandV0(
				serviceDirectorDecisionCommandMetaForTestV0(runRef, "open-revision-close"),
				orquestacoreworkflow.OpenPhaseCommandPayloadV0{
					PhaseID: string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
					Reason:  "Revisar entrega creada por la microtarea del director.",
				},
			)
		},
	))
	serviceApplyDirectorDecisionCommandForTestV0(t, store, sink, serviceMustDirectorDecisionCommandForTestV0(t,
		func() (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewRequestReviewCommandV0(
				serviceDirectorDecisionCommandMetaForTestV0(runRef, "request-review-close"),
				orquestacoreworkflow.RequestReviewCommandPayloadV0{
					ReviewRequestID: reviewRequestID,
					PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
					DeliveryRef:     deliveryRef,
					Summary:         "Solicitar revision causal de la entrega.",
					EvidenceRefs:    []string{"evidence-ref-director-decision-review-request-close-001"},
				},
			)
		},
	))
	serviceApplyDirectorDecisionCommandForTestV0(t, store, sink, serviceMustDirectorDecisionCommandForTestV0(t,
		func() (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewRecordReviewResultCommandV0(
				serviceDirectorDecisionCommandMetaForTestV0(runRef, "record-review-close"),
				orquestacoreworkflow.ReviewResultV0{
					ReviewResultRef: reviewResultRef,
					ReviewRequestID: reviewRequestID,
					DeliveryRef:     deliveryRef,
					Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
					Summary:         "Review aceptada para ejecutar tests requeridos.",
					EvidenceRefs:    []string{"evidence-ref-director-decision-review-result-close-001"},
				},
			)
		},
	))
	serviceApplyDirectorDecisionCommandForTestV0(t, store, sink, serviceMustDirectorDecisionCommandForTestV0(t,
		func() (orquestacoreworkflow.OrchestrationCommandV0, error) {
			return orquestacoreworkflow.NewAcceptReviewCommandV0(
				serviceDirectorDecisionCommandMetaForTestV0(runRef, "accept-review-close"),
				orquestacoreworkflow.AcceptReviewCommandPayloadV0{
					AcceptedReviewRef: acceptedReviewRef,
					PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
					ReviewRequestID:   reviewRequestID,
					DeliveryRef:       deliveryRef,
					Summary:           "Aceptar review para cierre operativo.",
					EvidenceRefs:      []string{"evidence-ref-director-decision-review-accepted-close-001"},
				},
			)
		},
	))

	executor := &fakeServiceRequiredTestCommandExecutorV0{
		results: map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			requiredTest: {
				Status:       orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-director-decision-required-test-close-001"},
			},
		},
	}
	closureSource := &serviceOperationalDirectorClosureSourceFromRequestForTestV0{
		Fixture: serviceOperationalDirectorPlanStateReviewFixtureV0{
			TaskRef:           taskRef,
			DeliveryRef:       deliveryRef,
			AcceptedReviewRef: acceptedReviewRef,
		},
	}

	continued, err := ContinueAppDirectorV0(ctx, ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T13:00:00Z",
		CorrelationID:              "corr-app-director-decision-plan-state-close-001",
		OperationalDirectorPlanRef: planRef,
		MaxBursts:                  2,
		MaxStepsPerBurst:           2,
		MaxDispatchesPerWait:       2,
		MaxCommands:                20,
		MaxOutboxPerCycle:          4,
	}, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		EventReader:                sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateWriter: planStateStore,
		OperationalPlanStateStore:  planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
		RequiredTestRunner:         orquestacionnucleoapp.RequiredTestRunnerV0{Executor: executor, EvidenceWriter: testEvidenceStore},
		OperationalClosureSource:   closureSource,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if len(continued.OperationalClosureIssues) != 0 ||
		continued.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		continued.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("continue no cerro ciclo: result=%+v", continued)
	}
	if len(executor.commands) != 1 || executor.commands[0] != requiredTest {
		t.Fatalf("runner commands=%+v required=%s", executor.commands, requiredTest)
	}
	if !closureSource.Called || len(closureSource.LastRequest.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("closure source sin evidencia requerida: called=%v request=%+v", closureSource.Called, closureSource.LastRequest)
	}
	evidence, err := testEvidenceStore.LoadRequiredTestEvidenceV0(ctx, runRef, closureSource.LastRequest.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TaskRef != taskRef ||
		evidence[0].DeliveryRef != deliveryRef ||
		evidence[0].AcceptedReviewRef != acceptedReviewRef {
		t.Fatalf("evidence generada invalida: %+v", evidence)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(ctx, runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepClosedV0 ||
		!serviceStringInSetV0(continued.Run.ClosedTasks, taskRef) {
		t.Fatalf("state/run no cerrado: state=%+v replanStep=%+v run=%+v", state, replanStep, continued.Run)
	}
}

func TestStartAppDirectorV0NoSaltaDecisionPendienteDelDirector(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	request := validStartAppDirectorRequestForTestV0()
	request.MaxDecisionCycles = 3
	request.MaxBursts = 8

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:               store,
			EventSink:              sink,
			OutboxLedger:           ledger,
			DeliverySource:         serviceDirectorArtifactSourceForTestV0{},
			DirectorDecisionSource: servicePendingDecisionSourceForTestV0{},
			DirectorTaskStore:      taskStore,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0 {
		t.Fatalf("current_phase=%s, want votacion_y_decision", result.Run.CurrentPhase)
	}
	if len(result.Run.Tasks) != 0 {
		t.Fatalf("tasks=%v, want empty", result.Run.Tasks)
	}
}
