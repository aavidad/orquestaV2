package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestRegisterAgentStopConfirmedV0ProjectsRefWithoutOutbox(t *testing.T) {
	run := mustRunWithStopRequestedV0(t, "agent-request-confirmed")
	command := mustRegisterAgentStopConfirmedCommandV0(t, "cmd-stop-confirmed", "idem-stop-confirmed", "agent-request-confirmed")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RegisterAgentStopConfirmed: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventAgentStopConfirmedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%+v, want none", result.Outbox)
	}
	got := mustApplyReducerEventV0(t, run, result.Events[0])
	if !reflect.DeepEqual(got.ConfirmedStoppedAgents, []string{"agent-request-confirmed"}) {
		t.Fatalf("confirmed_stopped_agents=%v", got.ConfirmedStoppedAgents)
	}
}

func TestRegisterAgentStopConfirmedV0RejectsWithoutStopRequested(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-not-stopped")
	command := mustRegisterAgentStopConfirmedCommandV0(t, "cmd-stop-confirmed-missing", "idem-stop-confirmed-missing", "agent-request-not-stopped")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 || publicErr.Field != "payload.agent_request_id" {
		t.Fatalf("error=%+v, want transicion_invalida payload.agent_request_id", publicErr)
	}
}

func TestReplayDurableEventsV0AcceptsAgentStopConfirmedAndDuplicate(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-stop-confirmed", 1, "idem-start-stop-confirmed"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-stop-confirmed", 2, "idem-phase-stop-confirmed", OrchestrationPhaseProgramacionV0),
		mustCapacityRequestedEventWithKeyV0(t, "evt-durable-capacity-before-stop-confirmed", 3, "idem-capacity-before-stop-confirmed", defaultAgentCapacityRequestIDV0),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision-before-stop-confirmed", 4, "idem-capacity-decision-before-stop-confirmed", defaultAgentCapacityRequestIDV0),
		mustAgentRequestedEventWithKeyV0(t, "evt-durable-agent-before-stop-confirmed", 5, "idem-agent-before-stop-confirmed", "agent-request-confirmed"),
		mustAgentStopRequestedEventWithKeyV0(t, "evt-durable-stop-before-confirmed", 6, "idem-stop-before-confirmed", "agent-request-confirmed"),
		mustAgentStopConfirmedEventWithKeyV0(t, "evt-durable-stop-confirmed", 7, "idem-stop-confirmed", "agent-request-confirmed"),
		mustAgentStopConfirmedEventWithKeyV0(t, "evt-durable-stop-confirmed", 7, "idem-stop-confirmed", "agent-request-confirmed"),
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable AgentStopConfirmed: %v", err)
	}
	if !reflect.DeepEqual(got.ConfirmedStoppedAgents, []string{"agent-request-confirmed"}) {
		t.Fatalf("confirmed_stopped_agents=%v", got.ConfirmedStoppedAgents)
	}
}

func TestRegisterAgentStopConfirmedV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustRunWithStopRequestedV0(t, "agent-request-confirmed")
	command := mustRegisterAgentStopConfirmedCommandV0(t, "cmd-stop-confirmed-repeat", "idem-stop-confirmed-repeat", "agent-request-confirmed")
	confirmed := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(confirmed, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestRegisterAgentStopConfirmedV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustRunWithStopRequestedV0(t, "agent-request-confirmed-effect")
	payload := validAgentStopConfirmedPayloadV0("agent-request-confirmed-effect")
	command := mustRegisterAgentStopConfirmedCommandWithPayloadV0(t, "cmd-stop-confirmed-effect", "idem-stop-confirmed-effect", payload)
	confirmed := mustApplySingleCommandEventV0(t, run, command)

	payload.ConfirmationRef = "confirmation-ref-changed"
	conflicting := mustRegisterAgentStopConfirmedCommandWithPayloadV0(t, "cmd-stop-confirmed-effect", "idem-stop-confirmed-effect", payload)
	_, err := HandleCommandV0(confirmed, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRegisterAgentStopConfirmedV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustRunWithStopRequestedV0(t, "agent-request-confirmed-key")
	command := mustRegisterAgentStopConfirmedCommandV0(t, "cmd-stop-confirmed-key", "idem-stop-confirmed-key", "agent-request-confirmed-key")
	confirmed := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRegisterAgentStopConfirmedCommandV0(t, "cmd-stop-confirmed-key-2", "idem-stop-confirmed-key-2", "agent-request-confirmed-key")
	_, err := HandleCommandV0(confirmed, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyAgentStopConfirmedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustRunWithStopRequestedV0(t, "agent-request-confirmed-event-effect")
	first := mustAgentStopConfirmedEventWithKeyV0(t, "evt-stop-confirmed-effect", run.LastSequence+1, "idem-stop-confirmed-effect", "agent-request-confirmed-event-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustAgentStopConfirmedEventWithKeyV0(t, "evt-stop-confirmed-effect-conflict", applied.LastSequence+1, "idem-stop-confirmed-effect-conflict", "agent-request-confirmed-event-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRegisterAgentStopConfirmedV0RejectsForbiddenDetails(t *testing.T) {
	payload := validAgentStopConfirmedPayloadV0("agent-request-forbidden")
	payload.Summary = "parada observada por runtime externo"

	_, err := NewRegisterAgentStopConfirmedCommandV0(validCommandMetaV0("cmd-stop-confirmed-forbidden", "idem-stop-confirmed-forbidden"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func mustRunWithStopRequestedV0(t *testing.T, requestID string) OrchestrationRunV0 {
	t.Helper()
	run := mustRunWithRequestedAgentV0(t, requestID)
	return mustApplySingleCommandEventV0(t, run, mustStopAgentCommandV0(t, "cmd-stop-"+requestID, "idem-stop-"+requestID, requestID))
}

func mustRegisterAgentStopConfirmedCommandV0(t *testing.T, commandID string, idempotencyKey string, requestID string) OrchestrationCommandV0 {
	t.Helper()
	return mustRegisterAgentStopConfirmedCommandWithPayloadV0(t, commandID, idempotencyKey, validAgentStopConfirmedPayloadV0(requestID))
}

func mustRegisterAgentStopConfirmedCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RegisterAgentStopConfirmedCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRegisterAgentStopConfirmedCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustAgentStopConfirmedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, requestID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewAgentStopConfirmedEventV0(meta, agentStopConfirmedPayloadFromCommandV0(validAgentStopConfirmedPayloadV0(requestID)))
	return mustReducerEventV0(t, event, err)
}

func validAgentStopConfirmedPayloadV0(requestID string) RegisterAgentStopConfirmedCommandPayloadV0 {
	return RegisterAgentStopConfirmedCommandPayloadV0{
		ConfirmationRef: "confirmation-ref-" + requestID,
		AgentRequestID:  requestID,
		ObservedAt:      "2026-05-06T10:30:00Z",
		Summary:         "Parada confirmada por el conector de agente.",
		EvidenceRefs:    []string{"docs/contratos_agentes.md#AgentStopConfirmed"},
	}
}
