package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	"testing"
)

func serviceAssertFullReplayReplanCountsV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	runRef string,
	planRef string,
	wantGate int,
	wantReplan int,
	wantTaskClosed int,
	wantRunClosed int,
	wantReplanAttempts int,
) {
	t.Helper()
	events := serviceFullReplayEventsForTestV0(t, store, runRef)
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != wantGate {
		t.Fatalf("QualityGateRecorded got=%d want=%d events=%+v", got, wantGate, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != wantReplan {
		t.Fatalf("ReplanDecisionRecorded got=%d want=%d events=%+v", got, wantReplan, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventTaskClosedV0); got != wantTaskClosed {
		t.Fatalf("TaskClosed got=%d want=%d events=%+v", got, wantTaskClosed, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventRunClosedV0); got != wantRunClosed {
		t.Fatalf("RunClosed got=%d want=%d events=%+v", got, wantRunClosed, events)
	}
	run, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.QualityGates) != wantGate || len(run.ReplanDecisions) != wantReplan ||
		len(run.ClosedTasks) != wantTaskClosed || len(run.Closures) != wantRunClosed {
		t.Fatalf("run counts invalidos: %+v", run)
	}
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ReplanAttempts != wantReplanAttempts {
		t.Fatalf("ReplanAttempts got=%d want=%d state=%+v", state.ReplanAttempts, wantReplanAttempts, state)
	}
	if wantReplanAttempts > 0 {
		waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
		if state.ActiveStepID != "step-wait-subagents" ||
			waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
			len(waitStep.WaitRefs) != 1 ||
			len(waitStep.PendingAgentRefs) != 1 {
			t.Fatalf("wait replay invalido: state=%+v waitStep=%+v", state, waitStep)
		}
	}
}

func serviceFullReplayStateFileSplitPortsForTestV0(
	store *orquestastatefile.StoreV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	reviewSource orquestacionnucleoapp.ReviewGateObservationProviderPortV0,
	replanSource orquestacionnucleoapp.ReviewReworkReplanPlanProviderPortV0,
) StartAppDirectorPortsV0 {
	return StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		EventReader:                store,
		OutboxLedger:               ledger,
		DirectorTaskStore:          store,
		WaitStateWriter:            store,
		WaitStateStore:             store,
		ReviewGateSource:           reviewSource,
		ReviewReworkReplanSource:   replanSource,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		Dispatchers:                serviceFullReplayStateFileSplitDispatchersForTestV0(store, ledger),
	}
}

func serviceFullReplayStateFileSplitDispatchersForTestV0(
	store *orquestastatefile.StoreV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) []orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return []orquestacionnucleoapp.OutboxDispatcherBindingV0{
		{
			TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
			Reader:     ledger,
			Claimer:    ledger,
			Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
				RunStore:        store,
				EventSink:       store,
				Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
				ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityHighV0,
				OccurredAt:      "2026-05-23T10:31:00Z",
				CorrelationID:   "corr-service-statefile-split-capacity",
				RequestedBy:     "orquesta-app-director-service-test",
			},
			Acker: ledger,
		},
		{
			TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			Reader:     ledger,
			Claimer:    ledger,
			Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
				RunStore:      store,
				EventSink:     store,
				Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
				OccurredAt:    "2026-05-23T10:32:00Z",
				CorrelationID: "corr-service-statefile-split-agent",
				RequestedBy:   "orquesta-app-director-service-test",
			},
			Acker: ledger,
		},
	}
}

func serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(
	store *orquestastatefile.StoreV0,
) StartAppDirectorPortsV0 {
	return StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		EventReader:                store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		RequiredTestEvidenceStore:  store,
	}
}

func serviceAssertFullReplayRequiredTestsReplanCountsV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	runRef string,
	planRef string,
	wantGate int,
	wantReplan int,
	wantPhaseOpened int,
	wantReplanAttempts int,
) {
	t.Helper()
	events := serviceFullReplayEventsForTestV0(t, store, runRef)
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0); got != wantGate {
		t.Fatalf("QualityGateRecorded got=%d want=%d events=%+v", got, wantGate, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != wantReplan {
		t.Fatalf("ReplanDecisionRecorded got=%d want=%d events=%+v", got, wantReplan, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventPhaseOpenedV0); got != wantPhaseOpened {
		t.Fatalf("PhaseOpened got=%d want=%d events=%+v", got, wantPhaseOpened, events)
	}
	run, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.QualityGates) != wantGate || len(run.ReplanDecisions) != wantReplan {
		t.Fatalf("run replan counts invalidos: %+v", run)
	}
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ReplanAttempts != wantReplanAttempts {
		t.Fatalf("ReplanAttempts got=%d want=%d state=%+v", state.ReplanAttempts, wantReplanAttempts, state)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		testsStep.Reason != "required-tests-failed" ||
		len(testsStep.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("state replan replay invalido: state=%+v testsStep=%+v", state, testsStep)
	}
}

func serviceAssertFullReplayReviewReworkReplanCountsV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
	wantRework int,
	wantReplan int,
	wantReplanAttempts int,
) {
	t.Helper()
	events := serviceFullReplayEventsForTestV0(t, store, fixture.RunRef)
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventReworkRequestedV0); got != wantRework {
		t.Fatalf("ReworkRequested got=%d want=%d events=%+v", got, wantRework, events)
	}
	if got := serviceCountEventsByTypeV0(events, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0); got != wantReplan {
		t.Fatalf("ReplanDecisionRecorded got=%d want=%d events=%+v", got, wantReplan, events)
	}
	run, err := store.LoadRunV0(context.Background(), fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.ReworkRequests) != wantRework ||
		len(run.ReplanDecisions) != wantReplan {
		t.Fatalf("run rework/replan counts invalidos: %+v", run)
	}
	if wantRework > 0 && serviceCountStringV0(run.ReworkRequests, run.ReworkRequests[0]) != 1 {
		t.Fatalf("run rework refs invalidas: %+v", run.ReworkRequests)
	}
	if wantReplan > 0 && serviceCountStringV0(run.ReplanDecisions, run.ReplanDecisions[0]) != 1 {
		t.Fatalf("run replan refs invalidas: %+v", run.ReplanDecisions)
	}
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-review-deliveries" ||
		state.ActiveWaveRef != fixture.WaveRef ||
		state.ActiveCohortRef != fixture.CohortRef ||
		state.ActiveParentTaskRef != fixture.ParentTaskRef ||
		len(state.PendingAgentRefs) != 0 ||
		state.ReplanAttempts != wantReplanAttempts ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-review-rework-replan-v0") != 1 {
		t.Fatalf("state=%+v", state)
	}
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepChangesRequestedV0 ||
		reviewStep.Reason != "review-rework-replan-recorded" ||
		len(reviewStep.TaskRefs) != 1 ||
		reviewStep.TaskRefs[0] != fixture.TaskRef ||
		len(reviewStep.AgentRefs) != 1 ||
		reviewStep.AgentRefs[0] != fixture.AgentRef ||
		len(reviewStep.PendingAgentRefs) != 0 ||
		len(reviewStep.ReworkRequestRefs) != wantRework ||
		reviewStep.ReworkRequestRefs[0] != fixture.ReworkRequestRef ||
		len(reviewStep.ReplanDecisionRefs) != wantReplan ||
		reviewStep.ReplanDecisionRefs[0] != fixture.ReplanDecisionRef ||
		serviceCountStringV0(reviewStep.ReworkRequestRefs, fixture.ReworkRequestRef) != 1 ||
		serviceCountStringV0(reviewStep.ReplanDecisionRefs, fixture.ReplanDecisionRef) != 1 ||
		len(reviewStep.AcceptedReviewRefs) != 0 {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
}
