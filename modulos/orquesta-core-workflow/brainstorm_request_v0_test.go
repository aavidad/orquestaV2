package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandleRequestBrainstormCommandV0ReturnsEventAndNoOutbox(t *testing.T) {
	run := mustBrainstormActiveRunV0(t)
	command := mustRequestBrainstormCommandV0(t, "cmd-brainstorm-001", "idem-brainstorm-001", "brainstorm-request-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RequestBrainstorm: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventBrainstormRequestedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
}

func TestApplyBrainstormRequestedV0ProjectsRefOnce(t *testing.T) {
	run := mustBrainstormActiveRunV0(t)
	event := mustBrainstormRequestedEventV0(t, "evt-brainstorm-reducer-001", run.LastSequence+1, "brainstorm-request-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply BrainstormRequested: %v", err)
	}
	if !reflect.DeepEqual(got.Brainstorms, []string{"brainstorm-request-001"}) {
		t.Fatalf("brainstorms=%v, want [brainstorm-request-001]", got.Brainstorms)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Brainstorms, got.Brainstorms) {
		t.Fatalf("brainstorms duplicated: %v", again.Brainstorms)
	}
}

func TestReplayDurableEventsV0AcceptsBrainstormRequested(t *testing.T) {
	brainstorm := mustBrainstormRequestedEventWithKeyV0(t, "evt-durable-brainstorm", 3, "idem-brainstorm", "brainstorm-request-001")
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-brainstorm", 1, "idem-start-brainstorm"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-open-brainstorm", 2, "idem-open-brainstorm", OrchestrationPhaseBrainstormingArquitecturaV0),
		brainstorm,
		brainstorm,
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable BrainstormRequested: %v", err)
	}
	if got.LastSequence != 3 {
		t.Fatalf("last_sequence=%d, want 3", got.LastSequence)
	}
	if !reflect.DeepEqual(got.Brainstorms, []string{"brainstorm-request-001"}) {
		t.Fatalf("brainstorms=%v, want [brainstorm-request-001]", got.Brainstorms)
	}
}

func TestHandleRequestBrainstormCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustBrainstormActiveRunV0(t)
	command := mustRequestBrainstormCommandV0(t, "cmd-brainstorm-repeat", "idem-brainstorm-repeat", "brainstorm-request-repeat")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestHandleRequestBrainstormCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustBrainstormActiveRunV0(t)
	payload := validRequestBrainstormPayloadV0("brainstorm-request-conflict")
	command := mustRequestBrainstormCommandWithPayloadV0(t, "cmd-brainstorm-conflict", "idem-brainstorm-conflict", payload)
	created := mustApplySingleCommandEventV0(t, run, command)

	payload.TopicRef = "topic:arquitectura_seguridad"
	conflicting := mustRequestBrainstormCommandWithPayloadV0(t, "cmd-brainstorm-conflict", "idem-brainstorm-conflict", payload)
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestHandleRequestBrainstormCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustBrainstormActiveRunV0(t)
	command := mustRequestBrainstormCommandV0(t, "cmd-brainstorm-key", "idem-brainstorm-key", "brainstorm-request-key")
	created := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRequestBrainstormCommandV0(t, "cmd-brainstorm-key-2", "idem-brainstorm-key-2", "brainstorm-request-key")
	_, err := HandleCommandV0(created, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyBrainstormRequestedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustBrainstormActiveRunV0(t)
	first := mustBrainstormRequestedEventWithKeyV0(t, "evt-brainstorm-effect", run.LastSequence+1, "idem-brainstorm-effect", "brainstorm-request-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustBrainstormRequestedEventWithKeyV0(t, "evt-brainstorm-effect-conflict", applied.LastSequence+1, "idem-brainstorm-effect-conflict", "brainstorm-request-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRequestBrainstormCommandV0RejectsNonCurrentPhase(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustRequestBrainstormCommandV0(t, "cmd-brainstorm-phase", "idem-brainstorm-phase", "brainstorm-request-phase")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrTransicionInvalidaV0)
	}
}

func TestRequestBrainstormCommandV0RejectsNonBrainstormPhase(t *testing.T) {
	payload := validRequestBrainstormPayloadV0("brainstorm-request-wrong-phase")
	payload.PhaseID = string(OrchestrationPhaseProgramacionV0)

	_, err := NewRequestBrainstormCommandV0(validCommandMetaV0("cmd-brainstorm-wrong-phase", "idem-brainstorm-wrong-phase"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrTransicionInvalidaV0)
	}
}

func TestRequestBrainstormCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRequestBrainstormPayloadV0("brainstorm-request-forbidden")
	payload.Summary = "usar Codex para decidir proveedor"

	_, err := NewRequestBrainstormCommandV0(validCommandMetaV0("cmd-brainstorm-forbidden", "idem-brainstorm-forbidden"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestBrainstormRequestedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := brainstormRequestedPayloadFromCommandV0(validRequestBrainstormPayloadV0("brainstorm-request-event-forbidden"))
	payload.Summary = "usar Claude"

	_, err := NewBrainstormRequestedEventV0(reducerEventMetaV0("evt-brainstorm-forbidden", 3), payload)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestHandleRequestBrainstormCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-brainstorm", "idem-start-after-brainstorm")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding brainstorm command: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustBrainstormActiveRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	run := mustHandlerStartedRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-brainstorm-base", "idem-open-brainstorm-base", OrchestrationPhaseBrainstormingArquitecturaV0)
	return mustApplySingleCommandEventV0(t, run, open)
}

func mustRequestBrainstormCommandV0(t *testing.T, commandID string, idempotencyKey string, requestID string) OrchestrationCommandV0 {
	t.Helper()
	return mustRequestBrainstormCommandWithPayloadV0(t, commandID, idempotencyKey, validRequestBrainstormPayloadV0(requestID))
}

func mustRequestBrainstormCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RequestBrainstormCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRequestBrainstormCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustBrainstormRequestedEventV0(t *testing.T, eventID string, sequence int64, requestID string) OrchestrationEventV0 {
	t.Helper()
	return mustBrainstormRequestedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, requestID)
}

func mustBrainstormRequestedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, requestID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewBrainstormRequestedEventV0(meta, brainstormRequestedPayloadFromCommandV0(validRequestBrainstormPayloadV0(requestID)))
	return mustReducerEventV0(t, event, err)
}

func validRequestBrainstormPayloadV0(requestID string) RequestBrainstormCommandPayloadV0 {
	return RequestBrainstormCommandPayloadV0{
		BrainstormRequestID:        requestID,
		PhaseID:                    string(OrchestrationPhaseBrainstormingArquitecturaV0),
		TopicRef:                   "topic:arquitectura_inicial",
		Summary:                    "Comparar opciones de arquitectura hexagonal e i18n con evidencia compacta.",
		MinimumRecommendedCapacity: OrchestrationCapacityHighV0,
		EvidenceRefs:               []string{"docs/contratos_fases.md#brainstorming_arquitectura"},
	}
}
