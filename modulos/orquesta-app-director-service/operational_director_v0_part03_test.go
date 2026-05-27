package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueOperationalDirectorPlanStatePostLoopV0ConsumeWaitAntesDeExpirarMaxWaitCero(t *testing.T) {
	runRef := "run-app-director-operational-plan-state-max-wait-zero-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.FunctionContracts = []string{contractRef}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)
	ports := StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  sink,
		OutboxLedger:               ledger,
		DirectorTaskStore:          taskStore,
		WaitStateStore:             waitStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	}

	first, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-22T16:20:00Z",
		CorrelationID:           "corr-app-director-operational-plan-state-max-wait-zero-first",
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
	}, ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	if len(first.Run.Tasks) != 1 {
		t.Fatalf("tasks=%v", first.Run.Tasks)
	}
	initialState, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 initial: %v", err)
	}
	initialWaitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, initialState, "step-wait-subagents")
	if len(initialWaitStep.WaitRefs) != 1 {
		t.Fatalf("initial wait refs=%+v", initialWaitStep.WaitRefs)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(first.Run.Tasks[0])
	deliveredRun := first.Run
	deliveredRun.DeliveredAgents = []string{agentRef}
	if err := store.SaveRunV0(context.Background(), deliveredRun); err != nil {
		t.Fatalf("SaveRunV0 delivered: %v", err)
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T16:20:01Z",
		CorrelationID:              "corr-app-director-operational-plan-state-max-wait-zero-post",
		OperationalDirectorPlanRef: plan.PlanRef,
		MaxBursts:                  4,
		MaxStepsPerBurst:           4,
		MaxDispatchesPerWait:       4,
		MaxCommands:                20,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           0,
	}
	_, err = continueOperationalDirectorPlanStatePostLoopV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status:             orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Run:                deliveredRun,
			PendingOutboxCount: 1,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef: runRef,
		},
		orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
				Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
				Run:    deliveredRun,
			},
			Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{{
				AttemptNumber: 1,
				Result: orquestacionnucleoapp.ProgressiveLoopResultV0{
					Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
					Run:    deliveredRun,
				},
			}},
		},
	)
	if err != nil {
		t.Fatalf("continueOperationalDirectorPlanStatePostLoopV0 pending outbox: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 pending outbox: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason == "external-wait-exhausted" {
		t.Fatalf("state avanzo con outbox pendiente: state=%+v wait=%+v", state, waitStep)
	}
	_, err = continueOperationalDirectorPlanStatePostLoopV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Run:    deliveredRun,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef: runRef,
		},
		orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
				Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
				Run:    deliveredRun,
			},
			Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{{
				AttemptNumber: 1,
				Result: orquestacionnucleoapp.ProgressiveLoopResultV0{
					Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
					Run:    deliveredRun,
				},
			}},
		},
	)
	if err != nil {
		t.Fatalf("continueOperationalDirectorPlanStatePostLoopV0 delivered: %v", err)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 final: %v", err)
	}
	waitStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-review-deliveries" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		state.ClosureReason != "" {
		t.Fatalf("state expiro en vez de avanzar: state=%+v wait=%+v review=%+v", state, waitStep, reviewStep)
	}
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, initialWaitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 final: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusContinuedV0 ||
		serviceStringInSetV0(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0") {
		t.Fatalf("waitState=%+v", waitState)
	}
}
