package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	"testing"
)

func serviceAssertFullReplayReviewReplaceAgentWaitV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
	newAgentRef string,
	unrelatedAgentRef string,
) {
	t.Helper()
	events := serviceFullReplayEventsForTestV0(t, store, fixture.RunRef)
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventReworkRequestedV0); got != 1 {
		t.Fatalf("ReworkRequested got=%d want=1 events=%+v", got, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != 1 {
		t.Fatalf("ReplanDecisionRecorded got=%d want=1 events=%+v", got, events)
	}
	run, err := store.LoadRunV0(context.Background(), fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.ReworkRequests) != 1 ||
		len(run.ReplanDecisions) != 1 ||
		serviceCountStringV0(run.ReworkRequests, run.ReworkRequests[0]) != 1 ||
		serviceCountStringV0(run.ReplanDecisions, run.ReplanDecisions[0]) != 1 {
		t.Fatalf("run rework/replan counts invalidos: %+v", run)
	}
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
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
		len(state.PendingAgentRefs) != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, newAgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		serviceStringInSetV0(state.PendingAgentRefs, unrelatedAgentRef) ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-v0") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-followup-agents-v0") != 1 {
		t.Fatalf("state=%+v", state)
	}
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		reviewStep.Reason != "review-rework-replan-recorded" ||
		serviceCountStringV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) != 1 ||
		serviceCountStringV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) != 1 {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	if waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "review-rework-replan-followup-agents-waiting" ||
		!serviceStringInSetV0(waitStep.BlockerRefs, "wait-subagents-replan-followup-agents") ||
		len(waitStep.TaskRefs) != 1 ||
		waitStep.TaskRefs[0] != fixture.TaskRef ||
		len(waitStep.PendingAgentRefs) != 1 ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, newAgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, unrelatedAgentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}
}

func serviceAssertFullReplayReviewSplitCountsV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
	wantRework int,
	wantReplan int,
	wantMicrotasks int,
	wantAgents int,
) {
	t.Helper()
	events := serviceFullReplayEventsForTestV0(t, store, fixture.RunRef)
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventReworkRequestedV0); got != wantRework {
		t.Fatalf("ReworkRequested got=%d want=%d events=%+v", got, wantRework, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != wantReplan {
		t.Fatalf("ReplanDecisionRecorded got=%d want=%d events=%+v", got, wantReplan, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0); got != wantMicrotasks {
		t.Fatalf("MicrotaskCreated got=%d want=%d events=%+v", got, wantMicrotasks, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventAgentStartedV0); got != wantAgents {
		t.Fatalf("AgentStarted got=%d want=%d events=%+v", got, wantAgents, events)
	}
	run, err := store.LoadRunV0(context.Background(), fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.ReworkRequests) != wantRework ||
		len(run.ReplanDecisions) != wantReplan ||
		len(run.Tasks) != 1+wantMicrotasks ||
		len(run.StartedAgents) != 1+wantAgents {
		t.Fatalf("run split counts invalidos: %+v", run)
	}
}

func serviceAssertFullReplayReviewSplitWaitStateV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
	firstFollowup orquestacoreworkflow.WorkflowTaskV0,
	secondFollowup orquestacoreworkflow.WorkflowTaskV0,
	followupAgentRefs []string,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	t.Helper()
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveParentTaskRef != fixture.TaskRef ||
		state.ActiveWaveRef != firstFollowup.WaveRef ||
		state.ActiveCohortRef != firstFollowup.CohortRef ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		waitStep.Reason != "review-rework-replan-followups-waiting" ||
		!serviceStringInSetV0(waitStep.TaskRefs, firstFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.TaskRefs, secondFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRefs[1]) ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		!serviceStringInSetV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) ||
		!serviceStringInSetV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) {
		t.Fatalf("state=%+v waitStep=%+v reviewStep=%+v", state, waitStep, reviewStep)
	}
	return state
}

func serviceFullReplayEventsForTestV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	runRef string,
) []orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	events, err := store.LoadRunEventsV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunEventsV0: %v", err)
	}
	return events
}

func serviceAssertFullReplayClosureCountsV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	runRef string,
	planRef string,
	wantTaskClosed int,
	wantValidation int,
	wantRunClosed int,
	wantEvidence int,
) {
	t.Helper()
	events := serviceFullReplayEventsForTestV0(t, store, runRef)
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventTaskClosedV0); got != wantTaskClosed {
		t.Fatalf("TaskClosed got=%d want=%d events=%+v", got, wantTaskClosed, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0); got != wantValidation {
		t.Fatalf("FinalValidationRegistered got=%d want=%d events=%+v", got, wantValidation, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventRunClosedV0); got != wantRunClosed {
		t.Fatalf("RunClosed got=%d want=%d events=%+v", got, wantRunClosed, events)
	}
	run, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.ClosedTasks) != wantTaskClosed || len(run.Validations) != wantValidation || len(run.Closures) != wantRunClosed {
		t.Fatalf("run cierre counts invalidos: %+v", run)
	}
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if len(testsStep.RequiredTestEvidenceRefs) != wantEvidence {
		t.Fatalf("plan evidence refs got=%d want=%d state=%+v testsStep=%+v", len(testsStep.RequiredTestEvidenceRefs), wantEvidence, state, testsStep)
	}
	evidence, err := store.LoadRequiredTestEvidenceV0(context.Background(), runRef, testsStep.RequiredTestEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != wantEvidence {
		t.Fatalf("required test evidence got=%d want=%d evidence=%+v", len(evidence), wantEvidence, evidence)
	}
}
