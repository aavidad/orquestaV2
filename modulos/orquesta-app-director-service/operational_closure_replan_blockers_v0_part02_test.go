package orquestaappdirectorservice

import (
	"context"
	"encoding/json"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueRequestWithOperationalDirectorPlanStateV0EmiteReplanSiClosureIssuesBloqueadoSinDecision(t *testing.T) {
	runRef := "run-service-operational-closure-issues-missing-decision"
	planRef := "plan-ref-service-operational-closure-issues-missing-decision"
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
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.ClosureReason = "operational-closure-issues"
	state.BlockerRefs = []string{"operational-closure-issues", "validation_ref", "operational_closure_insufficient"}
	for index := range state.Steps {
		if state.Steps[index].StepID != "step-replan-or-close" {
			continue
		}
		state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
		state.Steps[index].Reason = "operational-closure-issues"
		state.Steps[index].BlockerRefs = []string{"operational-closure-issues", "validation_ref", "operational_closure_insufficient"}
	}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T22:20:00Z",
		CorrelationID:              "corr-service-operational-closure-issues-missing-decision",
		RequestedBy:                "test",
		OperationalDirectorPlanRef: planRef,
	}

	if _, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, ports); err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 first: %v", err)
	}

	var gateEvents, replanEvents int
	var gatePayload orquestacoreworkflow.QualityGateRecordedPayloadV0
	var replanPayload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
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
	if gatePayload.SubjectRef != taskRef ||
		gatePayload.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
		!serviceStringInSetV0(gatePayload.IssueRefs, "validation_ref") ||
		!serviceStringInSetV0(gatePayload.IssueRefs, "operational_closure_insufficient") {
		t.Fatalf("gatePayload=%+v", gatePayload)
	}
	if replanPayload.SourceRef != gatePayload.GateRef ||
		replanPayload.TaskRef != taskRef ||
		replanPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		len(replanPayload.FollowupRefs) != 2 {
		t.Fatalf("replanPayload=%+v gatePayload=%+v", replanPayload, gatePayload)
	}
	stateAfterFirst, err := planStateStore.LoadOperationalDirectorPlanStateV0(context.Background(), runRef, planRef)
	if err != nil {
		t.Fatalf("LoadOperationalDirectorPlanStateV0 first: %v", err)
	}
	stepAfterFirst := serviceOperationalDirectorPlanStateStepForTestV0(t, stateAfterFirst, "step-replan-or-close")
	if stateAfterFirst.Status != orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0 ||
		stepAfterFirst.Status != orquestadirectoroperativo.OperationalDirectorStepBlockedV0 ||
		stepAfterFirst.Reason != "operational-closure-issues" {
		t.Fatalf("stateAfterFirst=%+v step=%+v", stateAfterFirst, stepAfterFirst)
	}

	if _, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, ports); err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 second: %v", err)
	}
	var gateEventsAfterSecond, replanEventsAfterSecond int
	for _, event := range eventSink.EventsV0() {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			gateEventsAfterSecond++
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEventsAfterSecond++
		}
	}
	if gateEventsAfterSecond != 1 || replanEventsAfterSecond != 1 {
		t.Fatalf("eventos duplicados gate=%d replan=%d events=%+v", gateEventsAfterSecond, replanEventsAfterSecond, eventSink.EventsV0())
	}
}

func TestContinueRequestWithOperationalDirectorPlanStateV0EmiteReplanParaBlockerReparableNoCubierto(t *testing.T) {
	runRef := "run-service-operational-closure-artifact-blocker"
	planRef := "plan-ref-service-operational-closure-artifact-blocker"
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
	state.Status = orquestacionnucleoapp.OperationalDirectorPlanStateBlockedV0
	state.ClosureReason = "operational-closure-issues"
	state.BlockerRefs = []string{"operational-closure-issues", "artifact_refs"}
	for index := range state.Steps {
		if state.Steps[index].StepID != "step-replan-or-close" {
			continue
		}
		state.Steps[index].Status = orquestadirectoroperativo.OperationalDirectorStepBlockedV0
		state.Steps[index].Reason = "operational-closure-issues"
		state.Steps[index].BlockerRefs = []string{"operational-closure-issues", "artifact_refs"}
	}
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	ports := StartAppDirectorPortsV0{
		RunStore:                   runStore,
		EventSink:                  eventSink,
		EventReader:                newServiceOperationalClosureEventReaderForTestV0(t, runRef),
		OperationalPlanStateStore:  planStateStore,
		OperationalPlanStateWriter: planStateStore,
	}
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-05-22T22:45:00Z",
		CorrelationID:              "corr-service-operational-closure-artifact-blocker",
		RequestedBy:                "test",
		OperationalDirectorPlanRef: planRef,
	}

	if _, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, ports); err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 first: %v", err)
	}

	var gateEvents, replanEvents int
	var gatePayload orquestacoreworkflow.QualityGateRecordedPayloadV0
	var replanPayload orquestacoreworkflow.ReplanDecisionRecordedPayloadV0
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
	if gatePayload.SubjectRef != taskRef ||
		gatePayload.Decision != orquestacoreworkflow.QualityGateDecisionBlockedV0 ||
		!serviceStringInSetV0(gatePayload.IssueRefs, "artifact_refs") ||
		!serviceStringInSetV0(gatePayload.IssueRefs, "operational_closure_insufficient") {
		t.Fatalf("gatePayload=%+v", gatePayload)
	}
	if replanPayload.SourceRef != gatePayload.GateRef ||
		replanPayload.TaskRef != taskRef ||
		replanPayload.AcceptedAction != orquestacoreworkflow.ReplanDecisionActionRetryTaskV0 ||
		len(replanPayload.FollowupRefs) != 2 {
		t.Fatalf("replanPayload=%+v gatePayload=%+v", replanPayload, gatePayload)
	}

	if _, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, ports); err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0 second: %v", err)
	}
	var gateEventsAfterSecond, replanEventsAfterSecond int
	for _, event := range eventSink.EventsV0() {
		switch event.EventType {
		case orquestacoreworkflow.OrchestrationEventQualityGateRecordedV0:
			gateEventsAfterSecond++
		case orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0:
			replanEventsAfterSecond++
		}
	}
	if gateEventsAfterSecond != 1 || replanEventsAfterSecond != 1 {
		t.Fatalf("eventos duplicados gate=%d replan=%d events=%+v", gateEventsAfterSecond, replanEventsAfterSecond, eventSink.EventsV0())
	}
}
