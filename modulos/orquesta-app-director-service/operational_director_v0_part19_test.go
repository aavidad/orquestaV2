package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueRequestWithOperationalDirectorPlanStateV0RequiredTestsFailedReentraConStateFile(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	gateRef := "quality-gate-ref-app-director-required-tests-failed-state-file"
	replanRef := "replan-ref-app-director-required-tests-failed-state-file"
	followupAgentRef := "agent-ref-app-director-required-tests-failed-state-file"
	fixture.Run.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	fixture.Run.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + followupAgentRef,
	}
	fixture.Events = append(fixture.Events,
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:     fixture.RunRef,
			GateRef:    gateRef,
			PhaseID:    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef: fixture.TaskRef,
			Decision:   orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:  []string{fixture.RequiredTestEvidenceRef},
			Summary:    "Tests requeridos fallidos.",
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{followupAgentRef},
			Summary:        "Replan pendiente de materializar followup.",
		}),
	)
	rootDir := t.TempDir()
	store := mustServiceStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(context.Background(), fixture.Run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(context.Background(), fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(context.Background(), fixture.State); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	if err := store.SaveRequiredTestEvidenceV0(context.Background(), serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
		fixture,
		orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
	)); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T16:30:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		RequiredTestEvidenceStore:  store,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}

	recovered := mustServiceStateFileStoreForTestV0(t, rootDir)
	blockedState, err := recovered.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	blockedStep := serviceOperationalDirectorPlanStateStepForTestV0(t, blockedState, "step-run-required-tests")
	if blockedState.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		blockedStep.Reason != "required-tests-failed" ||
		!serviceStringInSetV0(blockedStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("blockedState=%+v blockedStep=%+v", blockedState, blockedStep)
	}

	lateRun := fixture.Run
	lateRun.Agents = append(lateRun.Agents, followupAgentRef)
	if err := recovered.SaveRunV0(context.Background(), lateRun); err != nil {
		t.Fatalf("SaveRunV0 late: %v", err)
	}
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-21T16:30:01Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   recovered,
		EventReader:                recovered,
		OperationalPlanStateStore:  recovered,
		OperationalPlanStateWriter: recovered,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	reloaded := mustServiceStateFileStoreForTestV0(t, rootDir)
	state, err := reloaded.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reloaded: %v", err)
	}
	evidence, err := reloaded.LoadRequiredTestEvidenceV0(context.Background(), fixture.RunRef, []string{fixture.RequiredTestEvidenceRef})
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0 reloaded: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-wait-subagents")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ActiveStepID != "step-wait-subagents" ||
		state.ReplanAttempts != 1 ||
		!serviceStringInSetV0(state.PendingAgentRefs, followupAgentRef) ||
		testsStep.Reason != "required-tests-failed-replan-recorded" ||
		!serviceStringInSetV0(testsStep.ReplanDecisionRefs, replanRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) ||
		len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0 {
		t.Fatalf("state=%+v testsStep=%+v waitStep=%+v evidence=%+v", state, testsStep, waitStep, evidence)
	}
}

func TestEnsureContinueOperationalDirectorPlanStateFromWorkflowTasksV0ConStateBloqueadoConservaPlanRef(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	state := fixture.State
	state.PlanRef = defaultOperationalDirectorDecisionPlanRefV0(fixture.RunRef)
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.ActiveStepID = "step-run-required-tests"
	for index, step := range state.Steps {
		if step.StepID == "step-run-required-tests" {
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			state.Steps[index].Reason = "required-tests-failed"
			state.Steps[index].RequiredTestEvidenceRefs = []string{fixture.RequiredTestEvidenceRef}
		}
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)

	got, err := ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef: fixture.RunRef,
	}, StartAppDirectorPortsV0{OperationalPlanStateStore: planStateStore})
	if err != nil {
		t.Fatalf("ensureContinueOperationalDirectorPlanStateFromWorkflowTasksV0: %v", err)
	}
	if got.OperationalDirectorPlanRef != state.PlanRef {
		t.Fatalf("plan_ref=%q, want %q", got.OperationalDirectorPlanRef, state.PlanRef)
	}
}

func TestOperationalDirectorPlanStateAfterRequiredTestsReplanV0ReabreSoloFollowupCausalEnScopeMultitarea(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	gateRef := "quality-gate-ref-app-director-required-tests-failed-multitask"
	replanRef := "replan-ref-app-director-required-tests-failed-multitask"
	followupAgentRef := "agent-ref-app-director-required-tests-failed-multitask"
	fixture.Run.QualityGates = []string{
		gateRef + "#decision:" + string(orquestacoreworkflow.QualityGateDecisionBlockedV0) +
			"#subject:" + fixture.TaskRef,
	}
	fixture.Run.Agents = append(fixture.Run.Agents, followupAgentRef)
	fixture.Run.ReplanDecisions = []string{
		replanRef + "#source:" + gateRef +
			"#task:" + fixture.TaskRef +
			"#action:" + string(orquestacoreworkflow.ReplanDecisionActionRetryTaskV0) +
			"#followups:" + followupAgentRef,
	}
	fixture.Events = append(fixture.Events,
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 5, orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0, orquestacoreworkflow.QualityGateRecordedPayloadV0{
			RunRef:     fixture.RunRef,
			GateRef:    gateRef,
			PhaseID:    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			SubjectRef: fixture.TaskRef,
			Decision:   orquestacoreworkflow.QualityGateDecisionBlockedV0,
			IssueRefs:  []string{fixture.RequiredTestEvidenceRef},
			Summary:    "Tests requeridos fallidos en scope multitarea.",
		}),
		serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 6, orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0, orquestacoreworkflow.ReplanDecisionRecordedPayloadV0{
			ReplanRef:      replanRef,
			RunRef:         fixture.RunRef,
			TaskRef:        fixture.TaskRef,
			SourceRef:      gateRef,
			AcceptedAction: orquestacoreworkflow.ReplanDecisionActionRetryTaskV0,
			FollowupRefs:   []string{followupAgentRef},
			Summary:        "Replan parcial no aceptable para scope multitarea.",
		}),
	)
	state := fixture.State
	state.ActiveStepID = "step-run-required-tests"
	activeStep := orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
	for index, step := range state.Steps {
		if step.StepID == "step-run-required-tests" {
			state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			state.Steps[index].TaskRefs = []string{fixture.TaskRef, "task-ref-app-director-required-tests-failed-multitask-002"}
			state.Steps[index].AgentRefs = []string{fixture.AgentRef, "agent-ref-app-director-required-tests-failed-multitask-002"}
			state.Steps[index].DeliveryRefs = []string{fixture.DeliveryRef, "delivery-ref-app-director-required-tests-failed-multitask-002"}
			state.Steps[index].ReviewResultRefs = []string{fixture.ReviewResultRef, "review-result-ref-app-director-required-tests-failed-multitask-002"}
			activeStep = state.Steps[index]
		}
	}

	next, changed, err := operationalDirectorPlanStateAfterRequiredTestsReplanV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OccurredAt:                 "2026-05-21T16:40:00Z",
			OperationalDirectorPlanRef: fixture.PlanRef,
		},
		StartAppDirectorPortsV0{EventReader: serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events}},
		state,
		activeStep,
		fixture.Run,
		[]operationalDirectorPlanAcceptedReviewMatchV0{
			{TaskRef: fixture.TaskRef},
			{TaskRef: "task-ref-app-director-required-tests-failed-multitask-002"},
		},
		[]string{fixture.RequiredTestEvidenceRef},
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStateAfterRequiredTestsReplanV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-run-required-tests")
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, next, "step-wait-subagents")
	if !changed ||
		next.ActiveStepID != "step-wait-subagents" ||
		next.ReplanAttempts != 1 ||
		!serviceStringInSetV0(next.PendingAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(next.PendingAgentRefs, "agent-ref-app-director-required-tests-failed-multitask-002") ||
		testsStep.Reason != "required-tests-failed-replan-recorded" ||
		!serviceStringInSetV0(testsStep.ReplanDecisionRefs, replanRef) ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		len(waitStep.TaskRefs) != 1 ||
		waitStep.TaskRefs[0] != fixture.TaskRef ||
		!serviceStringInSetV0(waitStep.PendingAgentRefs, followupAgentRef) {
		t.Fatalf("next=%+v changed=%v", next, changed)
	}
}
