package orquestaappdirectorservice

import (
	"context"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

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
