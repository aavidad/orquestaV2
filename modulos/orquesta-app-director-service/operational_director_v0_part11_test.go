package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewChangesRequestedAbreWaitDeRetryAgent(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	followupAgentRef := "agent-ref-app-director-operational-plan-state-review-retry-followup"
	capacityRef := "capacity-ref-app-director-operational-plan-state-review-retry-followup"
	fixture.Run.CapacityRequests = append(fixture.Run.CapacityRequests, capacityRef)
	fixture.Run.Agents = append(fixture.Run.Agents, followupAgentRef)
	fixture.Run.ReplanDecisions = []string{
		fixture.ReplanDecisionRef + "#source:" + fixture.ReworkRequestRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + capacityRef + "+" + followupAgentRef,
	}
	fixture.Events[4] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
		ReplanRef:      fixture.ReplanDecisionRef,
		RunRef:         fixture.RunRef,
		TaskRef:        fixture.TaskRef,
		SourceRef:      fixture.ReworkRequestRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
		FollowupRefs:   []string{capacityRef, followupAgentRef},
		Summary:        "Replan retry por review negativa.",
		EvidenceRefs:   []string{"evidence-ref-replan-decision-retry-followup"},
	})
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T15:10:00Z",
		CorrelationID:              "corr-app-director-review-retry-followup",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	request.OccurredAt = "2026-05-21T15:10:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveWaveRef != "" ||
		state.ActiveCohortRef != "" ||
		state.ActiveParentTaskRef != "" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followup-agents-v0") {
		t.Fatalf("state=%+v", state)
	}
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
		!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "review-rework-replan-followup-agents-waiting" ||
		!serviceStringInSetV0(waitStep.BlockerRefs, "wait-subagents-replan-followup-agents") ||
		!serviceStringInSetV0(waitStep.TaskRefs, fixture.TaskRef) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if reentered.WaitWaveRef != "" ||
		reentered.WaitCohortRef != "" ||
		reentered.WaitParentTaskRef != "" ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	deliveredRetryRun := fixture.Run
	deliveredRetryRun.DeliveredAgents = append(deliveredRetryRun.DeliveredAgents, followupAgentRef)
	request.OccurredAt = "2026-05-21T15:10:02Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    deliveredRetryRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 delivered retry: %v", err)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 delivered retry: %v", err)
	}
	reviewStep = serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-review-deliveries" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.TaskRefs, fixture.TaskRef) ||
		!serviceStringInSetV0(reviewStep.AgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reviewStep.AgentRefs, fixture.AgentRef) ||
		len(reviewStep.ReviewResultRefs) != 0 ||
		len(reviewStep.ReworkRequestRefs) != 0 ||
		len(reviewStep.ReplanDecisionRefs) != 0 {
		t.Fatalf("state=%+v reviewStep=%+v", state, reviewStep)
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
		!serviceStringInSetV0(testsStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(replanStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) ||
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
		!serviceStringInSetV0(testsStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) ||
		len(testsStep.RequiredTestEvidenceRefs) != 1 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(replanStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) ||
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
