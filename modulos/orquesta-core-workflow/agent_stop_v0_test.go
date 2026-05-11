package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestHandleStopAgentCommandV0ReturnsEventAndOutbox(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-001")
	command := mustStopAgentCommandV0(t, "cmd-stop-001", "idem-stop-001", "agent-request-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle StopAgent: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventAgentStopRequestedV0)
	if len(result.Outbox) != 1 {
		t.Fatalf("outbox=%d, want 1", len(result.Outbox))
	}
	outbox := result.Outbox[0]
	if outbox.MessageType != OutboxMessageStopRuntimeAgentV0 || outbox.TargetPort != OutboxTargetAgentLauncherV0 {
		t.Fatalf("unexpected outbox envelope: %+v", outbox)
	}
	if outbox.CausationEventID != result.Events[0].EventID {
		t.Fatalf("causation_event_id=%q, want %q", outbox.CausationEventID, result.Events[0].EventID)
	}
	var payload StopRuntimeAgentRequestV0
	if err := json.Unmarshal(outbox.Payload, &payload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if payload.AgentRequestID != "agent-request-001" || payload.RunID != command.RunID {
		t.Fatalf("unexpected outbox payload: %+v", payload)
	}
	if payload.ReasonCode != "tarea_cancelada" {
		t.Fatalf("reason_code=%q, want tarea_cancelada", payload.ReasonCode)
	}
}

func TestApplyAgentStopRequestedV0ProjectsRefOnce(t *testing.T) {
	run := mustReducerRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	run = mustApplyReducerEventV0(t, run, mustAgentRequestedEventV0(t, "evt-agent-stop-base", run.LastSequence+1, "agent-request-001"))
	event := mustAgentStopRequestedEventV0(t, "evt-agent-stop-reducer-001", run.LastSequence+1, "agent-request-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply AgentStopRequested: %v", err)
	}
	if !reflect.DeepEqual(got.StoppedAgents, []string{"agent-request-001"}) {
		t.Fatalf("stopped_agents=%v, want [agent-request-001]", got.StoppedAgents)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.StoppedAgents, got.StoppedAgents) {
		t.Fatalf("stopped_agents duplicated: %v", again.StoppedAgents)
	}
}

func TestApplyAgentStopRequestedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-stop-effect")
	event := mustAgentStopRequestedEventV0(t, "evt-stop-effect-first", run.LastSequence+1, "agent-request-stop-effect")
	applied := mustApplyReducerEventV0(t, run, event)
	conflict := mustAgentStopRequestedEventWithKeyV0(t, "evt-stop-effect-second", applied.LastSequence+1, "idem-stop-effect-second", "agent-request-stop-effect")

	_, err := ApplyEventV0(applied, conflict)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrEventoConflictivoV0 || publicErr.Field != "idempotency" {
		t.Fatalf("error=%+v, want evento_conflictivo idempotency", publicErr)
	}
}

func TestReplayDurableEventsV0AcceptsAgentStopRequestedAndExactDuplicate(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-stop-agent", 1, "idem-start-stop-agent"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-stop-agent", 2, "idem-phase-stop-agent", OrchestrationPhaseProgramacionV0),
		mustCapacityRequestedEventWithKeyV0(t, "evt-durable-capacity-before-stop", 3, "idem-capacity-before-stop", defaultAgentCapacityRequestIDV0),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision-before-stop", 4, "idem-capacity-decision-before-stop", defaultAgentCapacityRequestIDV0),
		mustAgentRequestedEventWithKeyV0(t, "evt-durable-agent-before-stop", 5, "idem-agent-before-stop", "agent-request-001"),
		mustAgentStopRequestedEventWithKeyV0(t, "evt-durable-stop-agent", 6, "idem-stop-agent", "agent-request-001"),
		mustAgentStopRequestedEventWithKeyV0(t, "evt-durable-stop-agent", 6, "idem-stop-agent", "agent-request-001"),
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable AgentStopRequested: %v", err)
	}
	if got.LastSequence != 6 {
		t.Fatalf("last_sequence=%d, want 6", got.LastSequence)
	}
	if !reflect.DeepEqual(got.StoppedAgents, []string{"agent-request-001"}) {
		t.Fatalf("stopped_agents=%v, want [agent-request-001]", got.StoppedAgents)
	}
}

func TestHandleStopAgentCommandV0RejectsMissingAgent(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustStopAgentCommandV0(t, "cmd-stop-missing", "idem-stop-missing", "agent-request-missing")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 || publicErr.Field != "payload.agent_request_id" {
		t.Fatalf("error=%+v, want transicion_invalida payload.agent_request_id", publicErr)
	}
}

func TestHandleStopAgentCommandV0RetriesPendingOutboxAfterPartialPersist(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-001")
	command := mustStopAgentCommandV0(t, "cmd-stop-repeat", "idem-stop-repeat", "agent-request-001")
	stopped := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(stopped, command)
	if err != nil {
		t.Fatalf("retry StopAgent after partial persist: %v", err)
	}
	if len(result.Events) != 0 || len(result.Outbox) != 1 {
		t.Fatalf("result=%+v, want no events and one outbox", result)
	}
	if result.Outbox[0].MessageType != OutboxMessageStopRuntimeAgentV0 {
		t.Fatalf("unexpected stop retry outbox: %+v", result.Outbox[0])
	}
}

func TestHandleStopAgentCommandV0RepeatedAfterConfirmedDoesNotDuplicate(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-confirmed-repeat")
	command := mustStopAgentCommandV0(t, "cmd-stop-confirmed-repeat-base", "idem-stop-confirmed-repeat-base", "agent-request-confirmed-repeat")
	stopped := mustApplySingleCommandEventV0(t, run, command)
	confirm := mustRegisterAgentStopConfirmedCommandV0(t, "cmd-stop-confirmed-repeat", "idem-stop-confirmed-repeat", "agent-request-confirmed-repeat")
	confirmed := mustApplySingleCommandEventV0(t, stopped, confirm)

	result, err := HandleCommandV0(confirmed, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestHandleStopAgentCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-stop-key-conflict")
	command := mustStopAgentCommandV0(t, "cmd-stop-key-conflict", "idem-stop-key-conflict", "agent-request-stop-key-conflict")
	stopped := mustApplySingleCommandEventV0(t, run, command)
	conflict := mustStopAgentCommandV0(t, "cmd-stop-key-conflict-2", "idem-stop-key-conflict-2", "agent-request-stop-key-conflict")

	_, err := HandleCommandV0(stopped, conflict)
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestAgentStopOutboxPayloadDoesNotContainForbiddenDetails(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-clean")
	command := mustStopAgentCommandV0(t, "cmd-stop-clean", "idem-stop-clean", "agent-request-clean")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle StopAgent: %v", err)
	}
	serialized := strings.ToLower(string(result.Outbox[0].Payload))
	for _, forbidden := range []string{"provider", "proveedor", "model", "modelo", "runtime", "home", "oauth", "codex", "claude", "ollama", "vllm"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("outbox payload contains forbidden fragment %q: %s", forbidden, serialized)
		}
	}
}

func TestStopAgentCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validStopAgentPayloadV0("agent-request-forbidden")
	payload.Summary = "cerrar provider externo"

	_, err := NewStopAgentCommandV0(validCommandMetaV0("cmd-stop-forbidden", "idem-stop-forbidden"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestValidateStopRuntimeAgentOutboxRejectsInvalidPayload(t *testing.T) {
	message := OutboxMessageV0{
		MessageID:      "outbox-stop-invalid",
		MessageType:    OutboxMessageStopRuntimeAgentV0,
		RunID:          "run-001",
		IdempotencyKey: "idem-stop-invalid",
		TargetPort:     OutboxTargetAgentLauncherV0,
		PayloadVersion: OutboxPayloadVersionV0,
		Payload:        json.RawMessage(`{}`),
	}

	err := ValidateOutboxMessageV0(message)
	var publicErr OutboxMessageErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public outbox error, got %T %v", err, err)
	}
	if publicErr.Code != ErrOutboxPayloadInvalidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrOutboxPayloadInvalidoV0)
	}
}

func mustRunWithRequestedAgentV0(t *testing.T, requestID string) OrchestrationRunV0 {
	t.Helper()
	run := mustRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	return mustApplySingleCommandEventV0(t, run, mustRequestAgentCommandV0(t, "cmd-"+requestID, "idem-"+requestID, requestID))
}

func mustStopAgentCommandV0(t *testing.T, commandID string, idempotencyKey string, requestID string) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewStopAgentCommandV0(validCommandMetaV0(commandID, idempotencyKey), validStopAgentPayloadV0(requestID))
	return mustCommandV0(t, command, err)
}

func mustAgentStopRequestedEventV0(t *testing.T, eventID string, sequence int64, requestID string) OrchestrationEventV0 {
	t.Helper()
	return mustAgentStopRequestedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, requestID)
}

func mustAgentStopRequestedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, requestID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewAgentStopRequestedEventV0(meta, agentStopRequestedPayloadFromCommandV0(validStopAgentPayloadV0(requestID)))
	return mustReducerEventV0(t, event, err)
}

func validStopAgentPayloadV0(requestID string) StopAgentCommandPayloadV0 {
	return StopAgentCommandPayloadV0{
		AgentRequestID: requestID,
		ReasonCode:     "tarea_cancelada",
		Summary:        "Detener el trabajo solicitado por cambio de prioridad.",
		EvidenceRefs:   []string{"docs/contratos_agentes.md#StopAgent"},
	}
}
