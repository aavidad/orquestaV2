package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestContinueAppDirectorV0StateFileReplayReviewRunnerReplanOrCloseCloseNoDuplica(t *testing.T) {
	ctx := context.Background()
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	fixture.Events[2] = serviceOperationalDirectorPlanStateEventForTestV0(t, fixture.RunRef, 3, orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0, orquestacoreworkflow.ReviewResultV0{
		ReviewResultRef: fixture.ReviewResultRef,
		ReviewRequestID: fixture.ReviewRequestID,
		DeliveryRef:     fixture.DeliveryRef,
		Status:          orquestacoreworkflow.ReviewResultStatusAcceptedV0,
		Summary:         "Review aceptada; el runner state-file debe generar la evidencia durable.",
		EvidenceRefs:    []string{"evidence-ref-review-result-accepted-statefile-replay"},
	})
	run := serviceContinueClosureRunForTestV0(fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.ProjectRef = fixture.Run.ProjectRef
	run.AppSpecRef = fixture.Run.AppSpecRef
	run.Tasks = append([]string(nil), fixture.Run.Tasks...)
	run.Agents = append([]string(nil), fixture.Run.Agents...)
	run.StartedAgents = append([]string(nil), fixture.Run.StartedAgents...)
	run.DeliveredAgents = append([]string(nil), fixture.Run.DeliveredAgents...)
	run.DeliveredTasks = append([]string(nil), fixture.Run.DeliveredTasks...)
	run.Deliveries = append([]string(nil), fixture.Run.Deliveries...)
	run.Reviews = append([]string(nil), fixture.Run.Reviews...)
	run.ReviewResults = append([]string(nil), fixture.Run.ReviewResults...)
	run.AcceptedReviews = append([]string(nil), fixture.Run.AcceptedReviews...)

	rootDir := t.TempDir()
	store := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	if err := store.SaveWorkflowTaskV0(ctx, serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, fixture.State); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	executor := &fakeServiceRequiredTestCommandExecutorV0{
		results: map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			fixture.RequiredTest: {
				Status:       orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-service-required-test-output-statefile-replay-001"},
			},
		},
	}
	closureSource := &serviceOperationalDirectorClosureSourceFromRequestForTestV0{Fixture: fixture}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T23:10:00Z",
		CorrelationID:              "corr-service-full-statefile-replay-first",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
		MaxBursts:                  1,
		MaxStepsPerBurst:           1,
		MaxDispatchesPerWait:       1,
	}

	first, err := ContinueAppDirectorV0(ctx, request, serviceFullReplayStateFilePortsForTestV0(store, executor, closureSource))
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	if first.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("first run no cerrado: %+v", first.Run)
	}
	if len(executor.commands) != 1 || executor.commands[0] != fixture.RequiredTest {
		t.Fatalf("runner first commands=%+v", executor.commands)
	}
	if !closureSource.Called || len(closureSource.LastRequest.RequiredTestEvidenceRefs) != 1 {
		t.Fatalf("closure source first called=%v request=%+v", closureSource.Called, closureSource.LastRequest)
	}
	eventsAfterFirst := serviceFullReplayEventsForTestV0(t, store, fixture.RunRef)
	serviceAssertFullReplayClosureCountsV0(t, store, fixture.RunRef, fixture.PlanRef, 1, 1, 1, 1)

	recovered := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	replayExecutor := &fakeServiceRequiredTestCommandExecutorV0{
		results: map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			fixture.RequiredTest: {
				Status:       orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-service-required-test-output-statefile-replay-should-not-run"},
			},
		},
	}
	replayClosureSource := &serviceOperationalDirectorClosureSourceFromRequestForTestV0{Fixture: fixture}
	request.OccurredAt = "2026-05-22T23:10:01Z"
	request.CorrelationID = "corr-service-full-statefile-replay-second"
	closedRun, err := recovered.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 replay: %v", err)
	}
	replay, issues, err := maybeCloseOperationalDirectorV0(
		ctx,
		request,
		serviceFullReplayStateFilePortsForTestV0(recovered, replayExecutor, replayClosureSource),
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    closedRun,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 replay err=%v issues=%+v", err, issues)
	}
	if replay.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("replay run no cerrado: %+v", replay.Run)
	}
	if len(replayExecutor.commands) != 0 {
		t.Fatalf("runner replay no debe ejecutar: commands=%+v", replayExecutor.commands)
	}
	serviceAssertFullReplayClosureCountsV0(t, recovered, fixture.RunRef, fixture.PlanRef, 1, 1, 1, 1)
	eventsAfterReplay := serviceFullReplayEventsForTestV0(t, recovered, fixture.RunRef)
	if len(eventsAfterReplay) != len(eventsAfterFirst) {
		t.Fatalf("replay duplico eventos: first=%d replay=%d events=%+v", len(eventsAfterFirst), len(eventsAfterReplay), eventsAfterReplay)
	}

	directReplayExecutor := &fakeServiceRequiredTestCommandExecutorV0{
		results: map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			fixture.RequiredTest: {
				Status:       orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-service-required-test-output-statefile-direct-replay-should-not-run"},
			},
		},
	}
	directReplayClosureSource := &serviceOperationalDirectorClosureSourceFromRequestForTestV0{Fixture: fixture}
	request.OccurredAt = "2026-05-22T23:10:02Z"
	request.CorrelationID = "corr-service-full-statefile-replay-direct-continue"
	directReplay, err := ContinueAppDirectorV0(
		ctx,
		request,
		serviceFullReplayStateFilePortsForTestV0(recovered, directReplayExecutor, directReplayClosureSource),
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 direct replay: %v", err)
	}
	if directReplay.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		directReplay.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("direct replay no-op invalido: %+v", directReplay)
	}
	if len(directReplayExecutor.commands) != 0 {
		t.Fatalf("runner direct replay no debe ejecutar: commands=%+v", directReplayExecutor.commands)
	}
	if directReplayClosureSource.Called {
		t.Fatalf("closure source direct replay no debe llamarse: request=%+v", directReplayClosureSource.LastRequest)
	}
	serviceAssertFullReplayClosureCountsV0(t, recovered, fixture.RunRef, fixture.PlanRef, 1, 1, 1, 1)
	eventsAfterDirectReplay := serviceFullReplayEventsForTestV0(t, recovered, fixture.RunRef)
	if len(eventsAfterDirectReplay) != len(eventsAfterFirst) {
		t.Fatalf("direct replay duplico eventos: first=%d direct=%d events=%+v", len(eventsAfterFirst), len(eventsAfterDirectReplay), eventsAfterDirectReplay)
	}
}

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

func TestContinueAppDirectorV0StateFileReplayReviewChangesRequestedReworkReplanNoDuplica(t *testing.T) {
	ctx := context.Background()
	fixture := serviceOperationalDirectorPlanStateReviewReworkReplanFixtureForTestV0(
		t,
		orquestacoreworkflow.ReviewResultStatusChangesRequestedV0,
	)
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

	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-23T00:20:00Z",
		CorrelationID:              "corr-service-review-negative-statefile-replay-first",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(store), orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	serviceAssertFullReplayReviewReworkReplanCountsV0(t, store, fixture, 1, 1, 1)

	recovered := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	recoveredRun, err := recovered.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 recovered: %v", err)
	}
	request.OccurredAt = "2026-05-23T00:20:01Z"
	request.CorrelationID = "corr-service-review-negative-statefile-replay-second"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(recovered), orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    recoveredRun,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}
	serviceAssertFullReplayReviewReworkReplanCountsV0(t, recovered, fixture, 1, 1, 1)

	recoveredAgain := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	request.OccurredAt = "2026-05-23T00:20:02Z"
	request.CorrelationID = "corr-service-review-negative-statefile-replay-third"
	_, err = continueRequestWithOperationalDirectorPlanStateV0(ctx, request, serviceFullReplayStateFileRequiredTestsReplanPortsForTestV0(recoveredAgain))
	if err == nil {
		t.Fatalf("err nil")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "operational_director_plan_state.active_step" {
		t.Fatalf("err=%T %#v", err, err)
	}
	serviceAssertFullReplayReviewReworkReplanCountsV0(t, recoveredAgain, fixture, 1, 1, 1)
}

func serviceFullReplayStateFileStoreForTestV0(t *testing.T, rootDir string) *orquestastatefile.StoreV0 {
	t.Helper()
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: rootDir})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	return store
}

func serviceFullReplayStateFilePortsForTestV0(
	store *orquestastatefile.StoreV0,
	executor *fakeServiceRequiredTestCommandExecutorV0,
	closureSource *serviceOperationalDirectorClosureSourceFromRequestForTestV0,
) StartAppDirectorPortsV0 {
	dispatchRunStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	dispatchEventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	dispatchLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	return StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		EventReader:                store,
		OutboxLedger:               dispatchLedger,
		DirectorTaskStore:          store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		RequiredTestEvidenceStore:  store,
		RequiredTestRunner: orquestacionnucleoapp.RequiredTestRunnerV0{
			Executor:       executor,
			EvidenceWriter: store,
		},
		OperationalClosureSource: closureSource,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(dispatchRunStore, dispatchEventSink, dispatchLedger),
		},
	}
}

func serviceFullReplayStateFileClosurePortsForTestV0(
	store *orquestastatefile.StoreV0,
	closureSource *serviceOperationalClosureSourceForTestV0,
) StartAppDirectorPortsV0 {
	return StartAppDirectorPortsV0{
		RunStore:                   store,
		EventSink:                  store,
		EventReader:                store,
		DirectorTaskStore:          store,
		OperationalPlanStateStore:  store,
		OperationalPlanStateWriter: store,
		RequiredTestEvidenceStore:  store,
		OperationalClosureSource:   closureSource,
	}
}

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
