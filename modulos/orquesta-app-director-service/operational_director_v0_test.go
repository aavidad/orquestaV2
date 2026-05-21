package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestContinueAppDirectorV0MaterializaOperationalDirectorPlanYEsperaOla(t *testing.T) {
	runRef := "run-app-director-operational-plan-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-17T13:45:00Z",
		CorrelationID:           "corr-app-director-operational-plan-001",
		MaxBursts:               4,
		MaxStepsPerBurst:        4,
		MaxDispatchesPerWait:    4,
		MaxCommands:             20,
		MaxOutboxPerCycle:       8,
		OperationalDirectorPlan: plan,
		OperationalDirectorFunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
	}, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("loop_status=%s result=%+v", result.LoopStatus, result)
	}
	if len(result.Run.Tasks) != 1 {
		t.Fatalf("tasks=%v", result.Run.Tasks)
	}
	taskRef := result.Run.Tasks[0]
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if !serviceStringInSetV0(result.Run.StartedAgents, agentRef) {
		t.Fatalf("started=%v want=%s", result.Run.StartedAgents, agentRef)
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}
	if len(tasks) != 1 || tasks[0].WaveRef == "" || tasks[0].CohortRef == "" {
		t.Fatalf("operational task sin metadata: %+v", tasks)
	}
	waitRef := appDirectorWaitRefV0(
		runRef,
		appDirectorWaitFilterV0{WaveRef: tasks[0].WaveRef, CohortRef: tasks[0].CohortRef},
		"corr-app-director-operational-plan-001",
	)
	state, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if !serviceStringInSetV0(state.AgentRefs, agentRef) ||
		!serviceStringInSetV0(state.PendingAgentRefs, agentRef) {
		t.Fatalf("wait state=%+v agent=%s", state, agentRef)
	}
	planState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if planState.ActiveStepID != "step-wait-subagents" ||
		planState.ActiveWaveRef != tasks[0].WaveRef ||
		!serviceStringInSetV0(planState.PendingAgentRefs, agentRef) ||
		len(planState.Steps) == 0 {
		t.Fatalf("plan state=%+v", planState)
	}
	var waitStep orquestacionnucleoapp.OperationalDirectorPlanStepStateV0
	for _, step := range planState.Steps {
		if step.StepID == planState.ActiveStepID {
			waitStep = step
			break
		}
	}
	if !serviceStringInSetV0(waitStep.WaitRefs, waitRef) {
		t.Fatalf("plan state wait refs=%+v want=%s", waitStep.WaitRefs, waitRef)
	}
}

func TestContinueAppDirectorV0ReentraDesdeOperationalDirectorPlanState(t *testing.T) {
	runRef := "run-app-director-operational-plan-reentry-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)
	dispatchers := []orquestacionnucleoapp.OutboxDispatcherBindingV0{
		serviceCapacityDispatcherForTestV0(store, sink, ledger),
		serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
	}

	first, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-17T13:55:00Z",
		CorrelationID:           "corr-app-director-operational-plan-reentry-first",
		MaxBursts:               4,
		MaxStepsPerBurst:        4,
		MaxDispatchesPerWait:    4,
		MaxCommands:             20,
		MaxOutboxPerCycle:       8,
		OperationalDirectorPlan: plan,
		OperationalDirectorFunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
	}, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers:                dispatchers,
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	if len(first.Run.Tasks) != 1 {
		t.Fatalf("first tasks=%v", first.Run.Tasks)
	}
	taskRef := first.Run.Tasks[0]
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	tasks, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskRef})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0: %v", err)
	}

	second, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-17T13:56:00Z",
		CorrelationID:              "corr-app-director-operational-plan-reentry-second",
		MaxBursts:                  2,
		MaxStepsPerBurst:           2,
		MaxDispatchesPerWait:       2,
		MaxCommands:                10,
		MaxOutboxPerCycle:          4,
		OperationalDirectorPlanRef: plan.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                  store,
		EventSink:                 sink,
		OutboxLedger:              ledger,
		DirectorTaskStore:         taskStore,
		WaitStateWriter:           waitStore,
		OperationalPlanStateStore: planStateStore,
		Dispatchers:               dispatchers,
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 second: %v", err)
	}
	if second.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("loop_status=%s result=%+v", second.LoopStatus, second)
	}
	if len(second.Run.Tasks) != 1 || second.Run.Tasks[0] != taskRef {
		t.Fatalf("second tasks=%v want=%s", second.Run.Tasks, taskRef)
	}
	waitRef := appDirectorWaitRefV0(
		runRef,
		appDirectorWaitFilterV0{WaveRef: tasks[0].WaveRef, CohortRef: tasks[0].CohortRef},
		"corr-app-director-operational-plan-reentry-second",
	)
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitRef)
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if !serviceStringInSetV0(waitState.AgentRefs, agentRef) ||
		!serviceStringInSetV0(waitState.PendingAgentRefs, agentRef) {
		t.Fatalf("wait state from plan state=%+v agent=%s", waitState, agentRef)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaAReviewTrasWaitConsumido(t *testing.T) {
	runRef := "run-app-director-operational-plan-state-review-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)

	first, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-17T14:05:00Z",
		CorrelationID:           "corr-app-director-operational-plan-state-review-first",
		MaxBursts:               4,
		MaxStepsPerBurst:        4,
		MaxDispatchesPerWait:    4,
		MaxCommands:             20,
		MaxOutboxPerCycle:       8,
		OperationalDirectorPlan: plan,
		OperationalDirectorFunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
	}, StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		OperationalPlanStateWriter: planStateStore,
		OperationalPlanStateStore:  planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	if len(first.Run.Tasks) != 1 {
		t.Fatalf("tasks=%v", first.Run.Tasks)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(first.Run.Tasks[0])
	deliveredRun := first.Run
	deliveredRun.DeliveredAgents = []string{agentRef}
	if err := store.SaveRunV0(context.Background(), deliveredRun); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-17T14:06:00Z",
		OperationalDirectorPlanRef: plan.PlanRef,
	}, StartAppDirectorPortsV0{
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    deliveredRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ActiveStepID != "step-review-deliveries" || len(state.PendingAgentRefs) != 0 {
		t.Fatalf("state=%+v", state)
	}
	var waitAccepted, reviewRunning bool
	for _, step := range state.Steps {
		switch step.StepID {
		case "step-wait-subagents":
			waitAccepted = step.Status == orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 &&
				len(step.PendingAgentRefs) == 0
		case "step-review-deliveries":
			reviewRunning = step.Status == orquestadirectoroperativo.OperationalDirectorStepRunningV0 &&
				serviceStringInSetV0(step.AgentRefs, agentRef)
		}
	}
	if !waitAccepted || !reviewRunning {
		t.Fatalf("state=%+v waitAccepted=%v reviewRunning=%v", state, waitAccepted, reviewRunning)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeReviewATestsRequeridos(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ActiveStepID != "step-run-required-tests" {
		t.Fatalf("active_step_id=%s state=%+v", state.ActiveStepID, state)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, fixture.DeliveryRef) ||
		!serviceStringInSetV0(reviewStep.ReviewResultRefs, fixture.ReviewResultRef) ||
		!serviceStringInSetV0(reviewStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	if testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(testsStep.BlockerRefs, "required-tests-pending") ||
		!serviceStringInSetV0(testsStep.TaskRefs, fixture.TaskRef) {
		t.Fatalf("testsStep=%+v", testsStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewNegativaRegistraReworkReplan(t *testing.T) {
	for _, status := range []orquestacoreworkflow.ReviewResultStatusV0{
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
		orquestacoreworkflow.ReviewResultStatusRejectedV0,
	} {
		t.Run(string(status), func(t *testing.T) {
			fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(t, status)
			planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

			if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
				RunRef:                     fixture.RunRef,
				OccurredAt:                 "2026-05-17T14:20:10Z",
				OperationalDirectorPlanRef: fixture.PlanRef,
			}, StartAppDirectorPortsV0{
				EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
				OperationalPlanStateStore:  planStateStore,
				OperationalPlanStateWriter: planStateStore,
			}, orquestacionnucleoapp.ProgressiveLoopResultV0{
				Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
				Run:    fixture.Run,
			}); err != nil {
				t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
			}
			state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
			if err != nil {
				t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
			}
			reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
			testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
			replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
			if state.ActiveStepID != "step-review-deliveries" ||
				state.ReplanAttempts != 1 ||
				reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
				!serviceStringInSetV0(reviewStep.DeliveryRefs, fixture.DeliveryRef) ||
				!serviceStringInSetV0(reviewStep.ReviewResultRefs, fixture.ReviewResultRef) ||
				!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
				!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) ||
				len(reviewStep.AcceptedReviewRefs) != 0 ||
				testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepPendingV0 ||
				replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepPendingV0 {
				t.Fatalf("state=%+v reviewStep=%+v testsStep=%+v replanStep=%+v", state, reviewStep, testsStep, replanStep)
			}
		})
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeTestsAReplanConEvidenciaPassed(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		),
	)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:30Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.ActiveStepID != "step-replan-or-close" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(replanStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0EjecutaRunnerDeTestsRequeridos(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Events[2] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: fixture.ReviewResultRef,
		ReviewRequestID: fixture.ReviewRequestID,
		DeliveryRef:     fixture.DeliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		Summary:         "Review aceptada sin evidencia de test precocinada.",
		EvidenceRefs:    []string{"evidence-ref-review-result-accepted"},
	})
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	executor := &fakeServiceRequiredTestCommandExecutorV0{
		results: map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			fixture.RequiredTest: {
				Status:       orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-service-required-test-output-001"},
			},
		},
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T11:20:00Z",
		CorrelationID:              "correlation-service-required-tests-001",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
		RequiredTestRunner: orquestacionnucleoapp.RequiredTestRunnerV0{
			Executor:       executor,
			EvidenceWriter: testEvidenceStore,
		},
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if len(executor.commands) != 1 || executor.commands[0] != fixture.RequiredTest {
		t.Fatalf("commands=%+v", executor.commands)
	}
	if state.ActiveStepID != "step-replan-or-close" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		len(testsStep.RequiredTestEvidenceRefs) != 1 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		len(replanStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}

	evidence, err := testEvidenceStore.LoadRequiredTestEvidenceV0(context.Background(), fixture.RunRef, testsStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].TaskRef != fixture.TaskRef ||
		evidence[0].DeliveryRef != fixture.DeliveryRef ||
		evidence[0].ReviewResultRef != fixture.ReviewResultRef ||
		evidence[0].AcceptedReviewRef != fixture.AcceptedReviewRef ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		!serviceStringInSetV0(evidence[0].EvidenceRefs, "artifact-ref-service-required-test-output-001") {
		t.Fatalf("evidence=%+v", evidence)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0BloqueaTestsConEvidenciaFailed(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:40Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		!serviceStringInSetV0(testsStep.BlockerRefs, "required-tests-failed") ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaTestsConEvidenciaDeOtraReview(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	evidence := serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
		fixture,
		orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
	)
	evidence.ReviewResultRef = "review-result-ref-otra-review"
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(evidence)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:50Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.ActiveStepID != "step-run-required-tests" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		len(testsStep.RequiredTestEvidenceRefs) != 0 {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeReviewAReplanSinTests(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, false)
	fixture.State.Mode = orquestadirectoroperativo.OperationalDirectorModeDomainWorkV0
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:21:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.ActiveStepID != "step-replan-or-close" ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		t.Fatalf("state=%+v replanStep=%+v", state, replanStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaReviewFueraDeScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Events[0] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
		DeliveryRef:  fixture.DeliveryRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       "task-ref-review-out-of-scope",
		AgentRef:     "agent-ref-review-out-of-scope",
		Summary:      "Entrega fuera del scope activo.",
		EvidenceRefs: []string{"evidence-ref-review-out-of-scope"},
	})
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:22:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ActiveStepID != "step-review-deliveries" {
		t.Fatalf("state=%+v", state)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaReviewConOutboxPendiente(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:23:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status:             orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		PendingOutboxCount: 1,
		Run:                fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ActiveStepID != "step-review-deliveries" {
		t.Fatalf("state=%+v", state)
	}
}

func TestContinueAppDirectorV0PlanStateReentradaRequiereStore(t *testing.T) {
	runRef := "run-app-director-operational-plan-reentry-missing-store"
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(serviceRunForWaitRefsTestV0(runRef))
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	_, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-17T13:57:00Z",
		CorrelationID:              "corr-app-director-operational-plan-reentry-missing-store",
		OperationalDirectorPlanRef: "plan-ref-missing-store",
	}, StartAppDirectorPortsV0{
		RunStore:     store,
		EventSink:    sink,
		OutboxLedger: ledger,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "ports.operational_plan_state_store" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReentraReviewConScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if got.WaitWaveRef != fixture.WaveRef ||
		got.WaitCohortRef != fixture.CohortRef ||
		got.WaitParentTaskRef != fixture.ParentTaskRef ||
		!serviceStringInSetV0(got.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("request=%+v fixture=%+v", got, fixture)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReentraRunRequiredTestsConScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	for index := range fixture.State.Steps {
		step := &fixture.State.Steps[index]
		switch step.StepID {
		case "step-review-deliveries":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			step.DeliveryRefs = []string{fixture.DeliveryRef}
			step.ReviewResultRefs = []string{fixture.ReviewResultRef}
			step.AcceptedReviewRefs = []string{fixture.AcceptedReviewRef}
		case "step-run-required-tests":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			step.TaskRefs = []string{fixture.TaskRef}
			step.AgentRefs = []string{fixture.AgentRef}
			step.DeliveryRefs = []string{fixture.DeliveryRef}
			step.ReviewResultRefs = []string{fixture.ReviewResultRef}
			step.BlockerRefs = []string{"required-tests-pending"}
		}
	}
	fixture.State.ActiveStepID = "step-run-required-tests"
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if got.WaitWaveRef != fixture.WaveRef ||
		got.WaitCohortRef != fixture.CohortRef ||
		got.WaitParentTaskRef != fixture.ParentTaskRef ||
		!serviceStringInSetV0(got.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("request=%+v fixture=%+v", got, fixture)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RechazaActiveStepNoSoportado(t *testing.T) {
	runRef := "run-app-director-operational-plan-reentry-unsupported"
	planRef := "plan-ref-reentry-unsupported"
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:   orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:        "state-ref-reentry-unsupported",
		PlanRef:         planRef,
		RequestRef:      "request-ref-reentry-unsupported",
		RunRef:          runRef,
		ProjectRef:      "orquesta",
		Mode:            orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:          orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:    "step-replan-or-close",
		ActiveWaveRef:   "wave-reentry-unsupported",
		ActiveCohortRef: "cohort-reentry-unsupported",
		ObservedAt:      "2026-05-17T13:58:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
			StepID:    "step-replan-or-close",
			Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
			Status:    orquestadirectoroperativo.OperationalDirectorStepRunningV0,
			WaveRef:   "wave-reentry-unsupported",
			CohortRef: "cohort-reentry-unsupported",
		}},
	}
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	_, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RespetaWaitExplicito(t *testing.T) {
	request := ContinueAppDirectorRequestV0{
		RunRef:                     "run-app-director-operational-plan-explicit-wait",
		OperationalDirectorPlanRef: "plan-ref-explicit-wait",
		WaitAgentRefs:              []string{"agent-ref-explicit"},
	}
	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, StartAppDirectorPortsV0{})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if len(got.WaitAgentRefs) != 1 || got.WaitAgentRefs[0] != "agent-ref-explicit" {
		t.Fatalf("request=%+v", got)
	}
}

func TestMaterializeContinueOperationalDirectorPlanV0UsaFaseActualSiNoSeIndica(t *testing.T) {
	runRef := "run-app-director-operational-plan-phase-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0
	run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{{
		ID:                  orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
		RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
	}}
	run.FunctionContracts = []string{contractRef}
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()

	materialized, err := materializeContinueOperationalDirectorPlanV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-17T13:47:00Z",
		CorrelationID:           "corr-app-director-operational-plan-phase-001",
		RequestedBy:             "orquesta-app-director-test",
		OperationalDirectorPlan: serviceOperationalDirectorPlanForContinueTestV0(t, runRef),
	}, StartAppDirectorPortsV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EventSink:         orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		DirectorTaskStore: taskStore,
	})
	if err != nil {
		t.Fatalf("materializeContinueOperationalDirectorPlanV0: %T %#v", err, err)
	}
	if len(materialized.Tasks) != 1 {
		t.Fatalf("materialized=%+v", materialized)
	}
	if materialized.Tasks[0].PhaseID != orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0 {
		t.Fatalf("phase_id=%s", materialized.Tasks[0].PhaseID)
	}
}

type serviceOperationalDirectorPlanStateReviewFixtureV0 struct {
	RunRef                  string
	PlanRef                 string
	TaskRef                 string
	AgentRef                string
	WaveRef                 string
	CohortRef               string
	ParentTaskRef           string
	DeliveryRef             string
	ReviewRequestID         string
	ReviewResultRef         string
	AcceptedReviewRef       string
	ReworkRequestRef        string
	ReplanDecisionRef       string
	RequiredTest            string
	RequiredTestEvidenceRef string
	Run                     orquestacoreworkflow.OrchestrationRunV0
	State                   orquestacionnucleoapp.OperationalDirectorPlanStateV0
	Events                  []orquestacoreworkflow.OrchestrationEventV0
}

func serviceOperationalDirectorPlanStateReviewFixtureForTestV0(
	t *testing.T,
	withRequiredTests bool,
) serviceOperationalDirectorPlanStateReviewFixtureV0 {
	t.Helper()
	runRef := "run-app-director-operational-plan-state-review-accepted"
	planRef := "plan-ref-app-director-operational-plan-state-review-accepted"
	taskRef := "task-ref-app-director-operational-plan-state-review-accepted"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	waveRef := "wave-app-director-operational-plan-state-review-accepted"
	cohortRef := "cohort-app-director-operational-plan-state-review-accepted"
	parentTaskRef := "parent-task-ref-app-director-operational-plan-state-review-accepted"
	deliveryRef := "delivery-ref-app-director-operational-plan-state-review-accepted"
	reviewRequestID := "review-request-ref-app-director-operational-plan-state-review-accepted"
	reviewResultRef := "review-result-ref-app-director-operational-plan-state-review-accepted"
	acceptedReviewRef := "accepted-review-ref-app-director-operational-plan-state-review-accepted"
	requiredTest := "go test -count=1 ./modulos/orquesta-app-director-service"
	requiredTestEvidenceRef := "test-evidence-ref-" + deliveryRef
	requiredTests := []string(nil)
	reviewResultEvidenceRefs := []string{"evidence-ref-review-result-accepted"}
	if withRequiredTests {
		requiredTests = []string{requiredTest}
		reviewResultEvidenceRefs = append(reviewResultEvidenceRefs, requiredTestEvidenceRef)
	}
	run := serviceRunForWaitRefsTestV0(runRef, taskRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{{
		ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
		RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
	}}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	run.DeliveredTasks = []string{taskRef}
	run.Deliveries = []string{deliveryRef}
	run.Reviews = []string{reviewRequestID}
	run.ReviewResults = []string{reviewResultRef + "#review_result:accepted#review_request:" + reviewRequestID + "#delivery:" + deliveryRef}
	run.AcceptedReviews = []string{acceptedReviewRef}
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "state-ref-app-director-operational-plan-state-review-accepted",
		PlanRef:             planRef,
		RequestRef:          "request-ref-app-director-operational-plan-state-review-accepted",
		RunRef:              runRef,
		ProjectRef:          "orquesta",
		Mode:                orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        "step-review-deliveries",
		ActiveWaveRef:       waveRef,
		ActiveCohortRef:     cohortRef,
		ActiveParentTaskRef: parentTaskRef,
		RequiredTestRefs:    requiredTests,
		ObservedAt:          "2026-05-17T14:19:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:        "step-launch-subagents",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      []string{taskRef},
				AgentRefs:     []string{agentRef},
			},
			{
				StepID:        "step-wait-subagents",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      []string{taskRef},
				AgentRefs:     []string{agentRef},
				WaitRefs:      []string{"wait-ref-app-director-operational-plan-state-review-accepted"},
			},
			{
				StepID:        "step-review-deliveries",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      []string{taskRef},
				AgentRefs:     []string{agentRef},
			},
			{
				StepID:        "step-run-required-tests",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
			{
				StepID:        "step-replan-or-close",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
		},
	}
	events := []orquestacoreworkflow.OrchestrationEventV0{
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  deliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       taskRef,
			AgentRef:     agentRef,
			Summary:      "Entrega del scope activo.",
			EvidenceRefs: []string{"evidence-ref-delivery-review-accepted"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 2, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: reviewRequestID,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     deliveryRef,
			Summary:         "Review del scope activo.",
			EvidenceRefs:    []string{"evidence-ref-review-requested"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: reviewResultRef,
			ReviewRequestID: reviewRequestID,
			DeliveryRef:     deliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Review aceptada.",
			EvidenceRefs:    reviewResultEvidenceRefs,
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, runRef, 4, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: acceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   reviewRequestID,
			DeliveryRef:       deliveryRef,
			Summary:           "Review aceptada.",
			EvidenceRefs:      []string{"evidence-ref-review-accepted"},
		}),
	}
	return serviceOperationalDirectorPlanStateReviewFixtureV0{
		RunRef:                  runRef,
		PlanRef:                 planRef,
		TaskRef:                 taskRef,
		AgentRef:                agentRef,
		WaveRef:                 waveRef,
		CohortRef:               cohortRef,
		ParentTaskRef:           parentTaskRef,
		DeliveryRef:             deliveryRef,
		ReviewRequestID:         reviewRequestID,
		ReviewResultRef:         reviewResultRef,
		AcceptedReviewRef:       acceptedReviewRef,
		RequiredTest:            requiredTest,
		RequiredTestEvidenceRef: requiredTestEvidenceRef,
		Run:                     run,
		State:                   state,
		Events:                  events,
	}
}

func serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
	t *testing.T,
	status orquestacoreworkflow.ReviewResultStatusV0,
) serviceOperationalDirectorPlanStateReviewFixtureV0 {
	t.Helper()
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.ReviewResultRef = "review-result-ref-app-director-operational-plan-state-review-" + string(status)
	fixture.AcceptedReviewRef = ""
	fixture.ReworkRequestRef = "rework-request-ref-app-director-operational-plan-state-review-" + string(status)
	fixture.ReplanDecisionRef = "replan-decision-ref-app-director-operational-plan-state-review-" + string(status)
	fixture.Run.ReviewResults = []string{
		fixture.ReviewResultRef + "#review_result:" + string(status) +
			"#review_request:" + fixture.ReviewRequestID +
			"#delivery:" + fixture.DeliveryRef,
	}
	fixture.Run.AcceptedReviews = nil
	fixture.Run.ReworkRequests = []string{
		fixture.ReworkRequestRef + "#review_result:" + fixture.ReviewResultRef +
			"#review_request:" + fixture.ReviewRequestID +
			"#delivery:" + fixture.DeliveryRef,
	}
	fixture.Run.ReplanDecisions = []string{
		fixture.ReplanDecisionRef + "#source:" + fixture.ReworkRequestRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + fixture.TaskRef,
	}
	fixture.Events = []orquestacoreworkflow.OrchestrationEventV0{
		fixture.Events[0],
		fixture.Events[1],
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: fixture.ReviewResultRef,
			ReviewRequestID: fixture.ReviewRequestID,
			DeliveryRef:     fixture.DeliveryRef,
			Status:          status,
			Summary:         "Review pide cambios.",
			EvidenceRefs:    []string{"evidence-ref-review-result-" + string(status)},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 4, orquestacoreworkflow.OrchestrationEventReworkRequestedV0, orquestacoreworkflow.ReworkRequestedPayloadV0{
			ReworkRequestRef: fixture.ReworkRequestRef,
			PhaseID:          string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewResultRef:  fixture.ReviewResultRef,
			ReviewRequestID:  fixture.ReviewRequestID,
			DeliveryRef:      fixture.DeliveryRef,
			Summary:          "Rework requerido por review negativa.",
			EvidenceRefs:     []string{"evidence-ref-rework-requested-" + string(status)},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      fixture.ReplanDecisionRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      fixture.ReworkRequestRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{fixture.TaskRef},
			Summary:        "Replan por review negativa.",
			EvidenceRefs:   []string{"evidence-ref-replan-decision-" + string(status)},
		}),
	}
	return fixture
}

func serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
	status orquestacionnucleoapp.RequiredTestEvidenceStatusV0,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       fixture.RequiredTestEvidenceRef,
		RunRef:            fixture.RunRef,
		TaskRef:           fixture.TaskRef,
		TestCommand:       fixture.RequiredTest,
		Status:            status,
		DeliveryRef:       fixture.DeliveryRef,
		ReviewRequestID:   fixture.ReviewRequestID,
		ReviewResultRef:   fixture.ReviewResultRef,
		AcceptedReviewRef: fixture.AcceptedReviewRef,
		OccurredAt:        "2026-05-17T14:20:00Z",
		EvidenceRefs:      []string{"test-output-ref-" + fixture.DeliveryRef},
	}
}

type fakeServiceRequiredTestCommandExecutorV0 struct {
	results  map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0
	commands []string
}

func (executor *fakeServiceRequiredTestCommandExecutorV0) RunRequiredTestCommandV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestCommandExecutionResultV0, error) {
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	executor.commands = append(executor.commands, request.TestCommand)
	return executor.results[request.TestCommand], nil
}

func serviceOperationalDirectorPlanStateEventForTestV0(
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
		EventID:        eventType + "-plan-state-review-" + string(rune('0'+sequence)),
		EventType:      eventType,
		RunID:          runRef,
		Sequence:       sequence,
		OccurredAt:     "2026-05-17T14:19:00Z",
		PayloadVersion: orquestacoreworkflow.OrchestrationEventPayloadVersionV0,
		Payload:        raw,
	}
}

func serviceOperationalDirectorPlanStateStepForTestV0(
	t *testing.T,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	stepID string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	t.Helper()
	for _, step := range state.Steps {
		if step.StepID == stepID {
			return step
		}
	}
	t.Fatalf("step %s no encontrado en %+v", stepID, state.Steps)
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
}

func serviceOperationalDirectorPlanForContinueTestV0(
	t *testing.T,
	runRef string,
) orquestadirectoroperativo.OperationalDirectorPlanV0 {
	t.Helper()
	result := orquestadirectoroperativo.BuildOperationalDirectorPlanV0(orquestadirectoroperativo.OperationalDirectorRequestV0{
		RequestRef:       "req-app-director-operational-plan-001",
		RunRef:           runRef,
		ProjectRef:       "orquesta",
		Objective:        "Cerrar tramo launch wait del Director Operativo.",
		Mode:             orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		WorktreeRef:      "worktree-ref-app-director-operational-plan",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-app-director-operational-plan",
		WriteSet: []string{
			"docs/director_operational_plan_test.md",
		},
		RequiredTests: []string{"go test -count=1 ./modulos/orquesta-app-director-service"},
	})
	if !result.Accepted || !result.ReadyToLaunch {
		t.Fatalf("plan no listo: %+v", result)
	}
	return result.Plan
}
