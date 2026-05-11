package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestReplayEventsV0RebuildsMinimalRun(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReducerRunStartedEventV0(t, "evt-run-started-replay-001", 1),
		mustReducerPhaseOpenedEventV0(t, "evt-phase-opened-replay-001", 2, OrchestrationPhaseDescubrimientoV0),
		mustReducerDirectorQuestionRaisedEventV0(t, "evt-director-question-replay-001", 3, true),
		mustReducerRunBlockedEventV0(t, "evt-run-blocked-replay-001", 4),
	}

	got, err := ReplayEventsV0(events)
	if err != nil {
		t.Fatalf("replay events: %v", err)
	}

	if got.Status != OrchestrationRunStatusBlockedV0 {
		t.Fatalf("status=%q, want %q", got.Status, OrchestrationRunStatusBlockedV0)
	}
	if got.CurrentPhase != OrchestrationPhaseDescubrimientoV0 {
		t.Fatalf("current_phase=%q, want %q", got.CurrentPhase, OrchestrationPhaseDescubrimientoV0)
	}
	if got.LastEventID != "evt-run-blocked-replay-001" {
		t.Fatalf("last_event_id=%q", got.LastEventID)
	}
	if got.LastSequence != 4 {
		t.Fatalf("last_sequence=%d, want 4", got.LastSequence)
	}
	if !reflect.DeepEqual(got.DirectorQuestions, []string{"question-001"}) {
		t.Fatalf("director_questions=%v, want [question-001]", got.DirectorQuestions)
	}
	if len(activePhaseIDsV0(got)) != 1 {
		t.Fatalf("replay no conserva una unica fase activa: %v", activePhaseIDsV0(got))
	}
	if !reflect.DeepEqual(got.Blockers, []string{"blocker-001"}) {
		t.Fatalf("blockers=%v, want [blocker-001]", got.Blockers)
	}
	assertReducerRunValidV0(t, got)
}

func TestApplyEventV0UnknownEventReturnsPublicErrorAndDoesNotMutateState(t *testing.T) {
	current := mustReducerStartedRunV0(t)
	before := cloneRunForReducerV0(current)
	event := reducerUnknownEventV0()

	got, err := ApplyEventV0(current, event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrEventoNoSoportadoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrEventoNoSoportadoV0)
	}
	if !reflect.DeepEqual(got, current) {
		t.Fatalf("estado devuelto mutado: got=%+v current=%+v", got, current)
	}
	if !reflect.DeepEqual(current, before) {
		t.Fatalf("estado de entrada mutado: before=%+v after=%+v", before, current)
	}
}
