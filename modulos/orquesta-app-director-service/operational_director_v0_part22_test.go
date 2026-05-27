package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueRequestWithOperationalDirectorPlanStateV0ReabreReviewChangesRequestedConFollowupTardio(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	followupAgentRef := "agent-ref-app-director-operational-plan-state-review-late-retry-followup"
	capacityRef := "capacity-ref-app-director-operational-plan-state-review-late-retry-followup"
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
		Summary:        "Replan retry por review negativa con agente tardio.",
		EvidenceRefs:   []string{"evidence-ref-replan-decision-late-retry-followup"},
	})
	fixture.State.ReplanAttempts = 1
	fixture.State.EvidenceRefs = []string{"evidence-ref-app-director-operational-plan-state-review-rework-replan-v0"}
	for index := range fixture.State.Steps {
		step := &fixture.State.Steps[index]
		if step.StepID != "step-review-deliveries" {
			continue
		}
		step.Status = orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0
		step.DeliveryRefs = []string{fixture.DeliveryRef}
		step.ReviewResultRefs = []string{fixture.ReviewResultRef}
		step.ReworkRequestRefs = []string{fixture.ReworkRequestRef}
		step.ReplanDecisionRefs = []string{fixture.ReplanDecisionRef}
		step.BlockerRefs = []string{"review-rework-replan-recorded"}
		step.Reason = "review-rework-replan-recorded"
	}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T18:15:00Z",
		CorrelationID:              "corr-app-director-review-late-followup",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
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

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followups-late-v0") ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followup-agents-v0") {
		t.Fatalf("state=%+v", state)
	}
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		reviewStep.Reason != "review-rework-replan-recorded" ||
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

	replayed, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T18:15:01Z",
		CorrelationID:              "corr-app-director-review-late-followup",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 replay: %v", err)
	}
	if !serviceStringInSetV0(replayed.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(replayed.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("replayed=%+v followup=%s old=%s", replayed, followupAgentRef, fixture.AgentRef)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 replay: %v", err)
	}
	if state.ReplanAttempts != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followups-late-v0") != 1 {
		t.Fatalf("state replay=%+v", state)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReentraRunRequiredTestsConScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	for index := range fixture.State.Steps {
		step := &fixture.State.Steps[index]
		switch step.StepID {
		case "step-review-deliveries":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepAcceptedV0
			step.DeliveryRefs = []string{fixture.DeliveryRef}
			step.ReviewResultRefs = []string{fixture.ReviewResultRef}
			step.AcceptedReviewRefs = []string{fixture.AcceptedReviewRef}
		case "step-run-required-tests":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			step.TaskRefs = []string{fixture.TaskRef}
			step.AgentRefs = []string{fixture.AgentRef}
			step.DeliveryRefs = []string{fixture.DeliveryRef}
			step.ReviewResultRefs = []string{fixture.ReviewResultRef}
			step.BlockerRefs = []string{"required-tests-pending"}
		}
	}
	fixture.State.ActiveStepID = "step-run-required-tests"
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if got.WaitWaveRef != fixture.WaveRef ||
		got.WaitCohortRef != fixture.CohortRef ||
		got.WaitParentTaskRef != fixture.ParentTaskRef ||
		!serviceStringInSetV0(got.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("request=%+v fixture=%+v", got, fixture)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RechazaActiveStepNoSoportado(t *testing.T) {
	runRef := "run-app-director-operational-plan-reentry-unsupported"
	planRef := "plan-ref-reentry-unsupported"
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:   orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:        "state-ref-reentry-unsupported",
		PlanRef:         planRef,
		RequestRef:      "request-ref-reentry-unsupported",
		RunRef:          runRef,
		ProjectRef:      "orquesta",
		Mode:            orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:          orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:    "step-replan-or-close",
		ActiveWaveRef:   "wave-reentry-unsupported",
		ActiveCohortRef: "cohort-reentry-unsupported",
		ObservedAt:      "2026-05-17T13:58:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
			StepID:    "step-replan-or-close",
			Kind:      orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
			Status:    orquestadirectoroperativo.OperationalDirectorStepRunningV0,
			WaveRef:   "wave-reentry-unsupported",
			CohortRef: "cohort-reentry-unsupported",
		}},
	}
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	_, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0RespetaWaitExplicito(t *testing.T) {
	request := ContinueAppDirectorRequestV0{
		RunRef:                     "run-app-director-operational-plan-explicit-wait",
		OperationalDirectorPlanRef: "plan-ref-explicit-wait",
		WaitAgentRefs:              []string{"agent-ref-explicit"},
	}
	got, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, StartAppDirectorPortsV0{})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if len(got.WaitAgentRefs) != 1 || got.WaitAgentRefs[0] != "agent-ref-explicit" {
		t.Fatalf("request=%+v", got)
	}
}
