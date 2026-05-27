package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
	"testing"
)

func TestContinueRequestWithOperationalDirectorPlanStateV0RecuperaDeliveryTardiaTrasWaitExpirado(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, false)
	fixture.Run.Reviews = nil
	fixture.Run.ReviewResults = nil
	fixture.Run.AcceptedReviews = nil
	state := fixture.State
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.ActiveStepID = "step-wait-subagents"
	state.PendingAgentRefs = nil
	state.BlockerRefs = []string{"external-wait-exhausted"}
	state.ClosureReason = "external-wait-exhausted"
	for index := range state.Steps {
		step := &state.Steps[index]
		switch step.StepID {
		case "step-wait-subagents":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
			step.PendingAgentRefs = nil
			step.BlockerRefs = []string{"external-wait-exhausted"}
			step.Reason = "external-wait-exhausted"
		case "step-review-deliveries":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepPendingV0
			step.DeliveryRefs = nil
			step.ReviewResultRefs = nil
			step.AcceptedReviewRefs = nil
			step.Reason = ""
		}
	}
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)

	_, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T18:24:59Z",
		CorrelationID:              "corr-app-director-late-delivery-after-expired-wait-no-writer",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                  runStore,
		OperationalPlanStateStore: planStore,
	})
	if err == nil {
		t.Fatalf("err nil sin writer")
	}
	issue, ok := err.(AppDirectorServiceIssueV0)
	if !ok || issue.Field != "ports.operational_plan_state_writer" {
		t.Fatalf("err=%T %#v", err, err)
	}

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T18:25:00Z",
		CorrelationID:              "corr-app-director-late-delivery-after-expired-wait",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}, StartAppDirectorPortsV0{
		RunStore:                   runStore,
		OperationalPlanStateStore:  planStore,
		OperationalPlanStateWriter: planStore,
	})
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("reentered=%+v agent=%s", reentered, fixture.AgentRef)
	}
	recovered, err := planStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	waitStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-wait-subagents")
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, recovered, "step-review-deliveries")
	if recovered.Status != orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0 ||
		recovered.ActiveStepID != "step-review-deliveries" ||
		recovered.ClosureReason != "" ||
		waitStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepRunningV0 ||
		!serviceStringInSetV0(recovered.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-wait-expired-late-delivery-v0") {
		t.Fatalf("recovered=%+v wait=%+v review=%+v", recovered, waitStep, reviewStep)
	}
}

func TestOperationalDirectorPlanStatePostLoopScopeV0RecargaRunStoreAntesDeCerrarWait(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, false)
	state := fixture.State
	state.ActiveStepID = "step-wait-subagents"
	state.PendingAgentRefs = []string{fixture.AgentRef}
	for index := range state.Steps {
		step := &state.Steps[index]
		switch step.StepID {
		case "step-wait-subagents":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			step.PendingAgentRefs = []string{fixture.AgentRef}
			step.Reason = "wait-subagents-running"
		case "step-review-deliveries":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepPendingV0
			step.DeliveryRefs = nil
			step.ReviewResultRefs = nil
			step.AcceptedReviewRefs = nil
			step.Reason = ""
		}
	}
	latest := fixture.Run
	latest.DeliveredAgents = nil
	latest.DeliveredTasks = nil
	latest.Deliveries = nil
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(latest)
	planStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	staleLoop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0,
		Run: orquestacoreworkflow.OrchestrationRunV0{
			SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
			RunID:         fixture.RunRef,
		},
	}

	got, err := operationalDirectorPlanStatePostLoopScopeV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{RunStore: runStore, OperationalPlanStateStore: planStore},
		staleLoop,
		orquestacionnucleoapp.ProgressiveLoopRequestV0{RunRef: fixture.RunRef},
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStatePostLoopScopeV0 pending: %v", err)
	}
	if got.Status != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("scope cerro con run stale: got=%+v latest=%+v", got, latest)
	}

	latest.DeliveredAgents = []string{fixture.AgentRef}
	if err := runStore.SaveRunV0(context.Background(), latest); err != nil {
		t.Fatalf("SaveRunV0 delivered: %v", err)
	}
	got, err = operationalDirectorPlanStatePostLoopScopeV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{RunStore: runStore, OperationalPlanStateStore: planStore},
		staleLoop,
		orquestacionnucleoapp.ProgressiveLoopRequestV0{RunRef: fixture.RunRef},
	)
	if err != nil {
		t.Fatalf("operationalDirectorPlanStatePostLoopScopeV0 delivered: %v", err)
	}
	if got.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		!serviceStringInSetV0(got.Run.DeliveredAgents, fixture.AgentRef) {
		t.Fatalf("scope no cerro con delivery latest: got=%+v", got)
	}
}

func TestContinueAppDirectorV0AvanzaDeWaitAReviewAbriendoRevision(t *testing.T) {
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
	run := serviceContinueClosureRunForTestV0(fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.ProjectRef = fixture.Run.ProjectRef
	run.AppSpecRef = fixture.Run.AppSpecRef
	run.Tasks = []string{fixture.TaskRef}
	run.Agents = []string{fixture.AgentRef}
	run.StartedAgents = []string{fixture.AgentRef}
	run.DeliveredAgents = []string{fixture.AgentRef}
	run.DeliveredTasks = []string{fixture.TaskRef}
	run.Deliveries = []string{fixture.DeliveryRef}
	run.LastEventID = "event-ref-delivery-review-autofollow-001"
	run.LastSequence = 1
	state := fixture.State
	state.ActiveStepID = "step-wait-subagents"
	state.PendingAgentRefs = []string{fixture.AgentRef}
	for index := range state.Steps {
		step := &state.Steps[index]
		switch step.StepID {
		case "step-wait-subagents":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepRunningV0
			step.PendingAgentRefs = []string{fixture.AgentRef}
			step.Reason = "wait-subagents-running"
		case "step-review-deliveries":
			step.Status = orquestadirectoroperativo.OperationalDirectorStepPendingV0
			step.DeliveryRefs = nil
			step.ReviewResultRefs = nil
			step.AcceptedReviewRefs = nil
			step.Reason = ""
		}
	}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := eventSink.AppendRunEventsV0(context.Background(), fixture.RunRef, []orquestacoreworkflow.OrchestrationEventV0{fixture.Events[0]}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	outboxLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	reviewSource := &acceptedServiceReviewGateSourceV0{Fixture: fixture}

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T13:00:00Z",
		CorrelationID:              "corr-service-wait-review-open-revision-001",
		OperationalDirectorPlanRef: fixture.PlanRef,
		MaxBursts:                  6,
		MaxStepsPerBurst:           6,
		MaxDispatchesPerWait:       2,
		MaxCommands:                20,
		MaxOutboxPerCycle:          8,
	}, StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                eventSink,
		OutboxLedger:               outboxLedger,
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)),
		ReviewGateSource:           reviewSource,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outboxLedger),
		},
	})
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if !reviewSource.Called ||
		result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 ||
		!serviceStringInSetV0(result.Run.Reviews, fixture.ReviewRequestID) ||
		!strings.Contains(strings.Join(result.Run.ReviewResults, " "), fixture.ReviewResultRef) ||
		!serviceStringInSetV0(result.Run.AcceptedReviews, fixture.AcceptedReviewRef) {
		t.Fatalf("review no aplicada: called=%v run=%+v", reviewSource.Called, result.Run)
	}
	state, err = planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	reviewStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-review-deliveries")
	if state.ActiveStepID != "step-run-required-tests" ||
		reviewStep.Status != orquestadirectoroperativo.OperationalDirectorStepAcceptedV0 ||
		!serviceStringInSetV0(reviewStep.AcceptedReviewRefs, fixture.AcceptedReviewRef) {
		t.Fatalf("state=%+v reviewStep=%+v", state, reviewStep)
	}
}
