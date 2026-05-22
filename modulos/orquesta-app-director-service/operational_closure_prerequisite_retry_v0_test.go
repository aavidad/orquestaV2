package orquestaappdirectorservice

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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

func serviceReopenAndCloseOperationalClosurePrerequisiteForTestV0(
	t *testing.T,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
) {
	t.Helper()
	ports := fixture.fullPortsV0(t)
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:                     fixture.RunRef,
			OccurredAt:                 "2026-05-22T22:40:01Z",
			OperationalDirectorPlanRef: fixture.PlanRef,
		},
		ports,
	)
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-ref-service-operational-closure-001")
	if !serviceStringInSetV0(reentered.WaitAgentRefs, agentRef) {
		t.Fatalf("reentered=%+v falta agent=%s", reentered, agentRef)
	}
	serviceAssertOperationalClosurePrerequisiteReopenedV0(t, fixture)

	run, err := fixture.RunStore.LoadRunV0(context.Background(), fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		reentered,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 close err=%v issues=%+v", err, issues)
	}
	if loop.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(loop.Run.ClosedTasks) != 1 ||
		len(loop.Run.Validations) != 1 ||
		len(loop.Run.Closures) != 1 {
		t.Fatalf("run no cerrado tras retry: %+v", loop.Run)
	}
	state, err := fixture.PlanStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 closed: %v", err)
	}
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 ||
		state.ClosureReason != "operational-closure-succeeded" {
		t.Fatalf("state no cerrado tras retry: %+v", state)
	}
}

func serviceAssertOperationalClosurePrerequisiteBlockedV0(
	t *testing.T,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
	reason string,
) {
	t.Helper()
	state, err := fixture.PlanStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 blocked: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ClosureReason != reason ||
		!serviceStringInSetV0(state.BlockerRefs, reason) ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		step.Reason != reason ||
		!serviceStringInSetV0(step.BlockerRefs, reason) {
		t.Fatalf("state no bloqueado por %s: state=%+v step=%+v", reason, state, step)
	}
}

func serviceAssertOperationalClosurePrerequisiteReopenedV0(
	t *testing.T,
	fixture serviceOperationalClosurePrerequisiteRetryFixtureV0,
) {
	t.Helper()
	state, err := fixture.PlanStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 reopened: %v", err)
	}
	step := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-replan-or-close")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		state.ClosureReason != "" ||
		len(state.BlockerRefs) != 0 ||
		step.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		step.Reason != "operational-closure-prerequisite-ready" ||
		len(step.BlockerRefs) != 0 ||
		!serviceStringInSetV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-closure-prerequisite-ready-v0") {
		t.Fatalf("state no reabierto: state=%+v step=%+v", state, step)
	}
}
