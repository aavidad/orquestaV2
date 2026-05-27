package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueRequestWithOperationalDirectorPlanStateV0ReintentaCierreTrasSourceDisponible(t *testing.T) {
	fixture := newServiceOperationalClosurePrerequisiteRetryFixtureV0(t, "source")
	blockPorts := fixture.fullPortsV0(t)
	blockPorts.OperationalClosureSource = nil

	serviceBlockOperationalClosurePrerequisiteForTestV0(t, fixture, blockPorts, "operational-closure-source-unavailable", 0)
	serviceReopenAndCloseOperationalClosurePrerequisiteForTestV0(t, fixture)
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReintentaCierreTrasTaskStoreDisponible(t *testing.T) {
	fixture := newServiceOperationalClosurePrerequisiteRetryFixtureV0(t, "task-store")
	blockPorts := fixture.fullPortsV0(t)
	blockPorts.DirectorTaskStore = nil

	serviceBlockOperationalClosurePrerequisiteForTestV0(t, fixture, blockPorts, "operational-closure-task-store-unavailable", 0)
	if fixture.Source.Called {
		t.Fatalf("closure source no debe invocarse sin task store")
	}
	serviceReopenAndCloseOperationalClosurePrerequisiteForTestV0(t, fixture)
}

func TestContinueRequestWithOperationalDirectorPlanStateV0ReintentaCierreTrasOutboxDrenado(t *testing.T) {
	fixture := newServiceOperationalClosurePrerequisiteRetryFixtureV0(t, "outbox")

	serviceBlockOperationalClosurePrerequisiteForTestV0(t, fixture, fixture.fullPortsV0(t), "operational-closure-outbox-pending", 2)
	if fixture.Source.Called {
		t.Fatalf("closure source no debe invocarse con outbox pendiente")
	}
	serviceReopenAndCloseOperationalClosurePrerequisiteForTestV0(t, fixture)
}

func TestContinueRequestWithOperationalDirectorPlanStateV0StateFileRestartReintentaPrereqSinDuplicar(t *testing.T) {
	ctx := context.Background()
	fixture := newServiceOperationalClosurePrerequisiteRetryFixtureV0(t, "statefile-restart")
	rootDir := t.TempDir()
	store := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	if err := store.SaveRunV0(ctx, fixture.Run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := store.AppendRunEventsV0(ctx, fixture.RunRef, newServiceOperationalClosureEventReaderForTestV0(t, fixture.RunRef).Events); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	if err := store.SaveWorkflowTaskV0(ctx, serviceOperationalClosureTaskForTestV0(fixture.RunRef)); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := store.SaveRequiredTestEvidenceV0(ctx, serviceOperationalClosureRequiredTestEvidenceForTestV0(fixture.RunRef)); err != nil {
		t.Fatalf("SaveRequiredTestEvidenceV0: %v", err)
	}
	if err := store.SaveOperationalDirectorPlanStateV0(ctx, serviceOperationalClosurePlanStateReadyForCloseV0(fixture.RunRef, fixture.PlanRef)); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}

	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T23:50:00Z",
		CorrelationID:              "corr-service-closure-prereq-statefile-first",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	_, issues, err := maybeCloseOperationalDirectorV0(
		ctx,
		request,
		serviceOperationalClosurePrerequisiteStateFilePortsV0(store, nil),
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    fixture.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first block err=%v issues=%+v", err, issues)
	}
	serviceAssertOperationalClosurePrerequisiteStateFileBlockedV0(t, store, fixture, "operational-closure-source-unavailable")

	recovered := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	loadedRun, err := recovered.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 recovered: %v", err)
	}
	request.OccurredAt = "2026-05-22T23:50:01Z"
	request.CorrelationID = "corr-service-closure-prereq-statefile-reblock"
	_, replayIssues, err := maybeCloseOperationalDirectorV0(
		ctx,
		request,
		serviceOperationalClosurePrerequisiteStateFilePortsV0(recovered, nil),
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    loadedRun,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(replayIssues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 replay block err=%v issues=%+v", err, replayIssues)
	}
	serviceAssertOperationalClosurePrerequisiteStateFileBlockedV0(t, recovered, fixture, "operational-closure-source-unavailable")

	request.OccurredAt = "2026-05-22T23:50:02Z"
	request.CorrelationID = "corr-service-closure-prereq-statefile-ready"
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(
		ctx,
		request,
		serviceOperationalClosurePrerequisiteStateFilePortsV0(recovered, fixture.Source),
	)
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 ready: %v", err)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")
	if !serviceStringInSetV0(reentered.WaitAgentRefs, agentRef) {
		t.Fatalf("reentered=%+v falta agent=%s", reentered, agentRef)
	}
	serviceAssertOperationalClosurePrerequisiteStateFileReadyV0(t, recovered, fixture)

	runForClose, err := recovered.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 close: %v", err)
	}
	request.OccurredAt = "2026-05-22T23:50:03Z"
	request.CorrelationID = "corr-service-closure-prereq-statefile-close"
	closed, closeIssues, err := maybeCloseOperationalDirectorV0(
		ctx,
		request,
		serviceOperationalClosurePrerequisiteStateFilePortsV0(recovered, fixture.Source),
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    runForClose,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(closeIssues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 close err=%v issues=%+v", err, closeIssues)
	}
	if closed.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no cerrado: %+v", closed.Run)
	}
	serviceAssertOperationalClosurePrerequisiteStateFileClosedV0(t, recovered, fixture)

	recoveredAgain := serviceFullReplayStateFileStoreForTestV0(t, rootDir)
	closedRun, err := recoveredAgain.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 closed replay: %v", err)
	}
	request.OccurredAt = "2026-05-22T23:50:04Z"
	request.CorrelationID = "corr-service-closure-prereq-statefile-closed-replay"
	_, closedReplayIssues, err := maybeCloseOperationalDirectorV0(
		ctx,
		request,
		serviceOperationalClosurePrerequisiteStateFilePortsV0(recoveredAgain, fixture.Source),
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    closedRun,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(closedReplayIssues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 closed replay err=%v issues=%+v", err, closedReplayIssues)
	}
	serviceAssertOperationalClosurePrerequisiteStateFileClosedV0(t, recoveredAgain, fixture)
}

type serviceOperationalClosurePrerequisiteRetryFixtureV0 struct {
	RunRef         string
	PlanRef        string
	Run            orquestacoreworkflow.OrchestrationRunV0
	RunStore       *orquestacionnucleoapp.InMemoryRunStoreV0
	EventSink      *orquestacionnucleoapp.InMemoryEventSinkV0
	PlanStateStore *orquestacionnucleoapp.InMemoryOperationalDirectorPlanStateStoreV0
	Source         *serviceOperationalClosureSourceForTestV0
}

func newServiceOperationalClosurePrerequisiteRetryFixtureV0(
	t *testing.T,
	suffix string,
) serviceOperationalClosurePrerequisiteRetryFixtureV0 {
	t.Helper()
	runRef := "run-service-operational-closure-prerequisite-" + suffix
	planRef := "plan-ref-service-operational-closure-prerequisite-" + suffix
	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{"task-ref-service-operational-closure-001"}
	run.Deliveries = []string{"delivery-ref-service-operational-closure-001"}
	run.AcceptedReviews = []string{"accepted-review-ref-service-operational-closure-001"}
	return serviceOperationalClosurePrerequisiteRetryFixtureV0{
		RunRef:         runRef,
		PlanRef:        planRef,
		Run:            run,
		RunStore:       orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EventSink:      orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		PlanStateStore: orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(serviceOperationalClosurePlanStateReadyForCloseV0(runRef, planRef)),
		Source: &serviceOperationalClosureSourceForTestV0{
			Request: orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
				TaskID:                   "task-ref-service-operational-closure-001",
				DeliveryRef:              "delivery-ref-service-operational-closure-001",
				AcceptedReviewRef:        "accepted-review-ref-service-operational-closure-001",
				ValidationRef:            "validation-ref-service-operational-closure-prerequisite-" + suffix,
				ClosureRef:               "closure-ref-service-operational-closure-prerequisite-" + suffix,
				RequiredTestEvidenceRefs: []string{"test-evidence-ref-service-operational-closure-001"},
				EvidenceRefs:             []string{"evidence-ref-service-operational-closure-prerequisite-" + suffix},
			},
		},
	}
}

func (fixture serviceOperationalClosurePrerequisiteRetryFixtureV0) fullPortsV0(t *testing.T) StartAppDirectorPortsV0 {
	t.Helper()
	return StartAppDirectorPortsV0{
		RunStore:                   fixture.RunStore,
		EventSink:                  fixture.EventSink,
		EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, fixture.RunRef),
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalClosureTaskForTestV0(fixture.RunRef)),
		RequiredTestEvidenceStore:  orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(serviceOperationalClosureRequiredTestEvidenceForTestV0(fixture.RunRef)),
		OperationalClosureSource:   fixture.Source,
		OperationalPlanStateStore:  fixture.PlanStateStore,
		OperationalPlanStateWriter: fixture.PlanStateStore,
	}
}

func serviceBlockOperationalClosurePrerequisiteForTestV0(
	t *testing.T,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
	ports StartAppDirectorPortsV0,
	reason string,
	pendingOutbox int,
) {
	t.Helper()
	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OccurredAt:                 "2026-05-22T22:40:00Z",
			OperationalDirectorPlanRef: fixture.PlanRef,
		},
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status:             orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			PendingOutboxCount: pendingOutbox,
			Run:                fixture.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 block err=%v issues=%+v", err, issues)
	}
	if loop.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run no debe cerrar con %s: %+v", reason, loop.Run)
	}
	serviceAssertOperationalClosurePrerequisiteBlockedV0(t, fixture, reason)
}
