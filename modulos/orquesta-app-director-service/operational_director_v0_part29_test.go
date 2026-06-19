package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestUpdateOperationalDirectorPlanStateAfterLoopV0EmparejaReviewAceptadaConResultadoPorEvidencia(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	staleReviewResultRef := "review-result-ref-app-director-operational-plan-state-review-stale"
	currentReviewResultRef := "review-result-ref-app-director-operational-plan-state-review-current"
	currentAcceptedReviewRef := "accepted-review-ref-app-director-operational-plan-state-review-current"
	sharedEvidenceRef := "evidence-ref-app-director-operational-plan-state-review-current"
	fixture.ReviewResultRef = currentReviewResultRef
	fixture.AcceptedReviewRef = currentAcceptedReviewRef
	fixture.Run.ReviewResults = []string{
		staleReviewResultRef + "#review_result:accepted#review_request:" + fixture.ReviewRequestID + "#delivery:" + fixture.DeliveryRef,
		currentReviewResultRef + "#review_result:accepted#review_request:" + fixture.ReviewRequestID + "#delivery:" + fixture.DeliveryRef,
	}
	fixture.Run.AcceptedReviews = []string{currentAcceptedReviewRef}
	fixture.Events = []orquestacoreworkflow.OrchestrationEventV0{
		fixture.Events[0],
		fixture.Events[1],
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: staleReviewResultRef,
			ReviewRequestID: fixture.ReviewRequestID,
			DeliveryRef:     fixture.DeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Resultado aceptado antiguo de un ciclo previo.",
			EvidenceRefs:    []string{"evidence-ref-review-result-stale"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 4, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: currentReviewResultRef,
			ReviewRequestID: fixture.ReviewRequestID,
			DeliveryRef:     fixture.DeliveryRef,
			Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
			Summary:         "Resultado aceptado vigente.",
			EvidenceRefs:    []string{sharedEvidenceRef},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReviewAcceptedV0, orquestacoreworkflow.ReviewAcceptedPayloadV0{
			AcceptedReviewRef: currentAcceptedReviewRef,
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseRevisionV0),
			ReviewRequestID:   fixture.ReviewRequestID,
			DeliveryRef:       fixture.DeliveryRef,
			Summary:           "Review aceptada vigente por el director.",
			EvidenceRefs:      []string{sharedEvidenceRef, fixture.RequiredTestEvidenceRef},
		}),
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		),
	)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-06-19T14:40:00Z",
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
		!serviceStringInSetV0(testsStep.ReviewResultRefs, currentReviewResultRef) ||
		serviceStringInSetV0(testsStep.ReviewResultRefs, staleReviewResultRef) ||
		!serviceStringInSetV0(testsStep.AcceptedReviewRefs, currentAcceptedReviewRef) ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(replanStep.ReviewResultRefs, currentReviewResultRef) ||
		serviceStringInSetV0(replanStep.ReviewResultRefs, staleReviewResultRef) {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}
}
