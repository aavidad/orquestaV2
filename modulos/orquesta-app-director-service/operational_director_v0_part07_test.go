package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestOperationalDirectorOlaCohorteAmpliaOfflineScopeReviewTestsClose(t *testing.T) {
	runRef := "run-app-director-operational-wide-wave-offline"
	planRef := "plan-ref-app-director-operational-wide-wave-offline"
	parentTaskRef := "parent-task-app-director-operational-wide-wave-offline"
	waveRef := "wave-service-operational-closure-plan-state"
	cohortRef := "cohort-service-operational-closure-plan-state"
	childA := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-app-director-wide-wave-a",
		DeliveryRef:     "delivery-ref-app-director-wide-wave-a",
		ReviewRequestID: "review-request-ref-app-director-wide-wave-a",
		ReviewResultRef: "review-result-ref-app-director-wide-wave-a",
		AcceptedRef:     "accepted-review-ref-app-director-wide-wave-a",
		TestEvidenceRef: "test-evidence-ref-app-director-wide-wave-a",
		ValidationRef:   "validation-ref-app-director-wide-wave-a",
		ClosureRef:      "closure-ref-app-director-wide-wave-a",
	}
	childB := serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         "task-ref-app-director-wide-wave-b",
		DeliveryRef:     "delivery-ref-app-director-wide-wave-b",
		ReviewRequestID: "review-request-ref-app-director-wide-wave-b",
		ReviewResultRef: "review-result-ref-app-director-wide-wave-b",
		AcceptedRef:     "accepted-review-ref-app-director-wide-wave-b",
		TestEvidenceRef: "test-evidence-ref-app-director-wide-wave-b",
		ValidationRef:   "validation-ref-app-director-wide-wave-b",
		ClosureRef:      "closure-ref-app-director-wide-wave-b",
	}
	agentA := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childA.TaskRef)
	agentB := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(childB.TaskRef)
	outOfScopeAgent := "agent-ref-app-director-wide-wave-out-of-scope"
	waitAgentRefs := []string{agentA, agentB}
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{childA.TaskRef, childB.TaskRef}
	run.Agents = []string{agentA, agentB, outOfScopeAgent}
	run.StartedAgents = []string{agentA, agentB, outOfScopeAgent}
	run.DeliveredAgents = []string{agentA}
	run.DeliveredTasks = []string{childA.TaskRef}
	run.Deliveries = []string{childA.DeliveryRef}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(
		serviceOperationalDirectorWideWaveWaitStateForTestV0(
			runRef,
			planRef,
			parentTaskRef,
			waveRef,
			cohortRef,
			childA,
			childB,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T14:00:00Z",
		CorrelationID:              "corr-app-director-wide-wave-offline",
		OperationalDirectorPlanRef: planRef,
		WaitWaveRef:                waveRef,
		WaitCohortRef:              cohortRef,
		WaitParentTaskRef:          parentTaskRef,
	}
	loopRequest := orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:        runRef,
		WaitAgentRefs: waitAgentRefs,
	}
	partial, err := operationalDirectorPlanStatePostLoopScopeV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{RunStore: runStore, OperationalPlanStateStore: planStateStore},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Run:    run,
		},
		loopRequest,
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStatePostLoopScopeV0 partial: %v", err)
	}
	if partial.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("delivery parcial cerro scope: %+v", partial)
	}

	run.DeliveredAgents = []string{agentA, agentB}
	run.DeliveredTasks = []string{childA.TaskRef, childB.TaskRef}
	run.Deliveries = []string{childA.DeliveryRef, childB.DeliveryRef}
	run.Reviews = []string{childA.ReviewRequestID, childB.ReviewRequestID}
	run.ReviewResults = []string{
		serviceOperationalClosureReviewResultProjectionForTestV0(childA),
		serviceOperationalClosureReviewResultProjectionForTestV0(childB),
	}
	run.AcceptedReviews = []string{childA.AcceptedRef, childB.AcceptedRef}
	if err := runStore.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0 complete: %v", err)
	}
	eventReader := serviceOperationalClosureEventReaderForTestV0{
		Events: serviceOperationalClosureEventsForTasksForTestV0(t, runRef, childA, childB),
	}
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childA),
		serviceOperationalClosureRequiredTestEvidenceForTaskRefsV0(runRef, childB),
	)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
		serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childA),
		serviceOperationalClosureTaskWithRefsForTestV0(runRef, parentTaskRef, childB),
	)
	closureSource := &serviceOperationalClosureOpenTaskSourceForTestV0{
		Requests: map[string]orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
			childA.TaskRef: serviceOperationalClosureRequestForTaskRefsV0(childA),
			childB.TaskRef: serviceOperationalClosureRequestForTaskRefsV0(childB),
		},
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		EventReader:                eventReader,
		DirectorTaskStore:          taskStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
		OperationalClosureSource:   closureSource,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	scoped, err := operationalDirectorPlanStatePostLoopScopeV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
			Run:    run,
		},
		loopRequest,
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStatePostLoopScopeV0 complete: %v", err)
	}
	if scoped.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		serviceStringInSetV0(scoped.Run.DeliveredAgents, outOfScopeAgent) {
		t.Fatalf("scope no quedo acotado: %+v", scoped)
	}
	changed, err := applyOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, scoped)
	if err != nil || !changed {
		t.Fatalf("applyOperationalDirectorPlanStateAfterLoopV0 changed=%v err=%v", changed, err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 after apply: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	replanStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.ActiveStepID != "step-replan-or-close" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		replanStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, childA.DeliveryRef) ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, childB.DeliveryRef) ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, childA.TestEvidenceRef) ||
		!serviceStringInSetV0(testsStep.RequiredTestEvidenceRefs, childB.TestEvidenceRef) {
		t.Fatalf("state=%+v review=%+v tests=%+v replan=%+v", state, reviewStep, testsStep, replanStep)
	}

	first, issues, err := maybeCloseOperationalDirectorV0(context.Background(), request, ports, scoped, loopRequest)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first err=%v issues=%+v", err, issues)
	}
	if first.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		!serviceStringInSetV0(first.Run.ClosedTasks, childA.TaskRef) ||
		serviceStringInSetV0(first.Run.ClosedTasks, childB.TaskRef) {
		t.Fatalf("primer cierre no fue parcial por task: %+v", first.Run)
	}
	request.OccurredAt = "2026-05-22T14:00:01Z"
	second, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    first.Run,
		},
		loopRequest,
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 second err=%v issues=%+v", err, issues)
	}
	if second.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(second.Run.ClosedTasks, childA.TaskRef) ||
		!serviceStringInSetV0(second.Run.ClosedTasks, childB.TaskRef) ||
		serviceStringInSetV0(second.Run.DeliveredAgents, outOfScopeAgent) {
		t.Fatalf("cierre amplio no completo scope acotado: %+v", second.Run)
	}
	if len(closureSource.LastInput.WaitAgentRefs) != 2 ||
		!serviceStringInSetV0(closureSource.LastInput.WaitAgentRefs, agentA) ||
		!serviceStringInSetV0(closureSource.LastInput.WaitAgentRefs, agentB) ||
		serviceStringInSetV0(closureSource.LastInput.WaitAgentRefs, outOfScopeAgent) {
		t.Fatalf("closure source amplio scope=%+v", closureSource.LastInput.WaitAgentRefs)
	}
}

func TestUpdateOperationalDirectorPlanStateAfterLoopV0AvanzaDeReviewATestsRequeridos(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-17T14:20:00Z",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		EventReader:                serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}, orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0: %v", err)
	}
	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	if state.ActiveStepID != "step-run-required-tests" {
		t.Fatalf("active_step_id=%s state=%+v", state.ActiveStepID, state)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		!serviceStringInSetV0(reviewStep.DeliveryRefs, fixture.DeliveryRef) ||
		!serviceStringInSetV0(reviewStep.ReviewResultRefs, fixture.ReviewResultRef) ||
		!serviceStringInSetV0(reviewStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) {
		t.Fatalf("reviewStep=%+v", reviewStep)
	}
	if testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(testsStep.BlockerRefs, "required-tests-pending") ||
		!serviceStringInSetV0(testsStep.TaskRefs, fixture.TaskRef) {
		t.Fatalf("testsStep=%+v", testsStep)
	}
}
