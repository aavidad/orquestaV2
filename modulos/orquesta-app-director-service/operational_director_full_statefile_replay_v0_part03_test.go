package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueAppDirectorV0StateFileReplayReviewChangesRequestedReworkReplanNoDuplica(t *testing.T) {
	ctx := context.Background()
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	rootDir := t.TempDir()
	store := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(ctx, fixture.Run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, fixture.State); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-23T00:20:00Z",
		CorrelationID:              "corr-service-review-negative-statefile-replay-first",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(store), orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	serviceAssertFullReplayReviewReworkReplanCountsV0(t, store, fixture, 1, 1, 1)

	recovered := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	recoveredRun, err := recovered.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 recovered: %v", err)
	}
	request.OccurredAt = "2026-05-23T00:20:01Z"
	request.CorrelationID = "corr-service-review-negative-statefile-replay-second"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(recovered), orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    recoveredRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}
	serviceAssertFullReplayReviewReworkReplanCountsV0(t, recovered, fixture, 1, 1, 1)

	recoveredAgain := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	request.OccurredAt = "2026-05-23T00:20:02Z"
	request.CorrelationID = "corr-service-review-negative-statefile-replay-third"
	_, err = continueRequestWithOperationalDirectorPlanStateV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(recoveredAgain))
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
	serviceAssertFullReplayReviewReworkReplanCountsV0(t, recoveredAgain, fixture, 1, 1, 1)
}

func TestContinueAppDirectorV0StateFileReplayReviewChangesRequestedReplaceAgentEsperaSoloNuevo(t *testing.T) {
	ctx := context.Background()
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	newAgentRef := "agent-ref-service-statefile-review-replace-new"
	capacityRef := "capacity-ref-service-statefile-review-replace-new"
	unrelatedAgentRef := "agent-ref-service-statefile-review-replace-unrelated"
	fixture.Run.CapacityRequests = append(fixture.Run.CapacityRequests, capacityRef)
	fixture.Run.Agents = append(fixture.Run.Agents, newAgentRef, unrelatedAgentRef)
	fixture.Run.ReplanDecisions = []string{
		fixture.ReplanDecisionRef + "#source:" + fixture.ReworkRequestRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0) +
			"#followups:" + capacityRef + "+" + newAgentRef,
	}
	fixture.Events[4] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
		ReplanRef:      fixture.ReplanDecisionRef,
		RunRef:         fixture.RunRef,
		TaskRef:        fixture.TaskRef,
		SourceRef:      fixture.ReworkRequestRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0,
		FollowupRefs:   []string{capacityRef, newAgentRef},
		Summary:        "Reemplazar agente tras review negativa.",
		EvidenceRefs:   []string{"evidence-ref-replan-decision-replace-agent-statefile"},
	})

	rootDir := t.TempDir()
	store := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(ctx, fixture.Run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, fixture.State); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-23T11:20:00Z",
		CorrelationID:              "corr-service-review-replace-agent-statefile-first",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(store), orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	serviceAssertFullReplayReviewReplaceAgentWaitV0(t, store, fixture, newAgentRef, unrelatedAgentRef)

	recovered := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	recoveredRun, err := recovered.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 recovered: %v", err)
	}
	request.OccurredAt = "2026-05-23T11:20:01Z"
	request.CorrelationID = "corr-service-review-replace-agent-statefile-replay-update"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(recovered), orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    recoveredRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}
	serviceAssertFullReplayReviewReplaceAgentWaitV0(t, recovered, fixture, newAgentRef, unrelatedAgentRef)

	recoveredAgain := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	request.OccurredAt = "2026-05-23T11:20:02Z"
	request.CorrelationID = "corr-service-review-replace-agent-statefile-replay-wait"
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(recoveredAgain))
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 replay: %v", err)
	}
	if len(reentered.WaitAgentRefs) != 1 ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, newAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, unrelatedAgentRef) ||
		reentered.WaitWaveRef != "" ||
		reentered.WaitCohortRef != "" ||
		reentered.WaitParentTaskRef != "" {
		t.Fatalf("reentered=%+v new=%s old=%s unrelated=%s", reentered, newAgentRef, fixture.AgentRef, unrelatedAgentRef)
	}
	serviceAssertFullReplayReviewReplaceAgentWaitV0(t, recoveredAgain, fixture, newAgentRef, unrelatedAgentRef)
}
