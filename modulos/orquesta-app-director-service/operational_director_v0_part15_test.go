package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
	"testing"
)

func TestContinueAppDirectorV0TestsFailedReplanFollowupCierraSinDuplicarEventos(t *testing.T) {
	ctx := context.Background()
	fixture := serviceOperationalDirectorPlanStateReviewFixtureForTestV0(t, true)
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
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := eventSink.AppendRunEventsV0(ctx, fixture.RunRef, fixture.Events); err != nil {
		t.Fatalf("AppendRunEventsV0 seed: %v", err)
	}
	outboxLedger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(fixture.State)
	testEvidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(
		serviceOperationalDirectorRequiredTestEvidenceForFixtureV0(
			fixture,
			orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		),
	)
	executor := &fakeServiceRequiredTestCommandExecutorV0{
		results: map[string]orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			fixture.RequiredTest: {
				Status:       orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
				EvidenceRefs: []string{"artifact-ref-service-required-test-output-followup-001"},
			},
		},
	}
	closureSource := &serviceOperationalDirectorClosureSourceRefsForTestV0{
		TaskRef:           fixture.TaskRef,
		ValidationRef:     "validation-ref-service-required-tests-replan-followup-001",
		ClosureRef:        "closure-ref-service-required-tests-replan-followup-001",
		DeliveryRef:       "delivery-ref-app-director-required-tests-replan-followup",
		AcceptedReviewRef: "accepted-review-ref-app-director-required-tests-replan-followup",
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                eventSink,
		OutboxLedger:               outboxLedger,
		DirectorTaskStore:          orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceOperationalDirectorWorkflowTaskForFixtureV0(fixture)),
		RequiredTestEvidenceStore:  testEvidenceStore,
		RequiredTestRunner:         orquestacionnucleoapp.RequiredTestRunnerV0{Executor: executor, EvidenceWriter: testEvidenceStore},
		OperationalClosureSource:   closureSource,
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(runStore, eventSink, outboxLedger),
		},
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     fixture.RunRef,
		OccurredAt:                 "2026-05-22T23:40:00Z",
		CorrelationID:              "corr-service-required-tests-replan-followup-first",
		RequestedBy:                "orquesta-app-director-service-test",
		OperationalDirectorPlanRef: fixture.PlanRef,
		MaxBursts:                  4,
		MaxStepsPerBurst:           8,
		MaxDispatchesPerWait:       2,
		MaxCommands:                8,
		MaxOutboxPerCycle:          8,
		MaxExternalWaits:           1,
	}

	first, err := ContinueAppDirectorV0(ctx, request, ports)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0 first: %v", err)
	}
	if first.Run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("first no debe cerrar tras tests fallidos: %+v", first.Run)
	}
	if closureSource.Called {
		t.Fatalf("closure source no debe llamarse antes del replan: request=%+v", closureSource.LastRequest)
	}
	var gatePayload orquestacoreworkflow.QualityGateRecordedPayloadV0
	var replanPayload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
	var gateEvents, replanEvents int
	for _, event := range eventSink.EventsV0() {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			gateEvents++
			if err := json.Unmarshal(event.Payload, &gatePayload); err != nil {
				t.Fatalf("QualityGateRecorded payload: %v", err)
			}
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEvents++
			if err := json.Unmarshal(event.Payload, &replanPayload); err != nil {
				t.Fatalf("ReplanDecisionRecorded payload: %v", err)
			}
		}
	}
	if gateEvents != 1 || replanEvents != 1 {
		t.Fatalf("gateEvents=%d replanEvents=%d events=%+v", gateEvents, replanEvents, eventSink.EventsV0())
	}
	if gatePayload.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
		gatePayload.SubjectRef != fixture.TaskRef ||
		!serviceStringInSetV0(gatePayload.IssueRefs, fixture.RequiredTestEvidenceRef) {
		t.Fatalf("quality gate no causal del test fallido: %+v failed=%s", gatePayload, fixture.RequiredTestEvidenceRef)
	}
	if replanPayload.ReplanRef == "" || len(replanPayload.FollowupRefs) != 2 {
		t.Fatalf("replanPayload=%+v events=%+v", replanPayload, eventSink.EventsV0())
	}
	if replanPayload.SourceRef != gatePayload.GateRef ||
		replanPayload.TaskRef != fixture.TaskRef ||
		replanPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 {
		t.Fatalf("replan no cuelga del quality gate fallido: replan=%+v gate=%+v", replanPayload, gatePayload)
	}
	followupAgentRef := ""
	for _, ref := range replanPayload.FollowupRefs {
		if strings.HasPrefix(ref, "agent-ref-") {
			followupAgentRef = ref
		}
	}
	if followupAgentRef == "" {
		t.Fatalf("replan sin agente followup: %+v", replanPayload)
	}

	materializedRun, err := runStore.LoadRunV0(ctx, fixture.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 materialized: %v", err)
	}
	if len(materializedRun.QualityGates) != 1 ||
		len(materializedRun.ReplanDecisions) != 1 ||
		serviceStringInSetV0(materializedRun.Agents, followupAgentRef) ||
		materializedRun.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("run tras replan invalido: run=%+v followup=%s", materializedRun, followupAgentRef)
	}
	materializedRun.Agents = compactServiceRefsV0(append(materializedRun.Agents, followupAgentRef))
	materializedRun.StartedAgents = compactServiceRefsV0(append(materializedRun.StartedAgents, followupAgentRef))
	if err := runStore.SaveRunV0(ctx, materializedRun); err != nil {
		t.Fatalf("SaveRunV0 materialized: %v", err)
	}
	request.OccurredAt = "2026-05-22T23:40:01Z"
	request.CorrelationID = "corr-service-required-tests-replan-followup-wait"
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(ctx, request, ports)
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 followup: %v", err)
	}
	if !serviceStringInSetV0(reentered.WaitAgentRefs, followupAgentRef) ||
		serviceStringInSetV0(reentered.WaitAgentRefs, fixture.AgentRef) {
		t.Fatalf("wait scope no causal: request=%+v followup=%s old=%s", reentered, followupAgentRef, fixture.AgentRef)
	}

	serviceContinueRequiredTestsFailedReplanFollowupCloseForTestV0(t, ctx, fixture, request, ports, executor, eventSink, closureSource, followupAgentRef)
}
