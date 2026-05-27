package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
	"testing"
)

func TestReviewReworkSplitTaskDurableReabreWaitSoloFollowupsNuevos(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	firstFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-review-rework-durable-split-a",
		fixture.TaskRef,
	)
	secondFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-review-rework-durable-split-b",
		fixture.TaskRef,
	)
	parentTask := serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)
	parentTask.WaveRef = firstFollowup.WaveRef
	parentTask.CohortRef = firstFollowup.CohortRef
	parentTask.DelegationDepth = 0
	parentTask.MaxChildAgents = 2
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(parentTask)
	provider := composeStartAppDirectorProviderV0(
		orquestacionnucleoapp.StaticCandidateProviderV0{},
		StartAppDirectorPortsV0{
			DirectorTaskStore: taskStore,
			ReviewReworkReplanSource: serviceOperationalDirectorSplitReviewReworkReplanSourceV0{
				Fixture:    fixture,
				SplitTasks: []orquestacoreworkflow.WorkflowTaskV0{firstFollowup, secondFollowup},
			},
		},
		"director-service-test",
	)

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run:           fixture.Run,
		OccurredAt:    "2026-05-23T09:10:00Z",
		CorrelationID: "corr-app-director-review-rework-durable-split",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ReplanFollowupCandidates) != 1 {
		t.Fatalf("replan followups=%d candidates=%+v", len(candidates.ReplanFollowupCandidates), candidates)
	}
	input := candidates.ReplanFollowupCandidates[0].ReplanFollowupsInput
	if input.DecisionPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionSplitTaskV0 ||
		len(input.MicrotaskCandidates) != 2 {
		t.Fatalf("input=%+v", input)
	}
	loaded, err := taskStore.LoadWorkflowTasksV0(context.Background(), fixture.RunRef, input.DecisionPayload.FollowupRefs)
	if err != nil {
		t.Fatalf("LoadWorkflowTasksV0 followups: %v", err)
	}
	if len(loaded) != 2 ||
		loaded[0].ParentTaskRef != fixture.TaskRef ||
		loaded[1].ParentTaskRef != fixture.TaskRef {
		t.Fatalf("loaded followups=%+v", loaded)
	}

	followupAgentRefs := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstFollowup.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(secondFollowup.TaskID),
	}
	run := fixture.Run
	run.Tasks = append(run.Tasks, input.DecisionPayload.FollowupRefs...)
	run.Agents = append(run.Agents, "agent-ref-app-director-review-rework-unrelated-live")
	run.ReplanDecisions = []string{
		input.DecisionPayload.ReplanRef + "#source:" + input.DecisionPayload.SourceRef +
			"#task:" + input.DecisionPayload.TaskRef +
			"#action:" + string(input.DecisionPayload.AcceptedAction) +
			"#followups:" + strings.Join(input.DecisionPayload.FollowupRefs, "+"),
	}
	events := append([]orquestacoreworkflow.OrchestrationEventV0(nil), fixture.Events[:4]...)
	events = append(events, serviceOperationalDirectorPlanStateEventForTestV0(
		t,
		fixture.RunRef,
		5,
		orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0,
		orquestacoreworkflow.ReplanDecisionRecordedPayloadV0(input.DecisionPayload),
	))
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-23T09:10:01Z",
		CorrelationID:              "corr-app-director-review-rework-durable-split",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: events},
		DirectorTaskStore:          taskStore,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if reentered.WaitParentTaskRef != fixture.TaskRef ||
		reentered.WaitWaveRef != firstFollowup.WaveRef ||
		reentered.WaitCohortRef != firstFollowup.CohortRef ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, "agent-ref-app-director-review-rework-unrelated-live") {
		t.Fatalf("reentered=%+v followups=%+v old=%s", reentered, followupAgentRefs, fixture.AgentRef)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveParentTaskRef != fixture.TaskRef ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		waitStep.Reason != "review-rework-replan-followups-waiting" ||
		!serviceStringInSetV0(waitStep.TaskRefs, firstFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.TaskRefs, secondFollowup.TaskID) {
		t.Fatalf("state=%+v waitStep=%+v", state, waitStep)
	}
}
