package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestHandleClosePhaseCommandV0ReturnsPhaseClosedAndNoOutbox(t *testing.T) {
	run := mustOpenedPhaseRunV0(t, OrchestrationPhaseProgramacionV0)
	command := mustClosePhaseCommandV0(t, "cmd-close-001", "idem-close-001", OrchestrationPhaseProgramacionV0, "closure:programacion")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle ClosePhase: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventPhaseClosedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	var payload PhaseClosedPayloadV0
	if err := json.Unmarshal(result.Events[0].Payload, &payload); err != nil {
		t.Fatalf("decode PhaseClosed payload: %v", err)
	}
	if payload.PhaseID != string(OrchestrationPhaseProgramacionV0) || payload.ClosureRef != "closure:programacion" {
		t.Fatalf("payload inesperado: %+v", payload)
	}
	if !reflect.DeepEqual(payload.EvidenceRefs, []string{"docs/pruebas.md#phase-close", "docs/decisiones.md#phase-close"}) {
		t.Fatalf("evidence_refs=%v", payload.EvidenceRefs)
	}
}

func TestApplyPhaseClosedEventV0ClosesCurrentPhase(t *testing.T) {
	run := mustOpenedPhaseRunV0(t, OrchestrationPhaseProgramacionV0)
	opened := reducerPhaseByIDV0(t, run, OrchestrationPhaseProgramacionV0).OpenedAt
	event := mustPhaseClosedEventV0(t, "evt-phase-closed-reducer-001", run.LastSequence+1, OrchestrationPhaseProgramacionV0, "closure:programacion")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply PhaseClosed: %v", err)
	}

	closed := reducerPhaseByIDV0(t, got, OrchestrationPhaseProgramacionV0)
	if closed.Status != OrchestrationPhaseStatusClosedV0 {
		t.Fatalf("phase status=%q, want cerrada", closed.Status)
	}
	if closed.OpenedAt != opened {
		t.Fatalf("opened_at=%q, want preserved %q", closed.OpenedAt, opened)
	}
	if closed.ClosedAt != event.OccurredAt {
		t.Fatalf("closed_at=%q, want %q", closed.ClosedAt, event.OccurredAt)
	}
	if got.CurrentPhase != OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%q, want unchanged", got.CurrentPhase)
	}
	if got.LastEventID != event.EventID || got.LastSequence != event.Sequence {
		t.Fatalf("last event not updated: %+v", got)
	}
	assertReducerRunValidV0(t, got)
}

func TestPhaseClosedPayloadNormalizesEvidenceRefsWithoutDuplicates(t *testing.T) {
	command := mustClosePhaseCommandV0(t, "cmd-close-dedupe", "idem-close-dedupe", OrchestrationPhaseProgramacionV0, "closure:dedupe")

	var payload ClosePhaseCommandPayloadV0
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		t.Fatalf("decode ClosePhase payload: %v", err)
	}
	if !reflect.DeepEqual(payload.EvidenceRefs, []string{"docs/pruebas.md#phase-close", "docs/decisiones.md#phase-close"}) {
		t.Fatalf("evidence_refs=%v, want compact unique refs", payload.EvidenceRefs)
	}
}

func TestReplayDurableEventsV0AcceptsDuplicatePhaseClosed(t *testing.T) {
	start := mustReplayRunStartedEventWithKeyV0(t, "evt-durable-close-start", 1, "idem-durable-close-start")
	open := mustReplayPhaseEventWithKeyV0(t, "evt-durable-close-open", 2, "idem-durable-close-open", OrchestrationPhaseProgramacionV0)
	close := mustPhaseClosedEventWithKeyV0(t, "evt-durable-close-phase", 3, "idem-durable-close-phase", OrchestrationPhaseProgramacionV0, "closure:programacion")

	got, err := ReplayDurableEventsV0([]OrchestrationEventV0{start, open, close, close})
	if err != nil {
		t.Fatalf("replay durable PhaseClosed duplicate: %v", err)
	}

	phase := reducerPhaseByIDV0(t, got, OrchestrationPhaseProgramacionV0)
	if phase.Status != OrchestrationPhaseStatusClosedV0 || phase.ClosedAt != close.OccurredAt {
		t.Fatalf("phase no cerrada tras replay: %+v", phase)
	}
	if got.LastSequence != 3 {
		t.Fatalf("last_sequence=%d, want 3", got.LastSequence)
	}
}

func TestReplayDurableEventsV0RejectsPhaseClosedWhenCurrentPhasePending(t *testing.T) {
	start := mustReplayRunStartedEventWithKeyV0(t, "evt-durable-close-pending-start", 1, "idem-durable-close-pending-start")
	close := mustPhaseClosedEventWithKeyV0(t, "evt-durable-close-pending", 2, "idem-durable-close-pending", OrchestrationPhaseDescubrimientoV0, "closure:pending")

	_, err := ReplayDurableEventsV0([]OrchestrationEventV0{start, close})
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrSecuenciaInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrSecuenciaInvalidaV0)
	}
}

func TestHandleClosePhaseCommandV0IdempotentWhenAlreadyReflected(t *testing.T) {
	run := mustOpenedPhaseRunV0(t, OrchestrationPhaseProgramacionV0)
	command := mustClosePhaseCommandV0(t, "cmd-close-idem", "idem-close-idem", OrchestrationPhaseProgramacionV0, "closure:idem")
	closed := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(closed, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestHandleClosePhaseCommandV0RepeatedAfterLaterPhaseDoesNotDuplicate(t *testing.T) {
	run := mustOpenedPhaseRunV0(t, OrchestrationPhaseProgramacionV0)
	command := mustClosePhaseCommandV0(t, "cmd-close-after-later", "idem-close-after-later", OrchestrationPhaseProgramacionV0, "closure:after-later")
	closed := mustApplySingleCommandEventV0(t, run, command)
	openRevision := mustOpenPhaseCommandV0(t, "cmd-open-after-close", "idem-open-after-close", OrchestrationPhaseRevisionV0)
	later := mustApplySingleCommandEventV0(t, closed, openRevision)

	result, err := HandleCommandV0(later, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestHandleClosePhaseCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustOpenedPhaseRunV0(t, OrchestrationPhaseProgramacionV0)
	command := mustClosePhaseCommandV0(t, "cmd-close-effect", "idem-close-effect", OrchestrationPhaseProgramacionV0, "closure:effect")
	closed := mustApplySingleCommandEventV0(t, run, command)
	conflict, err := NewClosePhaseCommandV0(validCommandMetaV0("cmd-close-effect", "idem-close-effect"), ClosePhaseCommandPayloadV0{
		PhaseID:      string(OrchestrationPhaseProgramacionV0),
		ClosureRef:   "closure:effect",
		Summary:      "Resumen distinto para cierre ya reflejado.",
		EvidenceRefs: []string{"docs/pruebas.md#distinto"},
	})
	conflict = mustCommandV0(t, conflict, err)

	_, err = HandleCommandV0(closed, conflict)
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyPhaseClosedEventV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustOpenedPhaseRunV0(t, OrchestrationPhaseProgramacionV0)
	event := mustPhaseClosedEventWithKeyV0(t, "evt-phase-close-effect", run.LastSequence+1, "idem-phase-close-effect", OrchestrationPhaseProgramacionV0, "closure:effect-event")
	closed := mustApplyReducerEventV0(t, run, event)
	conflict := mustPhaseClosedEventWithKeyV0(t, "evt-phase-close-effect-conflict", closed.LastSequence+1, "idem-phase-close-effect-conflict", OrchestrationPhaseProgramacionV0, "closure:effect-event")

	_, err := ApplyEventV0(closed, conflict)
	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestHandleClosePhaseCommandV0DoesNotTreatDifferentClosureRefAsReflected(t *testing.T) {
	run := mustOpenedPhaseRunV0(t, OrchestrationPhaseProgramacionV0)
	command := mustClosePhaseCommandV0(t, "cmd-close-reflected", "idem-close-reflected", OrchestrationPhaseProgramacionV0, "closure:one")
	closed := mustApplySingleCommandEventV0(t, run, command)
	changed := mustClosePhaseCommandV0(t, "cmd-close-reflected", "idem-close-reflected", OrchestrationPhaseProgramacionV0, "closure:two")

	_, err := HandleCommandV0(closed, changed)
	assertCommandPublicErrorCodeV0(t, err, ErrTransicionInvalidaV0)
}

func TestHandleClosePhaseCommandV0DoesNotTreatDifferentPayloadAsReflected(t *testing.T) {
	run := mustOpenedPhaseRunV0(t, OrchestrationPhaseProgramacionV0)
	command := mustClosePhaseCommandV0(t, "cmd-close-payload", "idem-close-payload", OrchestrationPhaseProgramacionV0, "closure:payload")
	closed := mustApplySingleCommandEventV0(t, run, command)
	changed, err := NewClosePhaseCommandV0(validCommandMetaV0("cmd-close-payload", "idem-close-payload"), ClosePhaseCommandPayloadV0{
		PhaseID:      string(OrchestrationPhaseProgramacionV0),
		ClosureRef:   "closure:payload",
		Summary:      "Fase cerrada con otra evidencia durable.",
		EvidenceRefs: []string{"docs/pruebas.md#phase-close"},
	})
	changed = mustCommandV0(t, changed, err)

	_, err = HandleCommandV0(closed, changed)
	assertCommandPublicErrorCodeV0(t, err, ErrTransicionInvalidaV0)
}

func TestHandleClosePhaseCommandV0RejectsNonCurrentPhase(t *testing.T) {
	run := mustOpenedPhaseRunV0(t, OrchestrationPhaseProgramacionV0)
	command := mustClosePhaseCommandV0(t, "cmd-close-non-current", "idem-close-non-current", OrchestrationPhaseRevisionV0, "closure:revision")

	_, err := HandleCommandV0(run, command)
	assertCommandPublicErrorCodeV0(t, err, ErrTransicionInvalidaV0)
}

func TestClosePhaseRejectsForbiddenDetails(t *testing.T) {
	_, err := NewClosePhaseCommandV0(validCommandMetaV0("cmd-close-forbidden", "idem-close-forbidden"), ClosePhaseCommandPayloadV0{
		PhaseID:    string(OrchestrationPhaseProgramacionV0),
		ClosureRef: "closure:forbidden",
		Summary:    "ver api_key=valor",
	})

	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestOpenPhaseStillWorksBeforeClosePhase(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustOpenPhaseCommandV0(t, "cmd-open-before-close", "idem-open-before-close", OrchestrationPhaseProgramacionV0)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle OpenPhase after ClosePhase addition: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventPhaseOpenedV0)
}

func mustOpenedPhaseRunV0(t *testing.T, phaseID OrchestrationPhaseIDV0) OrchestrationRunV0 {
	t.Helper()
	started := mustHandlerStartedRunV0(t)
	return mustApplySingleCommandEventV0(t, started, mustOpenPhaseCommandV0(t, "cmd-open-"+string(phaseID), "idem-open-"+string(phaseID), phaseID))
}

func mustClosePhaseCommandV0(t *testing.T, commandID string, idempotencyKey string, phaseID OrchestrationPhaseIDV0, closureRef string) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewClosePhaseCommandV0(validCommandMetaV0(commandID, idempotencyKey), ClosePhaseCommandPayloadV0{
		PhaseID:      string(phaseID),
		ClosureRef:   closureRef,
		Summary:      "Fase cerrada con evidencia durable.",
		EvidenceRefs: []string{" docs/pruebas.md#phase-close ", "docs/pruebas.md#phase-close", "docs/decisiones.md#phase-close"},
	})
	return mustCommandV0(t, command, err)
}

func mustPhaseClosedEventV0(t *testing.T, eventID string, sequence int64, phaseID OrchestrationPhaseIDV0, closureRef string) OrchestrationEventV0 {
	t.Helper()
	event, err := NewPhaseClosedEventV0(reducerEventMetaV0(eventID, sequence), PhaseClosedPayloadV0{
		PhaseID:      string(phaseID),
		ClosureRef:   closureRef,
		Summary:      "Fase cerrada con evidencia durable.",
		EvidenceRefs: []string{"docs/pruebas.md#phase-close"},
	})
	return mustReducerEventV0(t, event, err)
}

func mustPhaseClosedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, phaseID OrchestrationPhaseIDV0, closureRef string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewPhaseClosedEventV0(meta, PhaseClosedPayloadV0{
		PhaseID:    string(phaseID),
		ClosureRef: closureRef,
		Summary:    "Fase cerrada con evidencia durable.",
	})
	return mustReducerEventV0(t, event, err)
}

func assertCommandPublicErrorCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if !strings.EqualFold(publicErr.Code, code) {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
