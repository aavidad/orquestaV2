package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueRequestWithOperationalDirectorPlanStateV0RequiredTestsFailedReabreWaitSplitTaskPosterior(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T10:00:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 initial block: %v", err)
	}

	firstFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-required-tests-failed-split-a",
		fixture.TaskRef,
	)
	secondFollowup := serviceOperationalDirectorReplanSplitTaskForTestV0(
		fixture.RunRef,
		"task-ref-app-director-required-tests-failed-split-b",
		fixture.TaskRef,
	)
	followupRefs := []string{firstFollowup.TaskID, secondFollowup.TaskID}
	followupAgentRefs := []string{
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(firstFollowup.TaskID),
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(secondFollowup.TaskID),
	}
	gateRef := "quality-gate-ref-app-director-required-tests-failed-late-split"
	replanRef := "replan-ref-app-director-required-tests-failed-late-split"
	replannedRun := fixture.Run
	replannedRun.Tasks = append(replannedRun.Tasks, followupRefs...)
	replannedRun.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	replannedRun.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0) +
			"#followups:" + firstFollowup.TaskID + "+" + secondFollowup.TaskID,
	}
	replannedEvents := append(append([]orquestacoreworkflow.OrchestrationEventV0(nil), fixture.Events...),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:       fixture.RunRef,
			GateRef:      gateRef,
			PhaseID:      string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef:   fixture.TaskRef,
			Decision:     orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:    []string{fixture.RequiredTestEvidenceRef},
			Summary:      "Tests requeridos fallidos.",
			EvidenceRefs: []string{"evidence-ref-quality-gate-required-tests-failed-late-split"},
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionSplitTaskV0,
			FollowupRefs:   followupRefs,
			Summary:        "Split posterior por tests requeridos fallidos.",
			EvidenceRefs:   []string{"evidence-ref-replan-required-tests-failed-late-split"},
		}),
	)

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T10:00:01Z",
		CorrelationID:              "corr-app-director-required-tests-failed-late-split",
		OperationalDirectorPlanRef: fixture.PlanRef,
		WaitAgentRefs:              []string{fixture.AgentRef},
		WaitWaveRef:                fixture.WaveRef,
		WaitCohortRef:              fixture.CohortRef,
		WaitParentTaskRef:          fixture.ParentTaskRef,
	}, StartAppDirectorPortsV0{
		RunStore:    orquestacionnucleoapp.NewInMemoryRunStoreV0(replannedRun),
		EventReader: serviceOperationalClosureEventReaderForTestV0{Events: replannedEvents},
		DirectorTaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
			serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture),
			firstFollowup,
			secondFollowup,
		),
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 late split: %v", err)
	}
	if reentered.WaitWaveRef != firstFollowup.WaveRef ||
		reentered.WaitCohortRef != firstFollowup.CohortRef ||
		reentered.WaitParentTaskRef != fixture.TaskRef ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followupAgentRefs=%+v old=%s", reentered, followupAgentRefs, fixture.AgentRef)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 late split: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ActiveWaveRef != firstFollowup.WaveRef ||
		state.ActiveCohortRef != firstFollowup.CohortRef ||
		state.ActiveParentTaskRef != fixture.TaskRef ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(state.PendingAgentRefs, fixture.AgentRef) {
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
		waitStep.Reason != "required-tests-failed-replan-followups-waiting" ||
		!serviceStringInSetV0(waitStep.TaskRefs, firstFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.TaskRefs, secondFollowup.TaskID) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRefs[0]) ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRefs[1]) ||
		serviceStringInSetV0(waitStep.PendingAgentRefs, fixture.AgentRef) {
		t.Fatalf("waitStep=%+v", waitStep)
	}
}
