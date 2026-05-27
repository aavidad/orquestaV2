package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueAppDirectorV0StateFileReplayClosureIssueReplanNoDuplica(t *testing.T) {
	ctx := context.Background()
	runRef := "run-service-operational-closure-statefile-replan-replay"
	planRef := "plan-ref-service-operational-closure-statefile-replan-replay"
	taskRef := "task-ref-service-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{taskRef}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.DeliveredTasks = []string{taskRef}
	run.Reviews = []string{"review-request-ref-service-operational-closure-001"}
	run.ReviewResults = []string{
		"review-result-ref-service-operational-closure-001#review_result:accepted#review_request:review-request-ref-service-operational-closure-001#delivery:delivery-ref-service-operational-closure-001",
	}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	run.Agents = []string{agentRef}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	state := serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)
	state.Steps = append([]orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
		{
			StepID:        "step-wait-subagents",
			Kind:          orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
			Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
			WaveRef:       state.ActiveWaveRef,
			CohortRef:     state.ActiveCohortRef,
			ParentTaskRef: state.ActiveParentTaskRef,
			TaskRefs:      []string{taskRef},
			AgentRefs:     []string{agentRef},
			WaitRefs:      []string{"wait-ref-service-operational-closure-statefile-replan-replay"},
		},
	}, state.Steps...)

	rootDir := t.TempDir()
	store := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, runRef, newServiceOperationalClosureEventReaderForTestV0(t, runRef).Events); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	if err := store.SaveWorkflowTaskV0(ctx, serviceOperationalClosureTaskForTestV0(runRef)); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := store.SaveRequiredTestEvidenceV0(ctx, serviceOperationalClosureRequiredTestEvidenceForTestV0(runRef)); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, state); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	source := &serviceOperationalClosureSourceForTestV0{
		Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			TaskID:                   taskRef,
			DeliveryRef:              "delivery-ref-service-operational-closure-001",
			AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
			ValidationRef:            "validation-ref-service-operational-closure-statefile-replan-replay",
			RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
			EvidenceRefs:             []string{"evidence-ref-service-operational-closure-statefile-replan-replay"},
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T23:30:00Z",
		CorrelationID:              "corr-service-operational-closure-statefile-replan-replay",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: planRef,
	}

	first, issues, err := maybeCloseOperationalDirectorV0(
		ctx,
		request,
		serviceFullReplayStateFileClosurePortsForTestV0(store, source),
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil {
		t.Fatalf("maybeCloseOperationalDirectorV0 first: %v", err)
	}
	if len(issues) == 0 || issues[0].Field != "closure_ref" {
		t.Fatalf("issues first=%+v", issues)
	}
	if first.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar: %+v", first.Run)
	}
	serviceAssertFullReplayReplanCountsV0(t, store, runRef, planRef, 1, 1, 0, 0, 0)

	recovered := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	request.OccurredAt = "2026-05-22T23:30:01Z"
	loadedRun, err := recovered.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0 replay: %v", err)
	}
	_, replayIssues, err := maybeCloseOperationalDirectorV0(
		ctx,
		request,
		serviceFullReplayStateFileClosurePortsForTestV0(recovered, source),
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    loadedRun,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(replayIssues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 replay err=%v issues=%+v", err, replayIssues)
	}
	serviceAssertFullReplayReplanCountsV0(t, recovered, runRef, planRef, 1, 1, 0, 0, 0)

	replannedRun, err := recovered.LoadRunV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadRunV0 replanned: %v", err)
	}
	if len(replannedRun.ReplanDecisions) != 1 {
		t.Fatalf("ReplanDecisions=%+v", replannedRun.ReplanDecisions)
	}
	followupAgentRef := serviceFirstAgentFollowupFromReplanProjectionForTestV0(t, replannedRun.ReplanDecisions[0])
	replannedRun.Agents = append(replannedRun.Agents, followupAgentRef)
	if err := recovered.SaveRunV0(ctx, replannedRun); err != nil {
		t.Fatalf("SaveRunV0 followup: %v", err)
	}

	request.OccurredAt = "2026-05-22T23:30:02Z"
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, serviceFullReplayStateFileClosurePortsForTestV0(recovered, source))
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 followup: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, agentRef) {
		t.Fatalf("reentered=%+v followup=%s old=%s", reentered, followupAgentRef, agentRef)
	}
	serviceAssertFullReplayReplanCountsV0(t, recovered, runRef, planRef, 1, 1, 0, 0, 1)

	recoveredAgain := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	request.OccurredAt = "2026-05-22T23:30:03Z"
	replayedWait, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, serviceFullReplayStateFileClosurePortsForTestV0(recoveredAgain, source))
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 replay wait: %v", err)
	}
	if !serviceStringInSetV0(replayedWait.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(replayedWait.WaitAgentRefs, agentRef) {
		t.Fatalf("replayedWait=%+v followup=%s old=%s", replayedWait, followupAgentRef, agentRef)
	}
	serviceAssertFullReplayReplanCountsV0(t, recoveredAgain, runRef, planRef, 1, 1, 0, 0, 1)
}

func TestContinueAppDirectorV0StateFileReplayRequiredTestsFailedReplanNoDuplica(t *testing.T) {
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

	rootDir := t.TempDir()
	store := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(ctx, fixture.Run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, fixture.State); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	if err := store.SaveRequiredTestEvidenceV0(ctx, serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
		fixture,
		orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
	)); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}

	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-23T00:00:00Z",
		CorrelationID:              "corr-service-required-tests-statefile-replan-replay-first",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	firstPorts := serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(store)
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, firstPorts, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	serviceAssertFullReplayRequiredTestsReplanCountsV0(t, store, fixture.RunRef, fixture.PlanRef, 1, 1, 1, 0)

	firstRun, err := store.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 first: %v", err)
	}
	if firstRun.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		len(firstRun.QualityGates) != 1 ||
		len(firstRun.ReplanDecisions) != 1 {
		t.Fatalf("run tras primer replan invalido: %+v", firstRun)
	}

	recovered := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	recoveredRun, err := recovered.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 recovered: %v", err)
	}
	request.OccurredAt = "2026-05-23T00:00:01Z"
	request.CorrelationID = "corr-service-required-tests-statefile-replan-replay-second"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(recovered), orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    recoveredRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}
	serviceAssertFullReplayRequiredTestsReplanCountsV0(t, recovered, fixture.RunRef, fixture.PlanRef, 1, 1, 1, 0)

	recoveredAgain := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	directReplayRun, err := recoveredAgain.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 recovered again: %v", err)
	}
	request.OccurredAt = "2026-05-23T00:00:02Z"
	request.CorrelationID = "corr-service-required-tests-statefile-replan-replay-third"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(recoveredAgain), orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    directReplayRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 direct replay: %v", err)
	}
	serviceAssertFullReplayRequiredTestsReplanCountsV0(t, recoveredAgain, fixture.RunRef, fixture.PlanRef, 1, 1, 1, 0)
}
