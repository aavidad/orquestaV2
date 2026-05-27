package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
	"testing"
)

func TestEnsureOperationalDirectorPassedRequiredTestsQualityGateV0ResuelveGateAntiguoStateFile(t *testing.T) {
	ctx := context.Background()
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	fixture.Run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusPendingV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
		{
			ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
			RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
		},
	}
	fixture.Run.LastSequence = 4
	fixture.Run.LastEventID = fixture.Events[len(fixture.Events)-1].EventID

	rootDir := t.TempDir()
	store := serviceRequiredTestsStateFileStoreForTestV0(t, rootDir)
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
		OccurredAt:                 "2026-05-22T21:05:00Z",
		CorrelationID:              "corr-service-required-tests-statefile-old-gate",
		RequestedBy:                "orquesta-app-director-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceRequiredTestsStateFilePortsV0(store), loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 missing: %v", err)
	}
	serviceAssertRequiredTestsMissingStateV0(t, store, fixture)
	serviceAssertRequiredTestsGateCountsV0(t, store, fixture.RunRef, 1, 0, 0)

	evidence := serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
		fixture,
		orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
	)
	if err := store.SaveRequiredTestEvidenceV0(ctx, evidence); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}
	blockedState, err := store.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	blockedTestsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, blockedState, "step-run-required-tests")
	passedState, changed, err := operationalDirectorPlanStateWithRequiredTestsPassedV0(
		request,
		blockedState,
		blockedTestsStep,
		[]string{fixture.RequiredTestEvidenceRef},
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStateWithRequiredTestsPassedV0: %v", err)
	}
	if !changed {
		t.Fatalf("expected passed state change")
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, passedState); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0 passed: %v", err)
	}
	serviceAssertRequiredTestsGateCountsV0(t, store, fixture.RunRef, 1, 0, 0)

	request.OccurredAt = "2026-05-22T21:05:01Z"
	request.CorrelationID = "corr-service-required-tests-statefile-old-gate-reentry"
	if err := ensureOperationalDirectorPassedRequiredTestsQualityGateV0(ctx, request, serviceRequiredTestsStateFilePortsV0(store)); err != nil {
		t.Fatalf("ensureOperationalDirectorPassedRequiredTestsQualityGateV0: %v", err)
	}

	serviceAssertRequiredTestsGateCountsV0(t, store, fixture.RunRef, 1, 1, 0)
	run, err := store.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 final: %v", err)
	}
	if blockers := orquestacoreworkflow.PendingBlockingQualityGateRefsForSubjectV0(run, fixture.TaskRef); len(blockers) != 0 {
		t.Fatalf("quality gate antiguo no resuelto: blockers=%v gates=%v", blockers, run.QualityGates)
	}
}

func serviceRequiredTestsStateFileStoreForTestV0(t *testing.T, rootDir string) *orquestastatefile.StoreV0 {
	t.Helper()
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: rootDir})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	return store
}

func serviceWaitExpiredStateFilePortsForTestV0(
	store *orquestastatefile.StoreV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	waiter *serviceContinueWaiterForTestV0,
) StartAppDirectorPortsV0 {
	return StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		EventReader:                store,
		OutboxLedger:               ledger,
		DirectorTaskStore:          store,
		WaitStateStore:             store,
		WaitStateWriter:            store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		ExternalWaiter:             waiter,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			{
				TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
				Reader:     ledger,
				Claimer:    ledger,
				Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
					RunStore:        store,
					EventSink:       store,
					Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
					ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityHighV0,
					OccurredAt:      "2026-05-22T22:10:00Z",
					CorrelationID:   "corr-app-director-service-statefile-capacity-001",
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
					OccurredAt:    "2026-05-22T22:10:00Z",
					CorrelationID: "corr-app-director-service-statefile-agent-001",
					RequestedBy:   "orquesta-app-director-service-test",
				},
				Acker: ledger,
			},
		},
	}
}

func serviceAssertWaitExpiredStateFileBlockedV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	runRef string,
	planRef string,
	wantAttempt int,
) (
	orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	orquestacionnucleoapp.OperationalDirectorPlanStepStateV0,
	orquestacionnucleoapp.WorkflowTaskWaitStateV0,
) {
	t.Helper()
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		!serviceStringInSetV0(state.BlockerRefs, "external-wait-exhausted") ||
		state.ClosureReason != "external-wait-exhausted" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		waitStep.Reason != "external-wait-exhausted" ||
		!serviceStringInSetV0(waitStep.BlockerRefs, "external-wait-exhausted") ||
		len(state.PendingAgentRefs) != 0 ||
		len(waitStep.PendingAgentRefs) != 0 ||
		len(waitStep.WaitRefs) != 1 {
		t.Fatalf("state=%+v waitStep=%+v", state, waitStep)
	}
	waitState, err := store.LoadWorkflowTaskWaitStateV0(context.Background(), runRef, waitStep.WaitRefs[0])
	if err != nil {
		t.Fatalf("LoadWorkflowTaskWaitStateV0: %v", err)
	}
	if waitState.Status != orquestacionnucleoapp.WorkflowTaskWaitStateStatusExpiredV0 ||
		waitState.Attempt != wantAttempt ||
		len(waitState.PendingAgentRefs) != 0 {
		t.Fatalf("waitState=%+v", waitState)
	}
	return state, waitStep, waitState
}

func serviceRequiredTestsStateFilePortsV0(store *orquestastatefile.StoreV0) StartAppDirectorPortsV0 {
	return StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		EventReader:                store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		RequiredTestEvidenceStore:  store,
	}
}

func serviceAssertRequiredTestsMissingStateV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	fixture serviceOperationalDirectorPlanStateReviewFixtureV0,
) {
	t.Helper()
	state, err := store.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 missing: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		!serviceStringInSetV0(state.BlockerRefs, "required-tests-evidence-missing") ||
		state.ReplanAttempts != 0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-evidence-missing" ||
		!serviceStringInSetV0(testsStep.BlockerRefs, "required-tests-evidence-missing") ||
		len(testsStep.RequiredTestEvidenceRefs) != 0 ||
		len(testsStep.ReplanDecisionRefs) != 0 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepPendingV0 {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}
}
