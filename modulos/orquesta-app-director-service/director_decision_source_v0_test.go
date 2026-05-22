package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestStartAppDirectorV0ConsumesDirectorDecisionSource(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := serviceDirectorDecisionSourceForTestV0{
		Decisions: []orquestadirectoragent.DirectorAgentDecisionV0{
			serviceOpenVoteDirectorDecisionForTestV0("run-app-director-service-001"),
		},
	}

	result, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:               store,
			EventSink:              sink,
			OutboxLedger:           ledger,
			DirectorDecisionSource: source,
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
		t.Fatalf("current_phase=%s", result.Run.CurrentPhase)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventPhaseOpenedV0) {
		t.Fatalf("sink sin PhaseOpened: %+v", sink.EventsV0())
	}
}

func TestStartAppDirectorV0IgnoresPersistedDirectorDecisionsAlreadyApplied(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := &servicePersistentDecisionSourceForTestV0{}

	request := validStartAppDirectorRequestForTestV0()
	request.MaxDecisionCycles = 2
	request.MaxBursts = 8

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			DeliverySource: serviceDirectorArtifactSourceForTestV0{
				AgentRef: "agent-agenda-director",
			},
			DirectorDecisionSource: source,
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
	if source.calls < 2 {
		t.Fatalf("decision_source_calls=%d", source.calls)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s", result.Run.CurrentPhase)
	}
	if len(result.Run.Tasks) != 1 || result.Run.Tasks[0] != "task-ref-agenda-autonomy-001" {
		t.Fatalf("tasks=%v", result.Run.Tasks)
	}
}

func TestStartAppDirectorV0DecisionCreateMicrotaskRequiredTestsCreaPlanStateReentrable(t *testing.T) {
	runRef := "run-app-director-decision-plan-state-001"
	planRef := "operational-director-plan-decision-plan-state-001"
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	source := &servicePersistentDecisionSourceForTestV0{}

	request := validStartAppDirectorRequestForTestV0()
	request.RunRef = runRef
	request.OperationalDirectorPlanRef = planRef
	request.MaxDecisionCycles = 2
	request.MaxBursts = 8

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:                   store,
			EventSink:                  sink,
			OutboxLedger:               ledger,
			DeliverySource:             serviceDirectorArtifactSourceForTestV0{},
			DirectorDecisionSource:     source,
			DirectorTaskStore:          taskStore,
			WaitStateWriter:            waitStore,
			OperationalPlanStateWriter: planStateStore,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if len(result.Run.Tasks) != 1 {
		t.Fatalf("tasks=%v", result.Run.Tasks)
	}
	taskRef := result.Run.Tasks[0]
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	tasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(tasks) != 1 || len(tasks[0].RequiredTests) == 0 {
		t.Fatalf("workflow task sin required_tests: %+v", tasks)
	}

	planState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, planState, "step-wait-subagents")
	if planState.ActiveStepID != "step-wait-subagents" ||
		!serviceStringInSetV0(waitStep.TaskRefs, taskRef) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, agentRef) ||
		!serviceStringInSetV0(planState.RequiredTestRefs, tasks[0].RequiredTests[0]) {
		t.Fatalf("planState=%+v waitStep=%+v task=%+v", planState, waitStep, tasks[0])
	}

	second, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-21T12:00:00Z",
		CorrelationID:              "corr-app-director-decision-plan-state-reentry",
		MaxBursts:                  2,
		MaxStepsPerBurst:           2,
		MaxDispatchesPerWait:       2,
		MaxCommands:                10,
		MaxOutboxPerCycle:          4,
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{
		RunStore:                  store,
		EventSink:                 sink,
		OutboxLedger:              ledger,
		DirectorTaskStore:         taskStore,
		WaitStateWriter:           waitStore,
		OperationalPlanStateStore: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if second.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("loop_status=%s result=%+v", second.LoopStatus, second)
	}
	waitRef := appDirectorWaitRefV0(
		runRef,
		appDirectorWaitFilterV0{WaveRef: planState.ActiveWaveRef, CohortRef: planState.ActiveCohortRef},
		"corr-app-director-decision-plan-state-reentry",
	)
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if !serviceStringInSetV0(waitState.AgentRefs, agentRef) ||
		!serviceStringInSetV0(waitState.PendingAgentRefs, agentRef) {
		t.Fatalf("waitState=%+v agentRef=%s", waitState, agentRef)
	}
}

func TestStartAppDirectorV0DecisionPlanStateDefaultRefEsIdempotenteEnReentrada(t *testing.T) {
	runRef := "run-app-director-decision-plan-state-default-ref-001"
	planRef := defaultOperationalDirectorDecisionPlanRefV0(runRef)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	source := &servicePersistentDecisionSourceForTestV0{}

	request := validStartAppDirectorRequestForTestV0()
	request.RunRef = runRef
	request.MaxDecisionCycles = 2
	request.MaxBursts = 8

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:                   store,
			EventSink:                  sink,
			OutboxLedger:               ledger,
			DeliverySource:             serviceDirectorArtifactSourceForTestV0{},
			DirectorDecisionSource:     source,
			DirectorTaskStore:          taskStore,
			OperationalPlanStateWriter: planStateStore,
			OperationalPlanStateStore:  planStateStore,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if len(result.Run.Tasks) != 1 {
		t.Fatalf("tasks=%v", result.Run.Tasks)
	}
	first, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 first: %v", err)
	}

	reentered, err := ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        runRef,
			OccurredAt:    "2026-05-21T13:00:00Z",
			CorrelationID: "corr-app-director-decision-plan-state-default-ref-reentry",
		},
		StartAppDirectorPortsV0{
			RunStore:                   store,
			DirectorTaskStore:          taskStore,
			OperationalPlanStateStore:  planStateStore,
			OperationalPlanStateWriter: planStateStore,
		},
	)
	if err != nil {
		t.Fatalf("ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0: %v", err)
	}
	if reentered.OperationalDirectorPlanRef != planRef {
		t.Fatalf("operational_director_plan_ref=%q, want %q", reentered.OperationalDirectorPlanRef, planRef)
	}
	second, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 second: %v", err)
	}
	if second.ObservedAt != first.ObservedAt ||
		second.CorrelationID != first.CorrelationID ||
		second.ActiveStepID != first.ActiveStepID ||
		len(second.PendingAgentRefs) != len(first.PendingAgentRefs) {
		t.Fatalf("state reescrito en reentrada: first=%+v second=%+v", first, second)
	}
}

func TestStartAppDirectorV0DecisionPlanStateReentraTrasReloadStateFile(t *testing.T) {
	ctx := context.Background()
	runRef := "run-app-director-decision-plan-state-file-001"
	planRef := defaultOperationalDirectorDecisionPlanRefV0(runRef)
	rootDir := t.TempDir()
	store := mustServiceStateFileStoreForTestV0(t, rootDir)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	source := &servicePersistentDecisionSourceForTestV0{}

	request := validStartAppDirectorRequestForTestV0()
	request.RunRef = runRef
	request.MaxDecisionCycles = 2
	request.MaxBursts = 8

	result, err := StartAppDirectorV0(ctx, request, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		OutboxLedger:               ledger,
		DeliverySource:             serviceDirectorArtifactSourceForTestV0{},
		DirectorDecisionSource:     source,
		DirectorTaskStore:          store,
		WaitStateWriter:            store,
		OperationalPlanStateWriter: store,
		OperationalPlanStateStore:  store,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherWithPortsForTestV0(store, store, ledger),
			serviceAgentLauncherDispatcherWithPortsForTestV0(store, store, ledger),
		},
	})
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if len(result.Run.Tasks) != 1 {
		t.Fatalf("tasks=%v", result.Run.Tasks)
	}
	taskRef := result.Run.Tasks[0]
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)

	recovered := mustServiceStateFileStoreForTestV0(t, rootDir)
	planState, err := recovered.LoadOperationalDirectorPlanStateV0(ctx, runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 recovered: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, planState, "step-wait-subagents")
	if planState.ActiveStepID != "step-wait-subagents" ||
		!serviceStringInSetV0(waitStep.TaskRefs, taskRef) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, agentRef) {
		t.Fatalf("planState=%+v waitStep=%+v", planState, waitStep)
	}

	continueLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	continued, err := ContinueAppDirectorV0(ctx, ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-21T13:10:00Z",
		CorrelationID:              "corr-app-director-decision-plan-state-file-reentry",
		MaxBursts:                  2,
		MaxStepsPerBurst:           2,
		MaxDispatchesPerWait:       2,
		MaxCommands:                10,
		MaxOutboxPerCycle:          4,
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{
		RunStore:                  recovered,
		EventSink:                 recovered,
		OutboxLedger:              continueLedger,
		DirectorTaskStore:         recovered,
		WaitStateWriter:           recovered,
		OperationalPlanStateStore: recovered,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherWithPortsForTestV0(recovered, recovered, continueLedger),
			serviceAgentLauncherDispatcherWithPortsForTestV0(recovered, recovered, continueLedger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if continued.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("loop_status=%s result=%+v", continued.LoopStatus, continued)
	}
	waitRef := appDirectorWaitRefV0(
		runRef,
		appDirectorWaitFilterV0{WaveRef: planState.ActiveWaveRef, CohortRef: planState.ActiveCohortRef},
		"corr-app-director-decision-plan-state-file-reentry",
	)
	waitState, err := recovered.LoadWorkflowTaskWaitStateV0(ctx, runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 recovered: %v", err)
	}
	if !serviceStringInSetV0(waitState.AgentRefs, agentRef) ||
		!serviceStringInSetV0(waitState.PendingAgentRefs, agentRef) {
		t.Fatalf("waitState=%+v agentRef=%s", waitState, agentRef)
	}
}

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

func TestStartAppDirectorV0BloqueaDecisionInvalidaDelSource(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	decision := serviceMicrotaskDecisionForTestV0("run-app-director-service-001")
	decision.CreateMicrotask.Task.Summary = ""

	result, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			DirectorDecisionSource: serviceDirectorDecisionSourceForTestV0{
				Decisions: []orquestadirectoragent.DirectorAgentDecisionV0{decision},
			},
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	requireServiceDecisionRecoveryBlockV0(
		t,
		result.Run,
		sink.EventsV0(),
		"director_decision.decision.create_microtask.task.summary",
	)
	got, loadErr := store.LoadRunV0(context.Background(), result.Run.RunID)
	if loadErr != nil {
		t.Fatalf("LoadRunV0: %v", loadErr)
	}
	if got.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("stored status=%s", got.Status)
	}
}

func TestStartAppDirectorV0BloqueaErrorDelDecisionSourceSinReintentar(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	source := &serviceFailingDecisionSourceForTestV0{
		err: errors.New("decision source unavailable"),
	}

	result, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:               store,
			EventSink:              sink,
			OutboxLedger:           ledger,
			DirectorDecisionSource: source,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if source.calls != 1 {
		t.Fatalf("decision_source_calls=%d, want 1", source.calls)
	}
	requireServiceDecisionRecoveryBlockV0(
		t,
		result.Run,
		sink.EventsV0(),
		"director_decision_source",
	)
}

func mustServiceStateFileStoreForTestV0(
	t *testing.T,
	rootDir string,
) *orquestastatefile.StoreV0 {
	t.Helper()
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: rootDir})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	return store
}

func serviceCapacityDispatcherWithPortsForTestV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityHighV0,
			OccurredAt:      "2026-05-21T13:10:01Z",
			CorrelationID:   "corr-app-director-service-file-capacity-001",
			RequestedBy:     "orquesta-app-director-service-test",
		},
		Acker: ledger,
	}
}

func serviceAgentLauncherDispatcherWithPortsForTestV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-21T13:10:02Z",
			CorrelationID: "corr-app-director-service-file-agent-001",
			RequestedBy:   "orquesta-app-director-service-test",
		},
		Acker: ledger,
	}
}

type serviceDirectorDecisionSourceForTestV0 struct {
	Decisions []orquestadirectoragent.DirectorAgentDecisionV0
}

func (source serviceDirectorDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), source.Decisions...), nil
}

type servicePersistentDecisionSourceForTestV0 struct {
	calls int
}

func (source *servicePersistentDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source.calls++
	return serviceDirectorPlanDecisionsForTestV0(request.Run), nil
}

type servicePendingDecisionSourceForTestV0 struct{}

func (source servicePendingDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decisions := serviceDirectorPlanDecisionsForTestV0(request.Run)
	for i := range decisions {
		if decisions[i].AcceptDecision != nil {
			decisions[i].AcceptDecision.VoteRef = "vote-ref-pending-mismatch-001"
		}
	}
	return decisions, nil
}

type serviceFailingDecisionSourceForTestV0 struct {
	calls int
	err   error
}

func (source *serviceFailingDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source.calls++
	return nil, source.err
}

func serviceDirectorDecisionCommandMetaForTestV0(
	runRef string,
	action string,
) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "command-ref-director-decision-" + action,
		RunID:          runRef,
		IdempotencyKey: "idem-director-decision-" + action,
		CorrelationID:  "corr-director-decision-" + action,
		RequestedBy:    "orquesta-app-director-service-test",
		OccurredAt:     "2026-05-22T12:59:00Z",
	}
}

func serviceMustDirectorDecisionCommandForTestV0(
	t *testing.T,
	build func() (orquestacoreworkflow.OrchestrationCommandV0, error),
) orquestacoreworkflow.OrchestrationCommandV0 {
	t.Helper()
	command, err := build()
	if err != nil {
		t.Fatalf("build command: %v", err)
	}
	return command
}

func serviceApplyDirectorDecisionCommandForTestV0(
	t *testing.T,
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) {
	t.Helper()
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(context.Background(), store, sink, command); err != nil {
		t.Fatalf("HandleStoredWorkflowCommandV0(%s): %v", command.CommandType, err)
	}
}

func serviceOpenVoteDirectorDecisionForTestV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-service-open-vote-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-service-open-vote-001",
		Summary:       "Abrir fase de decision.",
		EvidenceRefs:  []string{"evidence-ref-service-open-vote-001"},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			Reason:  "Preparar votacion del director.",
		},
	}
}

func requireServiceDecisionRecoveryBlockV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
	field string,
) {
	t.Helper()
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("status=%s, want bloqueada", run.Status)
	}
	if len(run.Blockers) != 1 || !strings.HasPrefix(run.Blockers[0], "app-director-decision-") {
		t.Fatalf("blockers=%v", run.Blockers)
	}
	for _, event := range events {
		if event.EventType != orquestacoreworkflow.OrchestrationEventRunBlockedV0 {
			continue
		}
		var payload orquestacoreworkflow.RunBlockedPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("RunBlocked payload: %v", err)
		}
		if strings.Contains(payload.Summary, "field="+field) {
			return
		}
	}
	t.Fatalf("sin RunBlocked con field=%s: %+v", field, events)
}
