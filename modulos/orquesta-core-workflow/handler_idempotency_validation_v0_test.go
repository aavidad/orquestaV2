package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestHandleCommandV0SameIdempotencyKeyDoesNotDuplicateReflectedProgress(t *testing.T) {
	start := mustStartRunCommandV0(t, "cmd-start-002", "idem-start-002")
	started := mustApplySingleCommandEventV0(t, OrchestrationRunV0{}, start)

	result, err := HandleCommandV0(started, start)
	assertIdempotentNoEventsV0(t, result, err)

	open := mustOpenPhaseCommandV0(t, "cmd-open-002", "idem-open-002", OrchestrationPhaseProgramacionV0)
	opened := mustApplySingleCommandEventV0(t, started, open)
	result, err = HandleCommandV0(opened, open)
	assertIdempotentNoEventsV0(t, result, err)

	block := mustBlockRunCommandV0(t, "cmd-block-002", "idem-block-002", "blocker-002")
	blocked := mustApplySingleCommandEventV0(t, opened, block)
	result, err = HandleCommandV0(blocked, block)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestHandleStartRunCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	start := mustStartRunCommandV0(t, "cmd-start-effect", "idem-start-effect")
	started := mustApplySingleCommandEventV0(t, OrchestrationRunV0{}, start)
	conflict := mustStartRunCommandV0(t, "cmd-start-effect-conflict", "idem-start-effect-conflict")

	_, err := HandleCommandV0(started, conflict)
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestHandleOpenPhaseCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	open := mustOpenPhaseCommandV0(t, "cmd-open-effect", "idem-open-effect", OrchestrationPhaseProgramacionV0)
	opened := mustApplySingleCommandEventV0(t, run, open)
	conflict, err := NewOpenPhaseCommandV0(validCommandMetaV0("cmd-open-effect", "idem-open-effect"), OpenPhaseCommandPayloadV0{
		PhaseID: string(OrchestrationPhaseProgramacionV0),
		Reason:  "razon incompatible para la misma apertura",
	})
	conflict = mustCommandV0(t, conflict, err)

	_, err = HandleCommandV0(opened, conflict)
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestHandleOpenPhaseCommandV0DistinctCommandForActivePhaseCreatesEvent(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	first := mustOpenPhaseCommandV0(t, "cmd-open-active-first", "idem-open-active-first", OrchestrationPhaseProgramacionV0)
	opened := mustApplySingleCommandEventV0(t, run, first)
	second := mustOpenPhaseCommandV0(t, "cmd-open-active-second", "idem-open-active-second", OrchestrationPhaseProgramacionV0)

	result, err := HandleCommandV0(opened, second)
	if err != nil {
		t.Fatalf("handle second OpenPhase: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventPhaseOpenedV0)
}

func TestHandleOpenPhaseCommandV0RepeatedAfterLaterPhaseDoesNotDuplicate(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	openProgramacion := mustOpenPhaseCommandV0(t, "cmd-open-repeat-later", "idem-open-repeat-later", OrchestrationPhaseProgramacionV0)
	programacion := mustApplySingleCommandEventV0(t, run, openProgramacion)
	openRevision := mustOpenPhaseCommandV0(t, "cmd-open-repeat-later-revision", "idem-open-repeat-later-revision", OrchestrationPhaseRevisionV0)
	revision := mustApplySingleCommandEventV0(t, programacion, openRevision)

	result, err := HandleCommandV0(revision, openProgramacion)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestApplyPhaseOpenedEventV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustReducerStartedRunV0(t)
	event := mustReducerPhaseOpenedEventV0(t, "evt-phase-open-effect", 2, OrchestrationPhaseProgramacionV0)
	opened := mustApplyReducerEventV0(t, run, event)
	conflict, err := NewPhaseOpenedEventV0(reducerEventMetaV0("evt-phase-open-effect", 3), PhaseOpenedPayloadV0{
		PhaseID:  string(OrchestrationPhaseRevisionV0),
		OpenedBy: "director",
		Reason:   "payload incompatible para misma apertura",
	})
	conflict = mustReducerEventV0(t, conflict, err)

	_, err = ApplyEventV0(opened, conflict)
	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestHandleBlockRunCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	block := mustBlockRunCommandV0(t, "cmd-block-effect", "idem-block-effect", "blocker-effect")
	blocked := mustApplySingleCommandEventV0(t, run, block)
	conflict, err := NewBlockRunCommandV0(validCommandMetaV0("cmd-block-effect", "idem-block-effect"), BlockRunCommandPayloadV0{
		BlockerID:    "blocker-effect",
		ReasonCode:   "consulta_director_requerida",
		Summary:      "Resumen incompatible para el mismo blocker.",
		SourceGroup:  "workflow",
		EvidenceRefs: []string{"docs/contratos.md#OrchestrationCommandV0"},
	})
	conflict = mustCommandV0(t, conflict, err)

	_, err = HandleCommandV0(blocked, conflict)
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestApplyRunBlockedEventV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustReducerStartedRunV0(t)
	blocked := mustApplyReducerEventV0(t, run, mustReducerRunBlockedEventV0(t, "evt-run-blocked-effect", 2))
	conflict := mustReducerRunBlockedEventV0(t, "evt-run-blocked-effect-conflict", 3)

	_, err := ApplyEventV0(blocked, conflict)
	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestHandleCommandV0UnknownCommandReturnsPublicError(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-unknown-001", "idem-unknown-001")
	command.CommandType = "UnknownCommand"

	_, err := HandleCommandV0(OrchestrationRunV0{}, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrComandoNoSoportadoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrComandoNoSoportadoV0)
	}
}

func TestHandleCommandV0RequiresIdempotencyKey(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-003", "idem-start-003")
	command.IdempotencyKey = " "

	_, err := HandleCommandV0(OrchestrationRunV0{}, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrIdempotencyKeyRequeridaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrIdempotencyKeyRequeridaV0)
	}
}

func TestHandleCommandV0RejectsUnsupportedPhase(t *testing.T) {
	command := mustOpenPhaseCommandV0(t, "cmd-open-003", "idem-open-003", OrchestrationPhaseProgramacionV0)
	command.Payload = []byte(`{"phase_id":"despliegue"}`)

	_, err := HandleCommandV0(mustHandlerStartedRunV0(t), command)
	var publicErr OrchestrationValidationIssueV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public phase error, got %T %v", err, err)
	}
	if publicErr.Code != OrchestrationFaseNoSoportadaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, OrchestrationFaseNoSoportadaV0)
	}
}
