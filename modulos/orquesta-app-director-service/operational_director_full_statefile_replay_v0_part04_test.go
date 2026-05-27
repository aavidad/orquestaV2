package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	"testing"
)

func TestContinueAppDirectorV0StateFileReplayReviewChangesRequestedSplitTaskDurableNoDuplica(t *testing.T) {
	ctx := context.Background()
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	firstFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-service-statefile-review-split-a",
		fixture.TaskRef,
	)
	secondFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-service-statefile-review-split-b",
		fixture.TaskRef,
	)
	parentTask := serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)
	parentTask.WaveRef = firstFollowup.WaveRef
	parentTask.CohortRef = firstFollowup.CohortRef
	parentTask.DelegationDepth = 0
	parentTask.MaxChildAgents = 2
	run := fixture.Run
	run.Tasks = []string{fixture.TaskRef}
	run.Agents = []string{fixture.AgentRef}
	run.StartedAgents = []string{fixture.AgentRef}
	run.DeliveredAgents = []string{fixture.AgentRef}
	run.DeliveredTasks = []string{fixture.TaskRef}
	run.Deliveries = []string{fixture.DeliveryRef}
	run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusPendingV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
	}
	run.Reviews = nil
	run.ReviewResults = nil
	run.AcceptedReviews = nil
	run.ReworkRequests = nil
	run.ReplanDecisions = nil
	run.FunctionContracts = []string{"contract:function:rework-split:v0"}
	run.LastEventID = fixture.Events[0].EventID
	run.LastSequence = 1

	rootDir := t.TempDir()
	store := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, fixture.RunRef, []orquestacoreworkflow.OrchestrationEventV0{fixture.Events[0]}); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	if err := store.SaveWorkflowTaskV0(ctx, parentTask); err != nil {
		t.Fatalf("SaveWorkflowTaskV0 parent: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, fixture.State); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	reviewSource := &changesRequestedServiceReviewGateSourceV0{Fixture: fixture}
	splitSource := serviceOperationalDirectorSplitReviewReworkReplanSourceV0{
		Fixture:    fixture,
		SplitTasks: []orquestacoreworkflow.WorkflowTaskV0{firstFollowup, secondFollowup},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-23T10:30:00Z",
		CorrelationID:              "corr-service-statefile-review-split-first",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
		MaxBursts:                  12,
		MaxStepsPerBurst:           8,
		MaxDispatchesPerWait:       4,
		MaxCommands:                40,
		MaxOutboxPerCycle:          12,
	}
	first, err := ContinueAppDirectorV0(
		ctx,
		request,
		serviceFullReplayStateFileSplitPortsForTestV0(store, ledger, reviewSource, splitSource),
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	followupAgentRefs := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstFollowup.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(secondFollowup.TaskID),
	}
	if !reviewSource.Called ||
		first.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		!serviceStringInSetV0(first.Run.Tasks, firstFollowup.TaskID) ||
		!serviceStringInSetV0(first.Run.Tasks, secondFollowup.TaskID) ||
		!serviceStringInSetV0(first.Run.StartedAgents, followupAgentRefs[0]) ||
		!serviceStringInSetV0(first.Run.StartedAgents, followupAgentRefs[1]) {
		t.Fatalf("split state-file no aplicado: called=%v result=%+v followups=%+v", reviewSource.Called, first, followupAgentRefs)
	}
	serviceAssertFullReplayReviewSplitCountsV0(t, store, fixture, 1, 1, 2, 2)

	recovered := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	loaded, err := recovered.LoadWorkflowTasksV0(ctx, fixture.RunRef, []string{firstFollowup.TaskID, secondFollowup.TaskID})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0 recovered: %v", err)
	}
	if len(loaded) != 2 ||
		loaded[0].ParentTaskRef != fixture.TaskRef ||
		loaded[1].ParentTaskRef != fixture.TaskRef ||
		loaded[0].WaveRef != firstFollowup.WaveRef ||
		loaded[1].CohortRef != firstFollowup.CohortRef ||
		len(loaded[0].FunctionContractRefs) != 1 ||
		len(loaded[1].FunctionContractRefs) != 1 {
		t.Fatalf("followups recuperados invalidos: %+v", loaded)
	}
	recoveredState := serviceAssertFullReplayReviewSplitWaitStateV0(t, recovered, fixture, firstFollowup, secondFollowup, followupAgentRefs)
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recoveredState, "step-wait-subagents")
	if len(waitStep.WaitRefs) != 1 {
		t.Fatalf("wait refs invalidas: state=%+v waitStep=%+v", recoveredState, waitStep)
	}
	waitState, err := recovered.LoadWorkflowTaskWaitStateV0(ctx, fixture.RunRef, waitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0 recovered: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0 ||
		waitState.WaveRef != firstFollowup.WaveRef ||
		waitState.CohortRef != firstFollowup.CohortRef ||
		waitState.ParentTaskRef != fixture.TaskRef ||
		!serviceStringInSetV0(waitState.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(waitState.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(waitState.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("wait state recuperado invalido: %+v", waitState)
	}

	replayLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	request.OccurredAt = "2026-05-23T10:30:01Z"
	request.CorrelationID = "corr-service-statefile-review-split-replay"
	replay, err := ContinueAppDirectorV0(
		ctx,
		request,
		serviceFullReplayStateFileSplitPortsForTestV0(
			recovered,
			replayLedger,
			&changesRequestedServiceReviewGateSourceV0{Fixture: fixture},
			splitSource,
		),
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 replay: %v", err)
	}
	if replay.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		!serviceStringInSetV0(replay.Run.StartedAgents, followupAgentRefs[0]) ||
		!serviceStringInSetV0(replay.Run.StartedAgents, followupAgentRefs[1]) {
		t.Fatalf("replay wait invalido: %+v", replay)
	}
	serviceAssertFullReplayReviewSplitCountsV0(t, recovered, fixture, 1, 1, 2, 2)
	serviceAssertFullReplayReviewSplitWaitStateV0(t, recovered, fixture, firstFollowup, secondFollowup, followupAgentRefs)
}

func serviceFullReplayStateFileStoreForTestV0(t *testing.T, rootDir string) *orquestastatefile.StoreV0 {
	t.Helper()
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: rootDir})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	return store
}

func serviceFullReplayStateFilePortsForTestV0(
	store *orquestastatefile.StoreV0,
	executor *fakeServiceRequiredTestCommandExecutorV0,
	closureSource *serviceOperationalDirectorClosureSourceFromRequestForTestV0,
) StartAppDirectorPortsV0 {
	dispatchRunStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	dispatchEventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	dispatchLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	return StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		EventReader:                store,
		OutboxLedger:               dispatchLedger,
		DirectorTaskStore:          store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		RequiredTestEvidenceStore:  store,
		RequiredTestRunner: orquestacionnucleoapp.RequiredTestRunnerV0{
			Executor:       executor,
			EvidenceWriter: store,
		},
		OperationalClosureSource: closureSource,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(dispatchRunStore, dispatchEventSink, dispatchLedger),
		},
	}
}

func serviceFullReplayStateFileClosurePortsForTestV0(
	store *orquestastatefile.StoreV0,
	closureSource *serviceOperationalClosureSourceForTestV0,
) StartAppDirectorPortsV0 {
	return StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		EventReader:                store,
		DirectorTaskStore:          store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		RequiredTestEvidenceStore:  store,
		OperationalClosureSource:   closureSource,
	}
}
