package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
	"testing"
)

func TestContinueAppDirectorV0ReviewChangesRequestedSplitTaskDurableV0(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	firstFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-continue-review-split-a",
		fixture.TaskRef,
	)
	secondFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-continue-review-split-b",
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

	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := eventSink.AppendRunEventsV0(context.Background(), fixture.RunRef, []orquestacoreworkflow.OrchestrationEventV0{fixture.Events[0]}); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	outboxLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(parentTask)
	waitStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	reviewSource := &changesRequestedServiceReviewGateSourceV0{Fixture: fixture}

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-23T10:10:00Z",
		CorrelationID:              "corr-app-director-continue-review-split",
		OperationalDirectorPlanRef: fixture.PlanRef,
		MaxBursts:                  12,
		MaxStepsPerBurst:           8,
		MaxDispatchesPerWait:       4,
		MaxCommands:                40,
		MaxOutboxPerCycle:          12,
	}, StartAppDirectorPortsV0{
		RunStore:          runStore,
		EventSink:         eventSink,
		EventReader:       eventSink,
		OutboxLedger:      outboxLedger,
		DirectorTaskStore: taskStore,
		WaitStateWriter:   waitStore,
		WaitStateStore:    waitStore,
		ReviewGateSource:  reviewSource,
		ReviewReworkReplanSource: serviceOperationalDirectorSplitReviewReworkReplanSourceV0{
			Fixture:    fixture,
			SplitTasks: []orquestacoreworkflow.WorkflowTaskV0{firstFollowup, secondFollowup},
		},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outboxLedger),
			serviceAgentLauncherDispatcherForTestV0(runStore, eventSink, outboxLedger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}

	followupAgentRefs := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstFollowup.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(secondFollowup.TaskID),
	}
	if !reviewSource.Called ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		!strings.Contains(strings.Join(result.Run.ReworkRequests, " "), fixture.ReworkRequestRef) ||
		!strings.Contains(strings.Join(result.Run.ReplanDecisions, " "), fixture.ReplanDecisionRef) ||
		!serviceStringInSetV0(result.Run.Tasks, firstFollowup.TaskID) ||
		!serviceStringInSetV0(result.Run.Tasks, secondFollowup.TaskID) ||
		!serviceStringInSetV0(result.Run.StartedAgents, followupAgentRefs[0]) ||
		!serviceStringInSetV0(result.Run.StartedAgents, followupAgentRefs[1]) {
		t.Fatalf("split no aplicado: called=%v result=%+v followups=%+v", reviewSource.Called, result, followupAgentRefs)
	}
	loaded, err := taskStore.LoadWorkflowTasksV0(context.Background(), fixture.RunRef, []string{firstFollowup.TaskID, secondFollowup.TaskID})
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0 followups: %v", err)
	}
	if len(loaded) != 2 ||
		loaded[0].ParentTaskRef != fixture.TaskRef ||
		loaded[1].ParentTaskRef != fixture.TaskRef {
		t.Fatalf("followups no persistidos como hijos: %+v", loaded)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveParentTaskRef != fixture.TaskRef ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		waitStep.Reason != "review-rework-replan-followups-waiting" ||
		!serviceStringInSetV0(waitStep.TaskRefs, firstFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.TaskRefs, secondFollowup.TaskID) ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
		!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) {
		t.Fatalf("state=%+v waitStep=%+v reviewStep=%+v", state, waitStep, reviewStep)
	}
	if got := serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("replan no idempotente en ciclo integrado: got=%d events=%+v", got, eventSink.EventsV0())
	}
}
