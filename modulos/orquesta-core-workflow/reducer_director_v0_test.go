package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

func TestApplyEventV0RunBlockedMarksRunAndAddsCompactBlocker(t *testing.T) {
	run := mustReducerStartedRunV0(t)
	event := mustReducerRunBlockedEventV0(t, "evt-run-blocked-reducer-001", 2)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply RunBlocked: %v", err)
	}

	if got.Status != OrchestrationRunStatusBlockedV0 {
		t.Fatalf("status=%q, want %q", got.Status, OrchestrationRunStatusBlockedV0)
	}
	if got.LastEventID != event.EventID {
		t.Fatalf("last_event_id=%q, want %q", got.LastEventID, event.EventID)
	}
	if !reflect.DeepEqual(got.Blockers, []string{"blocker-001"}) {
		t.Fatalf("blockers=%v, want [blocker-001]", got.Blockers)
	}

	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.Blockers, []string{"blocker-001"}) {
		t.Fatalf("blockers duplicados tras repetir evento: %v", again.Blockers)
	}
}

func TestApplyEventV0RunBlockerResolvedActivaSoloSiNoQuedanBlockers(t *testing.T) {
	run := mustReducerStartedRunV0(t)
	run = mustApplyReducerEventV0(t, run, mustReducerRunBlockedWithRefEventV0(t, "evt-run-blocked-resolve-001", 2, "blocker-a"))
	run = mustApplyReducerEventV0(t, run, mustReducerRunBlockedWithRefEventV0(t, "evt-run-blocked-resolve-002", 3, "blocker-b"))

	first, err := ApplyEventV0(run, mustReducerRunBlockerResolvedEventV0(t, "evt-run-blocker-resolved-001", 4, "blocker-a"))
	if err != nil {
		t.Fatalf("apply RunBlockerResolved first: %v", err)
	}
	if first.Status != OrchestrationRunStatusBlockedV0 ||
		!reflect.DeepEqual(first.Blockers, []string{"blocker-b"}) {
		t.Fatalf("first=%+v", first)
	}

	second, err := ApplyEventV0(first, mustReducerRunBlockerResolvedEventV0(t, "evt-run-blocker-resolved-002", 5, "blocker-b"))
	if err != nil {
		t.Fatalf("apply RunBlockerResolved second: %v", err)
	}
	if second.Status != OrchestrationRunStatusActiveV0 || len(second.Blockers) != 0 {
		t.Fatalf("second=%+v", second)
	}
}

func TestApplyEventV0RunBlockerResolvedEsIdempotente(t *testing.T) {
	run := mustReducerStartedRunV0(t)
	run = mustApplyReducerEventV0(t, run, mustReducerRunBlockedWithRefEventV0(t, "evt-run-blocked-idem-001", 2, "blocker-idem"))
	event := mustReducerRunBlockerResolvedEventV0(t, "evt-run-blocker-resolved-idem-001", 3, "blocker-idem")
	first := mustApplyReducerEventV0(t, run, event)

	second, err := ApplyEventV0(first, event)
	if err != nil {
		t.Fatalf("apply duplicate RunBlockerResolved: %v", err)
	}
	if second.Status != OrchestrationRunStatusActiveV0 || len(second.Blockers) != 0 {
		t.Fatalf("second=%+v", second)
	}
}

func TestApplyEventV0DirectorQuestionRaisedAddsCompactRef(t *testing.T) {
	run := mustReducerStartedRunV0(t)
	event := mustReducerDirectorQuestionRaisedEventV0(t, "evt-director-question-reducer-001", 2, false)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply DirectorQuestionRaised: %v", err)
	}

	if !reflect.DeepEqual(got.DirectorQuestions, []string{"question-001"}) {
		t.Fatalf("director_questions=%v, want [question-001]", got.DirectorQuestions)
	}
	if got.LastEventID != event.EventID || got.LastSequence != event.Sequence {
		t.Fatalf("last event not updated: %+v", got)
	}

	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.DirectorQuestions, []string{"question-001"}) {
		t.Fatalf("director questions duplicated: %v", again.DirectorQuestions)
	}
}

func TestApplyEventV0DirectorQuestionAnsweredAddsCompactRef(t *testing.T) {
	run := mustReducerRunWithDirectorQuestionV0(t, false)
	event := mustReducerDirectorQuestionAnsweredEventV0(t, "evt-director-answer-reducer-001", 3, false)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply DirectorQuestionAnswered: %v", err)
	}
	if !reflect.DeepEqual(got.DirectorAnswers, []string{"answer-001"}) {
		t.Fatalf("director_answers=%v, want [answer-001]", got.DirectorAnswers)
	}

	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.DirectorAnswers, []string{"answer-001"}) {
		t.Fatalf("director answers duplicated: %v", again.DirectorAnswers)
	}
}

func TestApplyEventV0DirectorQuestionAnsweredUnblocksRun(t *testing.T) {
	run := mustReducerRunWithDirectorQuestionV0(t, true)
	event := mustReducerDirectorQuestionAnsweredEventV0(t, "evt-director-answer-reducer-002", 4, true)

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply DirectorQuestionAnswered unblock: %v", err)
	}
	if got.Status != OrchestrationRunStatusActiveV0 {
		t.Fatalf("status=%q, want %q", got.Status, OrchestrationRunStatusActiveV0)
	}
	if blockerAlreadyReflectedV0(got, "director-question-question-001") {
		t.Fatalf("blocker sigue proyectado: %v", got.Blockers)
	}
}

func TestApplyEventV0DirectorQuestionAnsweredRequiresKnownQuestion(t *testing.T) {
	run := mustReducerStartedRunV0(t)
	event := mustReducerDirectorQuestionAnsweredEventV0(t, "evt-director-answer-reducer-003", 2, false)

	_, err := ApplyEventV0(run, event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrSecuenciaInvalidaV0 || publicErr.Field != "payload.question_id" {
		t.Fatalf("error=%+v, want invalid payload.question_id", publicErr)
	}
}

func mustReducerRunWithDirectorQuestionV0(t *testing.T, blocking bool) OrchestrationRunV0 {
	t.Helper()
	run := mustReducerStartedRunV0(t)
	run = mustApplyReducerEventV0(t, run, mustReducerDirectorQuestionRaisedEventV0(t, "evt-director-question-for-answer", 2, blocking))
	if blocking {
		run = mustApplyReducerEventV0(t, run, mustReducerDirectorQuestionBlockerEventV0(t))
	}
	return run
}

func mustReducerDirectorQuestionBlockerEventV0(t *testing.T) OrchestrationEventV0 {
	t.Helper()
	event, err := NewRunBlockedEventV0(reducerEventMetaV0("evt-director-question-blocker", 3), RunBlockedPayloadV0{
		BlockerID:    "director-question-question-001",
		ReasonCode:   "consulta_director_requerida",
		Summary:      "Falta una decision compacta del director.",
		SourceGroup:  "workflow",
		EvidenceRefs: []string{"docs/contratos.md#DirectorQuestionRaised"},
	})
	return mustReducerEventV0(t, event, err)
}
