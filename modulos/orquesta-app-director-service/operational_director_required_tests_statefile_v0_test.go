package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
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
	serviceAssertRequiredTestsGateCountsV0(t, store, fixture.RunRef, 1, 0, 0)

	recovered := serviceRequiredTestsStateFileStoreForTestV0(t, rootDir)
	request.OccurredAt = "2026-05-22T21:00:01Z"
	request.CorrelationID = "corr-service-required-tests-statefile-restart"
	if _, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, serviceRequiredTestsStateFilePortsV0(recovered)); err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 without evidence: %v", err)
	}
	serviceAssertRequiredTestsMissingStateV0(t, recovered, fixture)
	serviceAssertRequiredTestsGateCountsV0(t, recovered, fixture.RunRef, 1, 0, 0)

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
		!serviceStringInSetV0(testsStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		replanStep.Reason != "required-tests-passed" ||
		!serviceStringInSetV0(replanStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) ||
		!serviceStringInSetV0(replanStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("state=%+v testsStep=%+v replanStep=%+v", state, testsStep, replanStep)
	}
	serviceAssertRequiredTestsGateCountsV0(t, recovered, fixture.RunRef, 1, 1, 0)
	run, err := recovered.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 final: %v", err)
	}
	if blockers := orquestacoreworkflow.PendingBlockingQualityGateRefsForSubjectV0(run, fixture.TaskRef); len(blockers) != 0 {
		t.Fatalf("quality gate requerido no resuelto: blockers=%v gates=%v", blockers, run.QualityGates)
	}
}
