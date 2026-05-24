package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestContinueAppDirectorV0WaitExpiredStateFileRestartNoDuplicaEvidenceRefs(t *testing.T) {
	ctx := context.Background()
	runRef := "run-app-director-operational-wait-expired-statefile-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.FunctionContracts = []string{contractRef}
	plan := serviceOperationalDirectorPlanForContinueTestV0(t, runRef)
	rootDir := t.TempDir()
	store := serviceRequiredTestsStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	waiter := &serviceContinueWaiterForTestV0{Continue: true}

	result, err := ContinueAppDirectorV0(ctx, ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-22T22:10:00Z",
		CorrelationID:           "corr-app-director-operational-wait-expired-statefile-first",
		MaxBursts:               4,
		MaxStepsPerBurst:        4,
		MaxDispatchesPerWait:    4,
		MaxCommands:             20,
		MaxOutboxPerCycle:       8,
		MaxExternalWaits:        2,
		OperationalDirectorPlan: plan,
		OperationalDirectorFunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  contractRef,
			FunctionName: "OperationalDirectorCut",
		}},
	}, serviceWaitExpiredStateFilePortsForTestV0(store, ledger, waiter))
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	if result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		waiter.Calls != 2 {
		t.Fatalf("result=%+v waiter_calls=%d", result, waiter.Calls)
	}
	state, waitStep, waitState := serviceAssertWaitExpiredStateFileBlockedV0(t, store, runRef, plan.PlanRef, 3)
	waitRef := waitStep.WaitRefs[0]
	if serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-wait-expired-v0") != 1 ||
		serviceCountStringV0(waitStep.EvidenceRefs, "evidence-ref-app-director-wait-subagents-expired-v0") != 1 ||
		serviceCountStringV0(waitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0") != 1 {
		t.Fatalf("state=%+v waitStep=%+v waitState=%+v", state, waitStep, waitState)
	}

	recovered := serviceRequiredTestsStateFileStoreForTestV0(t, rootDir)
	replayLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	replayWaiter := &serviceContinueWaiterForTestV0{Continue: true}
	_, err = ContinueAppDirectorV0(ctx, ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T22:10:01Z",
		CorrelationID:              "corr-app-director-operational-wait-expired-statefile-restart",
		MaxExternalWaits:           2,
		OperationalDirectorPlanRef: plan.PlanRef,
	}, serviceWaitExpiredStateFilePortsForTestV0(recovered, replayLedger, replayWaiter))
	if err == nil {
		t.Fatalf("ContinueAppDirectorV0 replay err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
	replayedState, replayedWaitStep, replayedWaitState := serviceAssertWaitExpiredStateFileBlockedV0(t, recovered, runRef, plan.PlanRef, 3)
	if replayWaiter.Calls != 0 ||
		len(replayedWaitStep.WaitRefs) != 1 ||
		replayedWaitStep.WaitRefs[0] != waitRef ||
		serviceCountStringV0(replayedState.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-wait-expired-v0") != 1 ||
		serviceCountStringV0(replayedWaitStep.EvidenceRefs, "evidence-ref-app-director-wait-subagents-expired-v0") != 1 ||
		serviceCountStringV0(replayedWaitState.EvidenceRefs, "evidence-ref-app-director-wait-state-expired-v0") != 1 {
		t.Fatalf("replayWaiter=%d state=%+v waitStep=%+v waitState=%+v", replayWaiter.Calls, replayedState, replayedWaitStep, replayedWaitState)
	}
}

func TestRequiredTestsEvidenceMissingStateFileRestartNoDuplicaGateYReentraConEvidencePassed(t *testing.T) {
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
		OccurredAt:                 "2026-05-22T21:00:00Z",
		CorrelationID:              "corr-service-required-tests-statefile-missing",
		RequestedBy:                "orquesta-app-director-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceRequiredTestsStateFilePortsV0(store), loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	serviceAssertRequiredTestsMissingStateV0(t, store, fixture)
	serviceAssertRequiredTestsGateCountsV0(t, store, fixture.RunRef, 1, 0)

	recovered := serviceRequiredTestsStateFileStoreForTestV0(t, rootDir)
	request.OccurredAt = "2026-05-22T21:00:01Z"
	request.CorrelationID = "corr-service-required-tests-statefile-restart"
	if _, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, serviceRequiredTestsStateFilePortsV0(recovered)); err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 without evidence: %v", err)
	}
	serviceAssertRequiredTestsMissingStateV0(t, recovered, fixture)
	serviceAssertRequiredTestsGateCountsV0(t, recovered, fixture.RunRef, 1, 0)

	evidence := serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
		fixture,
		orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
	)
	if err := recovered.SaveRequiredTestEvidenceV0(ctx, evidence); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}
	request.OccurredAt = "2026-05-22T21:00:02Z"
	request.CorrelationID = "corr-service-required-tests-statefile-evidence"
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, serviceRequiredTestsStateFilePortsV0(recovered))
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 with evidence: %v", err)
	}
	if reentered.WaitWaveRef != fixture.WaveRef ||
		reentered.WaitCohortRef != fixture.CohortRef ||
		reentered.WaitParentTaskRef != fixture.ParentTaskRef ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v fixture=%+v", reentered, fixture)
	}

	state, err := recovered.LoadOperationalDirectorPlanStateV0(ctx, fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 final: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-replan-or-close" ||
		len(state.BlockerRefs) != 0 ||
		state.ReplanAttempts != 0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		testsStep.Reason != "required-tests-passed" ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		replanStep.Reason != "required-tests-passed" ||
		!serviceStringInSetV0(replanStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}
	serviceAssertRequiredTestsGateCountsV0(t, recovered, fixture.RunRef, 1, 0)
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

func serviceAssertRequiredTestsGateCountsV0(
	t *testing.T,
	store *orquestastatefile.StoreV0,
	runRef string,
	wantGateEvents int,
	wantReplanEvents int,
) {
	t.Helper()
	events, err := store.LoadRunEventsV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunEventsV0: %v", err)
	}
	var gateEvents, replanEvents int
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			gateEvents++
			var payload orquestacoreworkflow.QualityGateRecordedPayloadV0
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				t.Fatalf("QualityGateRecorded payload: %v", err)
			}
			if payload.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
				!serviceStringInSetV0(payload.IssueRefs, "required-tests-evidence-missing") {
				t.Fatalf("quality gate payload=%+v", payload)
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEvents++
		}
	}
	if gateEvents != wantGateEvents || replanEvents != wantReplanEvents {
		t.Fatalf("gateEvents=%d replanEvents=%d want=%d/%d events=%+v", gateEvents, replanEvents, wantGateEvents, wantReplanEvents, events)
	}
	run, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.QualityGates) != wantGateEvents || len(run.ReplanDecisions) != wantReplanEvents {
		t.Fatalf("run quality_gates=%v replan_decisions=%v", run.QualityGates, run.ReplanDecisions)
	}
}
