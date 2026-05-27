package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueAppDirectorV0BloqueaWaitSubagentsCuandoManagedWaitExpira(t *testing.T) {
	runRef := "run-app-director-operational-wait-expired-001"
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
	waiter := &serviceContinueWaiterForTestV0{Continue: true}

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-22T16:10:00Z",
		CorrelationID:           "corr-app-director-operational-wait-expired-001",
		MaxBursts:               4,
		MaxStepsPerBurst:        4,
		MaxDispatchesPerWait:    4,
		MaxCommands:             20,
		MaxOutboxPerCycle:       8,
		MaxExternalWaits:        2,
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
		WaitStateStore:             waitStore,
		WaitStateWriter:            waitStore,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		ExternalWaiter:             waiter,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
			serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		waiter.Calls != 2 {
		t.Fatalf("result=%+v waiter_calls=%d", result, waiter.Calls)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		!serviceStringInSetV0(state.BlockerRefs, "external-wait-exhausted") ||
		state.ClosureReason != "external-wait-exhausted" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		waitStep.Reason != "external-wait-exhausted" ||
		!serviceStringInSetV0(waitStep.BlockerRefs, "external-wait-exhausted") ||
		len(waitStep.PendingAgentRefs) != 0 ||
		len(state.PendingAgentRefs) != 0 {
		t.Fatalf("state=%+v waitStep=%+v", state, waitStep)
	}
	if len(waitStep.WaitRefs) != 1 {
		t.Fatalf("wait refs=%+v", waitStep.WaitRefs)
	}
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusExpiredV0 ||
		waitState.Attempt != 3 ||
		len(waitState.PendingAgentRefs) != 0 ||
		serviceCountStringV0(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0") != 1 {
		t.Fatalf("waitState=%+v", waitState)
	}
	changed, err := applyOperationalDirectorPlanStateAfterExternalWaitExhaustedV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T16:10:01Z",
		OperationalDirectorPlanRef: plan.PlanRef,
		MaxExternalWaits:           2,
	}, StartAppDirectorPortsV0{
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		WaitStateStore:             waitStore,
		WaitStateWriter:            waitStore,
	}, orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
		Final:  orquestacionnucleoapp.ProgressiveLoopResultV0{Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0, Run: result.Run},
		Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{
			{AttemptNumber: 1, Result: orquestacionnucleoapp.ProgressiveLoopResultV0{Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0}},
			{AttemptNumber: 2, Result: orquestacionnucleoapp.ProgressiveLoopResultV0{Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0}},
			{AttemptNumber: 3, Result: orquestacionnucleoapp.ProgressiveLoopResultV0{Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0}},
		},
		ExternalWaits: []orquestacionnucleoapp.ManagedProgressiveLoopExternalWaitV0{
			{WaitNumber: 1, Continue: true},
			{WaitNumber: 2, Continue: true},
		},
	})
	if err != nil {
		t.Fatalf("applyOperationalDirectorPlanStateAfterExternalWaitExhaustedV0 replay: %v", err)
	}
	if changed {
		t.Fatalf("replay changed blocked state")
	}
	replayedWaitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 replay: %v", err)
	}
	if serviceCountStringV0(replayedWaitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0") != 1 {
		t.Fatalf("replayedWaitState=%+v", replayedWaitState)
	}
}

func TestContinueOperationalDirectorPlanStatePostLoopV0MaxExternalWaitsCeroNoExpiraEsperaPendiente(t *testing.T) {
	runRef := "run-app-director-operational-wait-zero-pending-001"
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

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-22T16:30:00Z",
		CorrelationID:           "corr-app-director-operational-wait-zero-pending-001",
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
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("loop_status=%s result=%+v", result.LoopStatus, result)
	}

	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T16:30:01Z",
		CorrelationID:              "corr-app-director-operational-wait-zero-pending-002",
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
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Run:    result.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{
			RunRef: runRef,
		},
		orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
				Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
				Run:    result.Run,
			},
			Attempts: []orquestacionnucleoapp.ManagedProgressiveLoopAttemptV0{{
				AttemptNumber: 1,
				Result: orquestacionnucleoapp.ProgressiveLoopResultV0{
					Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
					Run:    result.Run,
				},
			}},
		},
	)
	if err != nil {
		t.Fatalf("continueOperationalDirectorPlanStatePostLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, plan.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason == "external-wait-exhausted" ||
		state.ClosureReason != "" {
		t.Fatalf("state expiro sin waits internos: state=%+v wait=%+v", state, waitStep)
	}
	waitState, err := waitStore.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0 ||
		serviceStringInSetV0(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0") {
		t.Fatalf("waitState=%+v", waitState)
	}
}
