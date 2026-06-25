package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestRuntimeResidentDirectorConduceDirectorCycleStepTrasRestartV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	ctx := context.Background()
	config := directorCycleResidentConfigV0(t)
	config.Addr = "127.0.0.1:0"
	config.ResidentDirectorEnabled = true
	config.ResidentDirectorMaxActions = 3
	config.TickInterval = 10 * time.Millisecond
	config.AuditDisabled = true
	config.IdleSelfImprovementDisabled = true
	config.ShutdownGracePeriod = 500 * time.Millisecond
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	workflow := newResidentCycleWorkflowV0(t)
	firstDirector := &residentCycleRuntimeDirectorV0{
		workflow: workflow,
		ledger:   stack.Stores.OutboxLedger,
	}
	firstRuntime, err := orquestaserver.NewRuntimeV0(config, orquestaserver.RuntimeDepsV0{
		ResidentDirector: firstDirector,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0 first: %v", err)
	}

	firstCancelCtx, firstCancel := context.WithCancel(ctx)
	firstDone := make(chan error, 1)
	go func() { firstDone <- firstRuntime.RunV0(firstCancelCtx) }()
	firstState := waitRuntimeResidentDirectorTicksForTestV0(t, firstRuntime, 1)
	firstCancel()
	if err := waitRuntimeDoneForCycleResidentTestV0(t, firstDone); err != nil {
		t.Fatalf("first RunV0: %v", err)
	}
	if firstDirector.calls != 1 ||
		len(firstDirector.results) != 1 ||
		firstDirector.results[0].Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 ||
		firstState.ResidentDirectorLastResult != string(orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0) ||
		firstState.ResidentDirectorExecutedActions != 1 {
		t.Fatalf("first state=%+v director=%+v", firstState, firstDirector)
	}

	restartedStack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("restart buildStackFromEnvV0: %v", err)
	}
	secondDirector := &residentCycleRuntimeDirectorV0{
		workflow: workflow,
		ledger:   restartedStack.Stores.OutboxLedger,
	}
	secondRuntime, err := orquestaserver.NewRuntimeV0(config, orquestaserver.RuntimeDepsV0{
		ResidentDirector: secondDirector,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0 second: %v", err)
	}
	secondCancelCtx, secondCancel := context.WithCancel(ctx)
	secondDone := make(chan error, 1)
	go func() { secondDone <- secondRuntime.RunV0(secondCancelCtx) }()
	secondState := waitRuntimeResidentDirectorTicksForTestV0(t, secondRuntime, 2)
	secondCancel()
	if err := waitRuntimeDoneForCycleResidentTestV0(t, secondDone); err != nil {
		t.Fatalf("second RunV0: %v", err)
	}
	if secondDirector.calls != 1 ||
		len(secondDirector.results) != 1 ||
		secondDirector.results[0].Status != orquestadirectorrunner.DirectorCycleStatusWaitingV0 ||
		!reflect.DeepEqual(secondDirector.results[0].PendingOutboxBeforeRefs, firstDirector.results[0].PendingOutboxAfterRefs) ||
		secondState.ResidentDirectorLastResult != string(orquestadirectorrunner.DirectorCycleStatusWaitingV0) {
		t.Fatalf("second state=%+v director=%+v first=%+v", secondState, secondDirector, firstDirector)
	}
	pending, issues := restartedStack.Stores.OutboxLedger.ListPending(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef: workflow.run.RunID,
	})
	if len(issues) != 0 || len(pending) != 1 {
		t.Fatalf("pending=%+v issues=%+v", pending, issues)
	}
}

func TestDirectorCycleResidentRestartSmokeV0(t *testing.T) {
	ctx := context.Background()
	config := directorCycleResidentConfigV0(t)
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	workflow := newResidentCycleWorkflowV0(t)

	first, err := orquestadirectorcycle.ExecuteDirectorCycleStepV0(
		ctx,
		residentCycleInputV0(workflow, stack.Stores.OutboxLedger, "cycle-ref-resident-001", "tick-ref-resident-001"),
	)
	if err != nil {
		t.Fatalf("first cycle: %v", err)
	}
	if first.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 ||
		first.OutboxSavedCount != 1 ||
		first.OutboxPendingAfterCount != 1 {
		t.Fatalf("first result=%+v", first)
	}

	restartedStack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("restart buildStackFromEnvV0: %v", err)
	}
	second, err := orquestadirectorcycle.ExecuteDirectorCycleStepV0(
		ctx,
		residentCycleInputV0(workflow, restartedStack.Stores.OutboxLedger, "cycle-ref-resident-002", "tick-ref-resident-002"),
	)
	if err != nil {
		t.Fatalf("second cycle: %v", err)
	}
	if second.Status != orquestadirectorrunner.DirectorCycleStatusWaitingV0 ||
		second.OutboxSavedCount != 0 ||
		!reflect.DeepEqual(second.PendingOutboxBeforeRefs, first.PendingOutboxAfterRefs) {
		t.Fatalf("second result=%+v first=%+v", second, first)
	}

	executor := residentCycleDispatchExecutorV0{}
	dispatched, err := orquestaoutboxdispatch.RunOutboxDispatchOnceV0(orquestaoutboxdispatch.RunOutboxDispatchOnceInputV0{
		RunID:       workflow.run.RunID,
		TargetPort:  orquestacoreworkflow.OutboxTargetCapacityV0,
		MessageType: orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
		Reader:      restartedStack.Stores.OutboxLedger,
		Claimer:     restartedStack.Stores.OutboxLedger,
		Executor:    &executor,
		Acker:       restartedStack.Stores.OutboxLedger,
	})
	if err != nil || dispatched.Status != orquestaoutboxdispatch.RunOutboxDispatchOnceDispatchedV0 {
		t.Fatalf("dispatch result=%+v err=%v", dispatched, err)
	}
	failedAck := orquestaoutboxdispatch.PlanOutboxDispatchAckClosureV0(orquestaoutboxdispatch.OutboxDispatchAckClosureInputV0{
		Intents: []orquestaoutboxdispatch.DispatchIntentV0{dispatched.Intent},
		Acks: []orquestaoutboxdispatch.OutboxDispatchAckObservationV0{{
			MessageID:    dispatched.Intent.MessageID,
			RunID:        dispatched.Intent.RunID,
			TargetPort:   dispatched.Intent.TargetPort,
			Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationFailedV0,
			EvidenceRefs: []string{"evidence-ref-resident-cycle-failed-ack"},
		}},
	})
	if len(failedAck.Failed) != 1 ||
		failedAck.Failed[0].Status != orquestaoutboxdispatch.OutboxDispatchAckClosureFailedV0 {
		t.Fatalf("failed ack closure=%+v", failedAck)
	}

	afterAckStack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("after ack buildStackFromEnvV0: %v", err)
	}
	third, err := orquestadirectorcycle.ExecuteDirectorCycleStepV0(
		ctx,
		residentCycleInputV0(workflow, afterAckStack.Stores.OutboxLedger, "cycle-ref-resident-003", "tick-ref-resident-003"),
	)
	if err != nil {
		t.Fatalf("third cycle: %v", err)
	}
	if third.Status != orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0 ||
		third.OutboxSavedCount != 1 ||
		reflect.DeepEqual(third.PendingOutboxAfterRefs, first.PendingOutboxAfterRefs) {
		t.Fatalf("third result=%+v first=%+v", third, first)
	}
	if executor.calls != 1 {
		t.Fatalf("dispatch calls=%d", executor.calls)
	}
}
