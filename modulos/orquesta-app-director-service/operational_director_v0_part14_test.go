package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
	"testing"
)

func TestUpdateOperationalDirectorPlanStateAfterLoopV0TestsFailedEmiteQualityGateYReplanAutomatico(t *testing.T) {
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
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := eventSink.AppendRunEventsV0(context.Background(), fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 initial: %v", err)
	}
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T09:10:00Z",
		CorrelationID:              "corr-app-director-required-tests-auto-replan",
		OperationalDirectorPlanRef: fixture.PlanRef,
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		RequiredTestEvidenceStore:  testEvidenceStore,
	}
	loop := orquestacionnucleoapp.ProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Run:    fixture.Run,
	}

	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 first: %v", err)
	}
	request.OccurredAt = "2026-05-22T09:10:01Z"
	if err := updateOperationalDirectorPlanStateAfterLoopV0(context.Background(), request, ports, loop); err != nil {
		t.Fatalf("updateOperationalDirectorPlanStateAfterLoopV0 replay: %v", err)
	}

	state, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), fixture.RunRef, fixture.PlanRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0: %v", err)
	}
	testsStep := serviceOperationalDirectorPlanStateStepForTestV0(t, state, "step-run-required-tests")
	if state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		state.ActiveStepID != "step-run-required-tests" ||
		testsStep.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		testsStep.Reason != "required-tests-failed" ||
		serviceCountStringV0(testsStep.RequiredTestEvidenceRefs, fixture.RequiredTestEvidenceRef) != 1 ||
		serviceCountStringV0(testsStep.BlockerRefs, "required-tests-failed") != 1 ||
		serviceCountStringV0(state.EvidenceRefs, "evidence-ref-app-director-operational-plan-state-required-tests-failed-v0") != 1 {
		t.Fatalf("state=%+v testsStep=%+v", state, testsStep)
	}

	events := eventSink.EventsV0()
	var gateEvents, replanEvents int
	var gatePayload orquestacoreworkflow.QualityGateRecordedPayloadV0
	var replanPayload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
	for _, event := range events {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			gateEvents++
			if err := json.Unmarshal(event.Payload, &gatePayload); err != nil {
				t.Fatalf("quality gate payload: %v", err)
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEvents++
			if err := json.Unmarshal(event.Payload, &replanPayload); err != nil {
				t.Fatalf("replan payload: %v", err)
			}
		}
	}
	if gateEvents != 1 || replanEvents != 1 {
		t.Fatalf("gateEvents=%d replanEvents=%d events=%+v", gateEvents, replanEvents, events)
	}
	if gatePayload.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
		gatePayload.SubjectRef != fixture.TaskRef ||
		!serviceStringInSetV0(gatePayload.IssueRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("gatePayload=%+v", gatePayload)
	}
	if replanPayload.SourceRef != gatePayload.GateRef ||
		replanPayload.TaskRef != fixture.TaskRef ||
		replanPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		len(replanPayload.FollowupRefs) != 2 ||
		!strings.HasPrefix(replanPayload.FollowupRefs[0], "capacity-ref-app-director-required-tests-retry-") ||
		!strings.HasPrefix(replanPayload.FollowupRefs[1], "agent-ref-app-director-required-tests-retry-") {
		t.Fatalf("replanPayload=%+v gate=%+v", replanPayload, gatePayload)
	}
	run, err := runStore.LoadRunV0(context.Background(), fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.QualityGates) != 1 ||
		len(run.ReplanDecisions) != 1 ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		serviceStringInSetV0(run.CapacityRequests, replanPayload.FollowupRefs[0]) ||
		serviceStringInSetV0(run.Agents, replanPayload.FollowupRefs[1]) {
		t.Fatalf("run=%+v replanPayload=%+v", run, replanPayload)
	}

	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OperationalDirectorPlanRef: fixture.PlanRef,
		CorrelationID:              "corr-app-director-required-tests-auto-replan-reenter",
	}, ports)
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 auto replan: %v", err)
	}
	if continueRequestHasWaitScopeV0(reentered) {
		t.Fatalf("la reentrada debe dejar avanzar al scheduler sin wait scope prematuro: %+v", reentered)
	}
}
