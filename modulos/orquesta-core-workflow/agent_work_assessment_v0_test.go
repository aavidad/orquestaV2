package orquestacoreworkflow

import (
	"errors"
	"strings"
	"testing"
)

func TestHandleAssessAgentWorkCommandV0ContinuesWithoutOutbox(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-001")
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-001", "idem-assess-001", validAssessmentPayloadV0("assessment-001", "agent-request-001"))

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle AssessAgentWork: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventAgentWorkAssessedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
}

func TestHandleAssessAgentWorkCommandV0RejectsMissingAgent(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-missing-agent", "idem-assess-missing-agent", validAssessmentPayloadV0("assessment-missing-agent", "agent-request-missing"))

	_, err := HandleCommandV0(run, command)
	assertAssessmentCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.agent_request_id")
}

func TestHandleAssessAgentWorkCommandV0RejectsMissingDelivery(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-001")
	payload := validAssessmentPayloadV0("assessment-missing-delivery", "agent-request-001")
	payload.DeliveryRef = "delivery-missing"
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-missing-delivery", "idem-assess-missing-delivery", payload)

	_, err := HandleCommandV0(run, command)
	assertAssessmentCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.delivery_ref")
}

func TestReplayDurableEventsV0AcceptsAgentWorkAssessedAndExactDuplicate(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-assessment", 1, "idem-start-assessment"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-assessment", 2, "idem-phase-assessment", OrchestrationPhaseProgramacionV0),
		mustCapacityRequestedEventWithKeyV0(t, "evt-durable-capacity-assessment", 3, "idem-capacity-assessment", defaultAgentCapacityRequestIDV0),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision-assessment", 4, "idem-capacity-decision-assessment", defaultAgentCapacityRequestIDV0),
		mustAgentRequestedEventWithKeyV0(t, "evt-durable-agent-assessment", 5, "idem-agent-assessment", "agent-request-001"),
		mustAgentWorkAssessedEventWithKeyV0(t, "evt-durable-assessment", 6, "idem-assessment", "assessment-001", "agent-request-001"),
		mustAgentWorkAssessedEventWithKeyV0(t, "evt-durable-assessment", 6, "idem-assessment", "assessment-001", "agent-request-001"),
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable AgentWorkAssessed: %v", err)
	}
	if got.LastSequence != 6 {
		t.Fatalf("last_sequence=%d, want 6", got.LastSequence)
	}
	if len(got.AgentAssessments) != 1 ||
		AgentAssessmentProjectionIDV0(got.AgentAssessments[0]) != "assessment-001" {
		t.Fatalf("agent_assessments=%v", got.AgentAssessments)
	}
}

func TestAssessAgentWorkCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-repeat")
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-repeat", "idem-assess-repeat", validAssessmentPayloadV0("assessment-repeat", "agent-request-repeat"))
	assessed := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(assessed, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestAssessAgentWorkCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-assess-conflict")
	payload := validAssessmentPayloadV0("assessment-conflict", "agent-request-assess-conflict")
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-conflict", "idem-assess-conflict", payload)
	assessed := mustApplySingleCommandEventV0(t, run, command)

	payload.Severity = AgentAssessmentSeverityHighV0
	conflicting := mustAssessAgentWorkCommandV0(t, "cmd-assess-conflict", "idem-assess-conflict", payload)
	result, err := HandleCommandV0(assessed, conflicting)

	if len(result.Events) != 0 || len(result.Outbox) != 0 {
		t.Fatalf("conflict emitted result: %+v", result)
	}
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestAssessAgentWorkCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-assess-idem")
	payload := validAssessmentPayloadV0("assessment-idem", "agent-request-assess-idem")
	command := mustAssessAgentWorkCommandV0(t, "cmd-assess-idem", "idem-assess-idem", payload)
	assessed := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustAssessAgentWorkCommandV0(t, "cmd-assess-idem-other", "idem-assess-idem-other", payload)
	_, err := HandleCommandV0(assessed, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyAgentWorkAssessedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-assess-effect")
	first := mustAgentWorkAssessedEventWithKeyV0(t, "evt-assess-effect", run.LastSequence+1, "idem-assess-effect", "assessment-effect", "agent-request-assess-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustAgentWorkAssessedEventWithKeyV0(t, "evt-assess-effect-conflict", applied.LastSequence+1, "idem-assess-effect-conflict", "assessment-effect", "agent-request-assess-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestAssessAgentWorkCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validAssessmentPayloadV0("assessment-forbidden", "agent-request-forbidden")
	payload.Summary = "authorization: Bearer abc123"

	_, err := NewAssessAgentWorkCommandV0(validCommandMetaV0("cmd-assess-forbidden", "idem-assess-forbidden"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestAssessAgentWorkCommandV0RejectsStopActionForAcceptableWork(t *testing.T) {
	payload := validAssessmentPayloadV0("assessment-stop-acceptable", "agent-request-001")
	payload.Action = AgentAssessmentActionStopAgentV0

	_, err := NewAssessAgentWorkCommandV0(validCommandMetaV0("cmd-assess-stop-acceptable", "idem-assess-stop-acceptable"), payload)
	assertAssessmentCommandErrorV0(t, err, ErrPayloadInvalidoV0, "payload.action")
}

func TestApplyAgentWorkAssessedV0AcceptsExistingDeliveryRef(t *testing.T) {
	run := mustRunWithRegisteredDeliveryV0(t)
	event := mustAgentWorkAssessedEventWithDeliveryV0(t, "evt-assess-delivery", run.LastSequence+1, "assessment-delivery", "agent-request-001", "delivery-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply AgentWorkAssessed: %v", err)
	}
	if len(got.AgentAssessments) != 1 ||
		AgentAssessmentProjectionIDV0(got.AgentAssessments[0]) != "assessment-delivery" {
		t.Fatalf("agent_assessments=%v", got.AgentAssessments)
	}
}

func mustRunWithRegisteredDeliveryV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustDeliveryReadyRunV0(t)
	return mustApplySingleCommandEventV0(t, run, mustRegisterDeliveryCommandV0(t, "cmd-delivery-for-assessment", "idem-delivery-for-assessment", "delivery-001"))
}

func mustAssessAgentWorkCommandV0(t *testing.T, commandID string, idempotencyKey string, payload AssessAgentWorkCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewAssessAgentWorkCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustAgentWorkAssessedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, assessmentRef string, agentRequestID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewAgentWorkAssessedEventV0(meta, agentWorkAssessedPayloadFromCommandV0(validAssessmentPayloadV0(assessmentRef, agentRequestID)))
	return mustReducerEventV0(t, event, err)
}

func mustAgentWorkAssessedEventWithDeliveryV0(t *testing.T, eventID string, sequence int64, assessmentRef string, agentRequestID string, deliveryRef string) OrchestrationEventV0 {
	t.Helper()
	payload := validAssessmentPayloadV0(assessmentRef, agentRequestID)
	payload.DeliveryRef = deliveryRef
	meta := reducerEventMetaV0(eventID, sequence)
	event, err := NewAgentWorkAssessedEventV0(meta, agentWorkAssessedPayloadFromCommandV0(payload))
	return mustReducerEventV0(t, event, err)
}

func validAssessmentPayloadV0(assessmentRef string, agentRequestID string) AssessAgentWorkCommandPayloadV0 {
	return AssessAgentWorkCommandPayloadV0{
		AssessmentRef:  assessmentRef,
		PhaseID:        string(OrchestrationPhaseProgramacionV0),
		AgentRequestID: agentRequestID,
		TaskRef:        "task-ncw-009",
		Verdict:        AgentAssessmentVerdictAcceptableV0,
		Action:         AgentAssessmentActionContinueV0,
		Severity:       AgentAssessmentSeverityLowV0,
		Summary:        "El avance registrado es coherente con la microtarea.",
		EvidenceRefs:   []string{"docs/contratos_agentes.md#AssessAgentWork"},
	}
}

func eventTypesV0(events []OrchestrationEventV0) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	return types
}

func assertAssessmentCommandErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code || strings.TrimSpace(publicErr.Field) != field {
		t.Fatalf("error=%+v, want %s %s", publicErr, code, field)
	}
}
