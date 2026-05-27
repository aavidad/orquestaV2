package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestUpdateOperationalDirectorPlanStateAfterLoopV0NoAvanzaReviewConOutboxPendiente(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:23:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status:             orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		PendingOutboxCount: 1,
		Run:                fixture.Run,
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

func TestContinueAppDirectorV0PlanStateReentradaRequiereStore(t *testing.T) {
	runRef := "run-app-director-operational-plan-reentry-missing-store"
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(serviceRunForWaitRefsTestV0(runRef))
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	_, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-17T13:57:00Z",
		CorrelationID:              "corr-app-director-operational-plan-reentry-missing-store",
		OperationalDirectorPlanRef: "plan-ref-missing-store",
	}, StartAppDirectorPortsV0{
		RunStore:     store,
		EventSink:    sink,
		OutboxLedger: ledger,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "ports.operational_plan_state_store" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReentraReviewConScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
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

func TestContinueRequestWithOperationalDirectorPlanStateV0NoReentraReviewChangesRequested(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	fixture.State.ReplanAttempts = 1
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
	store := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	_, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: store})
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReabreReviewChangesRequestedConSplitTaskSinScope(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
	followupTask := serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)
	followupTask.TaskID = "task-ref-app-director-operational-plan-state-review-split-no-scope"
	followupTask.ParentTaskRef = ""
	followupTask.WaveRef = ""
	followupTask.CohortRef = ""
	followupAgentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(followupTask.TaskID)
	fixture.Run.Tasks = append(fixture.Run.Tasks, followupTask.TaskID)
	fixture.Run.Agents = append(fixture.Run.Agents, "agent-ref-app-director-review-unrelated-live")
	fixture.Run.ReplanDecisions = []string{
		fixture.ReplanDecisionRef + "#source:" + fixture.ReworkRequestRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0) +
			"#followups:" + followupTask.TaskID,
	}
	fixture.Events[4] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
		ReplanRef:      fixture.ReplanDecisionRef,
		RunRef:         fixture.RunRef,
		TaskRef:        fixture.TaskRef,
		SourceRef:      fixture.ReworkRequestRef,
		AcceptedAction: orquestacoreworkflow.ReplanDecisionActionSplitTaskV0,
		FollowupRefs:   []string{followupTask.TaskID},
		Summary:        "Replan split sin scope; debe esperar por refs explicitas.",
		EvidenceRefs:   []string{"evidence-ref-replan-decision-split-no-scope"},
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
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
		serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture),
		followupTask,
	)

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T18:20:00Z",
		CorrelationID:              "corr-app-director-review-split-no-scope",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		DirectorTaskStore:          taskStore,
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
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, "agent-ref-app-director-review-unrelated-live") {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveWaveRef != "" ||
		state.ActiveCohortRef != "" ||
		state.ActiveParentTaskRef != "" ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followups-late-v0") ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followups-v0") {
		t.Fatalf("state=%+v", state)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "review-rework-replan-followups-waiting" ||
		!serviceStringInSetV0(waitStep.BlockerRefs, "wait-subagents-replan-followups") ||
		!serviceStringInSetV0(waitStep.TaskRefs, followupTask.TaskID) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}
}
