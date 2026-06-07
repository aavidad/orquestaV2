package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestHandleCommandV0StartRunReturnsRunStartedAndNoOutbox(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-001", "idem-start-001")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	event := result.Events[0]
	if event.IdempotencyKey != command.IdempotencyKey || event.CausationID != command.CommandID {
		t.Fatalf("event meta not derived from command: %+v", event)
	}
	if event.Sequence != 1 {
		t.Fatalf("event sequence=%d, want 1", event.Sequence)
	}

	run := mustApplyReducerEventV0(t, OrchestrationRunV0{}, event)
	if run.RunID != command.RunID || run.ProjectRef != "project:ventas" {
		t.Fatalf("run no refleja StartRun: %+v", run)
	}
}

func TestHandleCommandV0OpenPhaseReturnsPhaseOpenedWhenSupported(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustOpenPhaseCommandV0(t, "cmd-open-001", "idem-open-001", OrchestrationPhaseProgramacionV0)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle OpenPhase: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventPhaseOpenedV0)
	if result.Events[0].Sequence != run.LastSequence+1 {
		t.Fatalf("event sequence=%d, want %d", result.Events[0].Sequence, run.LastSequence+1)
	}
	next := mustApplyReducerEventV0(t, run, result.Events[0])
	if next.CurrentPhase != OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%q, want %q", next.CurrentPhase, OrchestrationPhaseProgramacionV0)
	}
}

func TestHandleCommandV0SequenceUsesLastSequence(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	run.LastSequence = 41
	command := mustOpenPhaseCommandV0(t, "cmd-open-sequence", "idem-open-sequence", OrchestrationPhaseProgramacionV0)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle OpenPhase: %v", err)
	}
	if result.Events[0].Sequence != 42 {
		t.Fatalf("sequence=%d, want 42", result.Events[0].Sequence)
	}
}

func TestHandleCommandV0BlockRunReturnsRunBlocked(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustBlockRunCommandV0(t, "cmd-block-001", "idem-block-001", "blocker-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle BlockRun: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventRunBlockedV0)
	next := mustApplyReducerEventV0(t, run, result.Events[0])
	if next.Status != OrchestrationRunStatusBlockedV0 || len(next.Blockers) != 1 {
		t.Fatalf("run no refleja BlockRun: %+v", next)
	}
}

func TestHandleCommandV0ResolveRunBlockerReturnsRunBlockerResolved(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustBlockRunCommandV0(t, "cmd-block-resolve-001", "idem-block-resolve-001", "blocker-resolve-001"))
	command := mustResolveRunBlockerCommandV0(t, "cmd-resolve-001", "idem-resolve-001", "blocker-resolve-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle ResolveRunBlocker: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventRunBlockerResolvedV0)
	next := mustApplyReducerEventV0(t, run, result.Events[0])
	if next.Status != OrchestrationRunStatusActiveV0 || len(next.Blockers) != 0 {
		t.Fatalf("run no refleja ResolveRunBlocker: %+v", next)
	}
	repeated, err := HandleCommandV0(next, command)
	if err != nil {
		t.Fatalf("handle ResolveRunBlocker repeat: %v", err)
	}
	if !repeated.Idempotent || len(repeated.Events) != 0 {
		t.Fatalf("repeat result=%+v", repeated)
	}
}

func TestHandleCommandV0ResolveRunBlockerRepairsPartialProjection(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	blocked := mustApplySingleCommandEventV0(t, run, mustBlockRunCommandV0(t, "cmd-block-partial-001", "idem-block-partial-001", "blocker-partial-001"))
	command := mustResolveRunBlockerCommandV0(t, "cmd-resolve-partial-001", "idem-resolve-partial-001", "blocker-partial-001")
	resolved := mustApplySingleCommandEventV0(t, blocked, command)
	partial := blocked
	partial.CommandEffects = resolved.CommandEffects

	result, err := HandleCommandV0(partial, command)
	if err != nil {
		t.Fatalf("handle ResolveRunBlocker partial: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunBlockerResolvedV0)
	if result.Events[0].Sequence != partial.LastSequence {
		t.Fatalf("repair sequence=%d, want current last_sequence=%d", result.Events[0].Sequence, partial.LastSequence)
	}
	repaired := mustApplyReducerEventV0(t, partial, result.Events[0])
	if repaired.Status != OrchestrationRunStatusActiveV0 || len(repaired.Blockers) != 0 {
		t.Fatalf("partial projection no queda reparada: %+v", repaired)
	}
	if repaired.LastEventID != partial.LastEventID || repaired.LastSequence != partial.LastSequence {
		t.Fatalf("repair movio historial: last_event=%q/%q last_sequence=%d/%d", repaired.LastEventID, partial.LastEventID, repaired.LastSequence, partial.LastSequence)
	}
	if len(repaired.CommandEffects) != len(partial.CommandEffects) {
		t.Fatalf("repair duplico command_effects: got %d want %d", len(repaired.CommandEffects), len(partial.CommandEffects))
	}

	repeated, err := HandleCommandV0(repaired, command)
	if err != nil {
		t.Fatalf("handle ResolveRunBlocker repaired repeat: %v", err)
	}
	if !repeated.Idempotent || len(repeated.Events) != 0 {
		t.Fatalf("repaired repeat result=%+v", repeated)
	}
}

func mustHandlerStartedRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	return mustApplySingleCommandEventV0(t, OrchestrationRunV0{}, mustStartRunCommandV0(t, "cmd-start-base", "idem-start-base"))
}

func mustApplySingleCommandEventV0(t *testing.T, current OrchestrationRunV0, command OrchestrationCommandV0) OrchestrationRunV0 {
	t.Helper()
	result, err := HandleCommandV0(current, command)
	if err != nil {
		t.Fatalf("handle %s: %v", command.CommandType, err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events=%d, want 1", len(result.Events))
	}
	return mustApplyReducerEventV0(t, current, result.Events[0])
}

func mustStartRunCommandV0(t *testing.T, commandID string, idempotencyKey string) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewStartRunCommandV0(validCommandMetaV0(commandID, idempotencyKey), StartRunCommandPayloadV0{
		ProjectRef: "project:ventas",
		AppSpecRef: "appspec:req-001",
	})
	return mustCommandV0(t, command, err)
}

func mustOpenPhaseCommandV0(t *testing.T, commandID string, idempotencyKey string, phaseID OrchestrationPhaseIDV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewOpenPhaseCommandV0(validCommandMetaV0(commandID, idempotencyKey), OpenPhaseCommandPayloadV0{
		PhaseID: string(phaseID),
		Reason:  "apertura de fase canonica",
	})
	return mustCommandV0(t, command, err)
}

func mustBlockRunCommandV0(t *testing.T, commandID string, idempotencyKey string, blockerID string) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewBlockRunCommandV0(validCommandMetaV0(commandID, idempotencyKey), BlockRunCommandPayloadV0{
		BlockerID:    blockerID,
		ReasonCode:   "consulta_director_requerida",
		Summary:      "Falta una decision de contrato antes de continuar.",
		SourceGroup:  "workflow",
		EvidenceRefs: []string{"docs/contratos.md#OrchestrationCommandV0"},
	})
	return mustCommandV0(t, command, err)
}

func mustResolveRunBlockerCommandV0(t *testing.T, commandID string, idempotencyKey string, blockerID string) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewResolveRunBlockerCommandV0(validCommandMetaV0(commandID, idempotencyKey), ResolveRunBlockerCommandPayloadV0{
		BlockerID:    blockerID,
		ReasonCode:   "bloqueo_resuelto",
		Summary:      "El criterio pendiente queda resuelto con evidencia durable.",
		EvidenceRefs: []string{"docs/contratos.md#ResolveRunBlocker"},
	})
	return mustCommandV0(t, command, err)
}

func validCommandMetaV0(commandID string, idempotencyKey string) OrchestrationCommandMetaV0 {
	return OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          "run-001",
		IdempotencyKey: idempotencyKey,
		CorrelationID:  "corr-command-001",
		RequestedBy:    "director",
		OccurredAt:     "2026-05-04T10:00:00Z",
	}
}

func mustCommandV0(t *testing.T, command OrchestrationCommandV0, err error) OrchestrationCommandV0 {
	t.Helper()
	if err != nil {
		t.Fatalf("command constructor failed: %v", err)
	}
	return command
}

func assertSingleEventTypeV0(t *testing.T, result OrchestrationCommandResultV0, eventType string) {
	t.Helper()
	if len(result.Events) != 1 {
		t.Fatalf("events=%d, want 1", len(result.Events))
	}
	if result.Events[0].EventType != eventType {
		t.Fatalf("event_type=%q, want %q", result.Events[0].EventType, eventType)
	}
}

func assertIdempotentNoEventsV0(t *testing.T, result OrchestrationCommandResultV0, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("handle idempotent command: %v", err)
	}
	if len(result.Events) != 0 || len(result.Outbox) != 0 {
		t.Fatalf("idempotent result emitted events/outbox: %+v", result)
	}
	if !result.Idempotent || result.NoopReason != OrchestrationCommandNoopAlreadyReflectedV0 {
		t.Fatalf("unexpected idempotent marker: %+v", result)
	}
}

func assertCommandErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want %s %s", publicErr, code, field)
	}
}

func assertEventErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want %s %s", publicErr, code, field)
	}
}
