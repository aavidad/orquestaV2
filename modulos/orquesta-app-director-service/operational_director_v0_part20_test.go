package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestUpdateOperationalDirectorPlanStateAfterLoopV0TestsFailedConQualityGateReplanRetryAbreWait(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	gateRef := "quality-gate-ref-app-director-required-tests-failed"
	replanRef := "replan-ref-app-director-required-tests-failed"
	capacityRef := "capacity-ref-app-director-required-tests-failed-retry"
	followupAgentRef := "agent-ref-app-director-required-tests-failed-retry"
	fixture.Run.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	fixture.Run.CapacityRequests = append(fixture.Run.CapacityRequests, capacityRef)
	fixture.Run.Agents = append(fixture.Run.Agents, followupAgentRef)
	fixture.Run.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + capacityRef + "+" + followupAgentRef,
	}
	fixture.Events = append(fixture.Events,
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:       fixture.RunRef,
			GateRef:      gateRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef:   fixture.TaskRef,
			Decision:     orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:    []string{fixture.RequiredTestEvidenceRef},
			Summary:      "Tests requeridos fallidos.",
			EvidenceRefs: []string{"evidence-ref-quality-gate-required-tests-failed"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{capacityRef, followupAgentRef},
			Summary:        "Replan por tests requeridos fallidos.",
			EvidenceRefs:   []string{"evidence-ref-replan-required-tests-failed"},
		}),
	)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T16:00:00Z",
		CorrelationID:              "corr-app-director-required-tests-failed-replan",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	request.OccurredAt = "2026-05-21T16:00:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-required-tests-failed-replan-v0") {
		t.Fatalf("state=%+v", state)
	}
	if testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed-replan-recorded" ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		!serviceStringInSetV0(testsStep.ReplanDecisionRefs, replanRef) ||
		!serviceStringInSetV0(testsStep.BlockerRefs, gateRef) {
		t.Fatalf("testsStep=%+v", testsStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "required-tests-failed-replan-followup-agents-waiting" ||
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
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	deliveredRetryRun := fixture.Run
	deliveredRetryRun.DeliveredAgents = append(deliveredRetryRun.DeliveredAgents, followupAgentRef)
	request.OccurredAt = "2026-05-21T16:00:02Z"
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
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-review-deliveries" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.TaskRefs, fixture.TaskRef) ||
		!serviceStringInSetV0(reviewStep.AgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reviewStep.AgentRefs, fixture.AgentRef) ||
		len(reviewStep.ReviewResultRefs) != 0 {
		t.Fatalf("state=%+v reviewStep=%+v", state, reviewStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaTestsConEvidenciaDeOtraReview(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	evidence := serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
		fixture,
		orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
	)
	evidence.ReviewResultRef = "review-result-ref-otra-review"
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(evidence)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:50Z",
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
	if state.ActiveStepID != "step-run-required-tests" ||
		state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		!serviceStringInSetV0(state.BlockerRefs, "required-tests-evidence-missing") ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-evidence-missing" ||
		len(testsStep.RequiredTestEvidenceRefs) != 0 {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeReviewAReplanSinTests(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, false)
	fixture.State.Mode = orquestadirectoroperativo.OperationalDirectorModeDomainWorkV0
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:21:00Z",
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
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.ActiveStepID != "step-replan-or-close" ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(replanStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) {
		t.Fatalf("state=%+v replanStep=%+v", state, replanStep)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaReviewFueraDeScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Events[0] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 1, orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0, orquestacoreworkflow.DeliveryRegisteredPayloadV0{
		DeliveryRef:  fixture.DeliveryRef,
		PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		TaskID:       "task-ref-review-out-of-scope",
		AgentRef:     "agent-ref-review-out-of-scope",
		Summary:      "Entrega fuera del scope activo.",
		EvidenceRefs: []string{"evidence-ref-review-out-of-scope"},
	})
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:22:00Z",
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
	if state.ActiveStepID != "step-review-deliveries" {
		t.Fatalf("state=%+v", state)
	}
}
