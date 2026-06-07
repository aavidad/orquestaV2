package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestReplayDurableEventsV0OrderedEventsPass(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-001", 1, "idem-durable-start-001"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-001", 2, "idem-durable-phase-001", OrchestrationPhaseProgramacionV0),
		mustReplayDirectorQuestionEventWithKeyV0(t, "evt-durable-question-001", 3, "idem-durable-question-001", true),
		mustReplayRunBlockedEventWithKeyV0(t, "evt-durable-block-001", 4, "idem-durable-block-001"),
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable: %v", err)
	}

	if got.Status != OrchestrationRunStatusBlockedV0 {
		t.Fatalf("status=%q, want %q", got.Status, OrchestrationRunStatusBlockedV0)
	}
	if got.CurrentPhase != OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%q, want %q", got.CurrentPhase, OrchestrationPhaseProgramacionV0)
	}
	if got.LastSequence != 4 {
		t.Fatalf("last_sequence=%d, want 4", got.LastSequence)
	}
	if !reflect.DeepEqual(got.DirectorQuestions, []string{"question-001"}) {
		t.Fatalf("director_questions=%v, want [question-001]", got.DirectorQuestions)
	}
	if !reflect.DeepEqual(got.Blockers, []string{"blocker-001"}) {
		t.Fatalf("blockers=%v, want [blocker-001]", got.Blockers)
	}
}

func TestReplayDurableEventsV0BrokenSequenceFailsWithPublicError(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-002", 1, "idem-durable-start-002"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-002", 3, "idem-durable-phase-002", OrchestrationPhaseProgramacionV0),
	}

	_, err := ReplayDurableEventsV0(events)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrSecuenciaInvalidaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrSecuenciaInvalidaV0)
	}
}

func TestReplayDurableEventsV0FirstEventMustStartRun(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-first", 1, "idem-durable-phase-first", OrchestrationPhaseProgramacionV0),
	}

	_, err := ReplayDurableEventsV0(events)
	assertReplayPublicErrorCodeV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestReplayDurableEventsV0MixedRunIDFails(t *testing.T) {
	phase := mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-runid", 2, "idem-durable-phase-runid", OrchestrationPhaseProgramacionV0)
	phase.RunID = "run-otro"
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-runid", 1, "idem-durable-start-runid"),
		phase,
	}

	_, err := ReplayDurableEventsV0(events)
	assertReplayPublicErrorCodeV0(t, err, ErrSecuenciaInvalidaV0)
}

func TestReplayDurableEventsV0ExactDuplicateDoesNotDuplicateProgress(t *testing.T) {
	phase := mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-003", 2, "idem-durable-phase-003", OrchestrationPhaseProgramacionV0)
	block := mustReplayRunBlockedEventWithKeyV0(t, "evt-durable-block-003", 3, "idem-durable-block-003")
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-003", 1, "idem-durable-start-003"),
		phase,
		phase,
		block,
		block,
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable duplicate: %v", err)
	}

	if active := activePhaseIDsV0(got); len(active) != 1 || active[0] != OrchestrationPhaseProgramacionV0 {
		t.Fatalf("active phases=%v, want solo %s", active, OrchestrationPhaseProgramacionV0)
	}
	if !reflect.DeepEqual(got.Blockers, []string{"blocker-001"}) {
		t.Fatalf("blockers duplicados: %v", got.Blockers)
	}
	if got.LastSequence != 3 {
		t.Fatalf("last_sequence tras duplicados=%d, want 3", got.LastSequence)
	}
}

func TestReplayDurableEventsV0RunBlockerResolvedActivaRun(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-resolve", 1, "idem-durable-start-resolve"),
		mustReplayRunBlockedEventWithKeyV0(t, "evt-durable-block-resolve", 2, "idem-durable-block-resolve"),
		mustReplayRunBlockerResolvedEventWithKeyV0(t, "evt-durable-resolve", 3, "idem-durable-resolve", "blocker-001"),
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable resolve: %v", err)
	}
	if got.Status != OrchestrationRunStatusActiveV0 || len(got.Blockers) != 0 {
		t.Fatalf("got=%+v", got)
	}
}

func TestReplayDurableEventsV0ConflictingDuplicateFails(t *testing.T) {
	first := mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-004-a", 2, "idem-phase-conflict", OrchestrationPhaseProgramacionV0)
	conflict := mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-004-b", 2, "idem-phase-conflict", OrchestrationPhaseRevisionV0)
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-004", 1, "idem-durable-start-004"),
		first,
		conflict,
	}

	_, err := ReplayDurableEventsV0(events)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrEventoConflictivoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrEventoConflictivoV0)
	}
}

func TestReplayDurableEventsV0DuplicateWithDifferentMetadataFails(t *testing.T) {
	first := mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-metadata", 2, "idem-phase-metadata", OrchestrationPhaseProgramacionV0)
	conflict := first
	conflict.CausationID = "cmd-distinto"
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-metadata", 1, "idem-durable-start-metadata"),
		first,
		conflict,
	}

	_, err := ReplayDurableEventsV0(events)
	assertReplayPublicErrorCodeV0(t, err, ErrEventoConflictivoV0)
}

func assertReplayPublicErrorCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}

func mustReplayRunStartedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewRunStartedEventV0(meta, RunStartedPayloadV0{
		ProjectRef:  "project:ventas",
		AppSpecRef:  "appspec:req-001",
		RequestedBy: "director",
	})
	return mustReducerEventV0(t, event, err)
}

func mustReplayPhaseEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, phaseID OrchestrationPhaseIDV0) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewPhaseOpenedEventV0(meta, PhaseOpenedPayloadV0{
		PhaseID:  string(phaseID),
		OpenedBy: "director",
		Reason:   "apertura de fase canonica",
	})
	return mustReducerEventV0(t, event, err)
}

func mustReplayRunBlockedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewRunBlockedEventV0(meta, RunBlockedPayloadV0{
		BlockerID:    "blocker-001",
		ReasonCode:   "consulta_director_requerida",
		Summary:      "Falta una decision de contrato antes de continuar.",
		SourceGroup:  "workflow",
		EvidenceRefs: []string{"docs/contratos.md#OrchestrationEventV0"},
	})
	return mustReducerEventV0(t, event, err)
}

func mustReplayRunBlockerResolvedEventWithKeyV0(
	t *testing.T,
	eventID string,
	sequence int64,
	idempotencyKey string,
	blockerID string,
) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewRunBlockerResolvedEventV0(meta, RunBlockerResolvedPayloadV0{
		BlockerID:    blockerID,
		ReasonCode:   "bloqueo_resuelto",
		Summary:      "El criterio pendiente queda resuelto con evidencia durable.",
		EvidenceRefs: []string{"docs/contratos.md#RunBlockerResolved"},
	})
	return mustReducerEventV0(t, event, err)
}

func mustReplayDirectorQuestionEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, blocking bool) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	payload := DirectorQuestionRaisedPayloadV0{
		QuestionID:   "question-001",
		SourceGroup:  "workflow",
		TargetGroup:  "director",
		Summary:      "Falta una decision de contrato antes de continuar.",
		EvidenceRefs: []string{"docs/contratos.md#OrchestrationEventV0"},
		Blocking:     blocking,
	}
	if blocking {
		payload.BlockerID = "director-question-question-001"
	}
	event, err := NewDirectorQuestionRaisedEventV0(meta, payload)
	return mustReducerEventV0(t, event, err)
}
