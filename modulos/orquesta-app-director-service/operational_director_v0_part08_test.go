package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewAceptadaReplayNoDuplicaPlanState(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}
	for _, occurredAt := range []string{"2026-05-17T14:20:00Z", "2026-05-17T14:20:01Z"} {
		if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OccurredAt:                 occurredAt,
			OperationalDirectorPlanRef: fixture.PlanRef,
		}, ports, loop); err != nil {
			t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 %s: %v", occurredAt, err)
		}
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.ActiveStepID != "step-run-required-tests" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 {
		t.Fatalf("state=%+v reviewStep=%+v testsStep=%+v", state, reviewStep, testsStep)
	}
	for label, values := range map[string][]string{
		"delivery":        reviewStep.DeliveryRefs,
		"review_result":   reviewStep.ReviewResultRefs,
		"accepted_review": reviewStep.AcceptedReviewRefs,
	} {
		want := map[string]string{
			"delivery":        fixture.DeliveryRef,
			"review_result":   fixture.ReviewResultRef,
			"accepted_review": fixture.AcceptedReviewRef,
		}[label]
		if serviceCountStringV0(values, want) != 1 {
			t.Fatalf("%s refs=%+v want una vez %s", label, values, want)
		}
	}
	if serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-accepted-v0") != 1 ||
		serviceCountStringV0(testsStep.BlockerRefs, "required-tests-pending") != 1 ||
		serviceCountStringV0(testsStep.TaskRefs, fixture.TaskRef) != 1 {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewNegativaRegistraReworkReplan(t *testing.T) {
	for _, status := range []orquestacoreworkflow.ReviewResultStatusV0{
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
		orquestacoreworkflow.ReviewResultStatusRejectedV0,
	} {
		t.Run(string(status), func(t *testing.T) {
			fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(t, status)
			planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

			if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
				RunRef:                     fixture.RunRef,
				OccurredAt:                 "2026-05-17T14:20:10Z",
				OperationalDirectorPlanRef: fixture.PlanRef,
			}, StartAppDirectorPortsV0{
				EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
				OperationalPlanStateStore:  planStateStore,
				OperationalPlanStateWriter: planStateStore,
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
			reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
			testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
			replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
			if state.ActiveStepID != "step-review-deliveries" ||
				state.ReplanAttempts != 1 ||
				reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
				!serviceStringInSetV0(reviewStep.DeliveryRefs, fixture.DeliveryRef) ||
				!serviceStringInSetV0(reviewStep.ReviewResultRefs, fixture.ReviewResultRef) ||
				!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
				!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) ||
				len(reviewStep.AcceptedReviewRefs) != 0 ||
				testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepPendingV0 ||
				replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepPendingV0 {
				t.Fatalf("state=%+v reviewStep=%+v testsStep=%+v replanStep=%+v", state, reviewStep, testsStep, replanStep)
			}
		})
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0ReviewChangesRequestedIdempotenteYNoReentraWaitAntiguo(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:10Z",
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
	request.OccurredAt = "2026-05-17T14:20:11Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-review-deliveries" ||
		len(state.PendingAgentRefs) != 0 ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-v0") {
		t.Fatalf("state=%+v", state)
	}
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		reviewStep.Reason != "review-rework-replan-recorded" ||
		len(reviewStep.DeliveryRefs) != 1 ||
		reviewStep.DeliveryRefs[0] != fixture.DeliveryRef ||
		len(reviewStep.ReviewResultRefs) != 1 ||
		reviewStep.ReviewResultRefs[0] != fixture.ReviewResultRef ||
		len(reviewStep.ReworkRequestRefs) != 1 ||
		reviewStep.ReworkRequestRefs[0] != fixture.ReworkRequestRef ||
		len(reviewStep.ReplanDecisionRefs) != 1 ||
		reviewStep.ReplanDecisionRefs[0] != fixture.ReplanDecisionRef ||
		len(reviewStep.AcceptedReviewRefs) != 0 ||
		len(reviewStep.BlockerRefs) != 1 ||
		reviewStep.BlockerRefs[0] != "review-rework-replan-recorded" {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	for _, ref := range []string{
		"evidence-ref-delivery-review-accepted",
		"evidence-ref-review-requested",
		"evidence-ref-review-result-changes_requested",
		"evidence-ref-rework-requested-changes_requested",
		"evidence-ref-replan-decision-changes_requested",
	} {
		if !serviceStringInSetV0(reviewStep.EvidenceRefs, ref) {
			t.Fatalf("reviewStep evidence_refs=%+v falta %s", reviewStep.EvidenceRefs, ref)
		}
	}

	got, applied, err := continueRequestWithLoadedOperationalDirectorPlanStateV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OperationalDirectorPlanRef: fixture.PlanRef,
		},
		state,
		StartAppDirectorPortsV0{},
	)
	if err != nil {
		t.Fatalf("continueRequestWithLoadedOperationalDirectorPlanStateV0: %v", err)
	}
	if applied ||
		len(got.WaitAgentRefs) != 0 ||
		got.WaitWaveRef != "" ||
		got.WaitCohortRef != "" ||
		got.WaitParentTaskRef != "" {
		t.Fatalf("reentrada inesperada applied=%v request=%+v", applied, got)
	}
	_, err = continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
}
