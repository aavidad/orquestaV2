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
	if _, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, serviceRequiredTestsStateFilePortsV0(recovered)); err == nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 without evidence: err nil")
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
