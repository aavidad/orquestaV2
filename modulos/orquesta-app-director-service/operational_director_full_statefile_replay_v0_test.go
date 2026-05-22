package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
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
