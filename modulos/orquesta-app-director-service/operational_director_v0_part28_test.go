package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestUpdateOperationalDirectorPlanStateAfterLoopV0RetryFollowupDeliveredAcceptedAvanzaARequiredTests(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	followupAgentRef := "agent-ref-app-director-operational-plan-state-review-retry-accepted"
	followupDeliveryRef := "delivery-ref-app-director-operational-plan-state-review-retry-accepted"
	followupReviewRequestRef := "review-request-ref-app-director-operational-plan-state-review-retry-accepted"
	followupReviewResultRef := "review-result-ref-app-director-operational-plan-state-review-retry-accepted"
	followupAcceptedReviewRef := "accepted-review-ref-app-director-operational-plan-state-review-retry-accepted"
	capacityRef := "capacity-ref-app-director-operational-plan-state-review-retry-accepted"
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
		EvidenceRefs:   []string{"evidence-ref-replan-decision-retry-accepted"},
	})
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-27T21:00:00Z",
		CorrelationID:              "corr-app-director-review-retry-accepted",
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
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 changes requested: %v", err)
	}

	deliveredRetryRun := fixture.Run
	deliveredRetryRun.DeliveredAgents = compactServiceRefsV0(append(deliveredRetryRun.DeliveredAgents, followupAgentRef))
	deliveredRetryRun.Deliveries = compactServiceRefsV0(append(deliveredRetryRun.Deliveries, followupDeliveryRef))
	deliveredRetryRun.DeliveredTasks = compactServiceRefsV0(append(deliveredRetryRun.DeliveredTasks, fixture.TaskRef))
	deliveredRetryRun.Reviews = compactServiceRefsV0(append(deliveredRetryRun.Reviews, followupReviewRequestRef))
	deliveredRetryRun.ReviewResults = compactServiceRefsV0(append(deliveredRetryRun.ReviewResults,
		followupReviewResultRef+"#review_result:accepted#review_request:"+followupReviewRequestRef+"#delivery:"+followupDeliveryRef,
	))
	deliveredRetryRun.AcceptedReviews = compactServiceRefsV0(append(deliveredRetryRun.AcceptedReviews, followupAcceptedReviewRef))
	ports.EventReader = serviceOperationalClosureEventReaderForTestV0{Events: append(fixture.Events,
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
			DeliveryRef:  followupDeliveryRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskID:       fixture.TaskRef,
			AgentRef:     followupAgentRef,
			Summary:      "Entrega followup causal tras retry.",
			EvidenceRefs: []string{"evidence-ref-delivery-retry-accepted"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 7, orquestacoreworkflow.OrchestrationEventReviewRequestedV0, orquestacoreworkflow.ReviewRequestedPayloadV0{
			ReviewRequestID: followupReviewRequestRef,
			PhaseID:         string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			DeliveryRef:     followupDeliveryRef,
			Summary:         "Review followup causal.",
			EvidenceRefs:    []string{"evidence-ref-review-requested-retry-accepted"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 8, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: followupReviewResultRef,
			ReviewRequestID: followupReviewRequestRef,
			DeliveryRef:     followupDeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Review followup aceptada.",
			EvidenceRefs:    []string{"evidence-ref-review-result-retry-accepted", fixture.RequiredTestEvidenceRef},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 9, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: followupAcceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   followupReviewRequestRef,
			DeliveryRef:       followupDeliveryRef,
			Summary:           "Review followup aceptada con cadena causal.",
			EvidenceRefs:      []string{"evidence-ref-review-accepted-retry-accepted"},
		}),
	)}
	request.OccurredAt = "2026-05-27T21:00:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    deliveredRetryRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 followup accepted: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.ActiveStepID != "step-run-required-tests" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		!serviceStringInSetV0(reviewStep.AgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reviewStep.AgentRefs, fixture.AgentRef) ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, followupDeliveryRef) ||
		!serviceStringInSetV0(reviewStep.AcceptedReviewRefs, followupAcceptedReviewRef) ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(testsStep.AgentRefs, followupAgentRef) ||
		serviceStringInSetV0(testsStep.AgentRefs, fixture.AgentRef) ||
		!serviceStringInSetV0(testsStep.DeliveryRefs, followupDeliveryRef) ||
		!serviceStringInSetV0(testsStep.ReviewResultRefs, followupReviewResultRef) {
		t.Fatalf("state=%+v waitStep=%+v reviewStep=%+v testsStep=%+v", state, waitStep, reviewStep, testsStep)
	}
}
