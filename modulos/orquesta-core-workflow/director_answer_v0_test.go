package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestHandleAnswerDirectorQuestionCommandV0UnblocksBlockingQuestion(t *testing.T) {
	run := mustQuestionedRunV0(t, true)
	command := mustAnswerDirectorQuestionCommandV0(t, "cmd-answer-001", "idem-answer-001", true)

	result, err := HandleAnswerDirectorQuestionCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle AnswerDirectorQuestion: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventDirectorQuestionAnsweredV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}

	next := mustApplyReducerEventV0(t, run, result.Events[0])
	if next.Status != OrchestrationRunStatusActiveV0 {
		t.Fatalf("status=%q, want %q", next.Status, OrchestrationRunStatusActiveV0)
	}
	if blockerAlreadyReflectedV0(next, "director-question-question-ask-001") {
		t.Fatalf("blocker sigue proyectado: %+v", next.Blockers)
	}
	if !directorAnswerAlreadyReflectedV0(next, "answer-ask-001") {
		t.Fatalf("answer not reflected: %+v", next.DirectorAnswers)
	}
	if !directorQuestionAnsweredAlreadyReflectedV0(next, "question-ask-001") {
		t.Fatalf("answered question not reflected: %+v", next.DirectorAnsweredQuestions)
	}
}

func TestHandleAnswerDirectorQuestionCommandV0RequiresExistingQuestion(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAnswerDirectorQuestionCommandV0(t, "cmd-answer-002", "idem-answer-002", false)

	_, err := HandleAnswerDirectorQuestionCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 || publicErr.Field != "payload.question_id" {
		t.Fatalf("error=%+v, want invalid payload.question_id", publicErr)
	}
}

func TestAnswerDirectorQuestionCommandV0IsSupportedByGenericHandler(t *testing.T) {
	run := mustQuestionedRunV0(t, false)
	command := mustAnswerDirectorQuestionCommandV0(t, "cmd-answer-003", "idem-answer-003", false)

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("generic handler rejected AnswerDirectorQuestion: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventDirectorQuestionAnsweredV0)

	next := mustApplyReducerEventV0(t, run, result.Events[0])
	result, err = HandleCommandV0(next, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestAnswerDirectorQuestionCommandV0RepeatedAfterUnblockDoesNotDuplicate(t *testing.T) {
	run := mustQuestionedRunV0(t, true)
	command := mustAnswerDirectorQuestionCommandV0(t, "cmd-answer-unblock-repeat", "idem-answer-unblock-repeat", true)
	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle blocking answer: %v", err)
	}
	next := mustApplyReducerEventV0(t, run, result.Events[0])

	result, err = HandleCommandV0(next, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestAnswerDirectorQuestionCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustQuestionedRunV0(t, false)
	payload := validAnswerDirectorQuestionPayloadV0(false)
	command := mustAnswerDirectorQuestionCommandWithPayloadV0(t, "cmd-answer-effect", "idem-answer-effect", payload)
	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle answer: %v", err)
	}
	answered := mustApplyReducerEventV0(t, run, result.Events[0])

	payload.Summary = "Continuar con otra respuesta compacta."
	conflicting := mustAnswerDirectorQuestionCommandWithPayloadV0(t, "cmd-answer-effect", "idem-answer-effect", payload)
	_, err = HandleCommandV0(answered, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestAnswerDirectorQuestionCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustQuestionedRunV0(t, false)
	command := mustAnswerDirectorQuestionCommandV0(t, "cmd-answer-key", "idem-answer-key", false)
	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle answer: %v", err)
	}
	answered := mustApplyReducerEventV0(t, run, result.Events[0])

	conflicting := mustAnswerDirectorQuestionCommandV0(t, "cmd-answer-key-2", "idem-answer-key-2", false)
	_, err = HandleCommandV0(answered, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyDirectorQuestionAnsweredV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustQuestionedRunV0(t, false)
	payload := directorQuestionAnsweredPayloadFromCommandV0(validAnswerDirectorQuestionPayloadV0(false))
	first := mustDirectorQuestionAnsweredEventWithKeyV0(t, "evt-answer-effect", run.LastSequence+1, "idem-answer-effect", payload)
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustDirectorQuestionAnsweredEventWithKeyV0(t, "evt-answer-effect-conflict", applied.LastSequence+1, "idem-answer-effect-conflict", payload)

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestValidateAnswerDirectorQuestionCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validAnswerDirectorQuestionPayloadV0(false)
	payload.Summary = "usar provider real para decidir"

	_, err := NewAnswerDirectorQuestionCommandV0(validCommandMetaV0("cmd-answer-004", "idem-answer-004"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func mustQuestionedRunV0(t *testing.T, blocking bool) OrchestrationRunV0 {
	t.Helper()
	run := mustHandlerStartedRunV0(t)
	result, err := HandleAskDirectorCommandV0(run, mustAskDirectorCommandV0(t, "cmd-ask-answer-001", "idem-ask-answer-001", blocking))
	if err != nil {
		t.Fatalf("handle AskDirector setup: %v", err)
	}
	for _, event := range result.Events {
		run = mustApplyReducerEventV0(t, run, event)
	}
	return run
}

func mustAnswerDirectorQuestionCommandV0(t *testing.T, commandID string, idempotencyKey string, unblocks bool) OrchestrationCommandV0 {
	t.Helper()
	return mustAnswerDirectorQuestionCommandWithPayloadV0(t, commandID, idempotencyKey, validAnswerDirectorQuestionPayloadV0(unblocks))
}

func mustAnswerDirectorQuestionCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload AnswerDirectorQuestionCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewAnswerDirectorQuestionCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustDirectorQuestionAnsweredEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, payload DirectorQuestionAnsweredPayloadV0) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewDirectorQuestionAnsweredEventV0(meta, payload)
	return mustReducerEventV0(t, event, err)
}

func validAnswerDirectorQuestionPayloadV0(unblocks bool) AnswerDirectorQuestionCommandPayloadV0 {
	return AnswerDirectorQuestionCommandPayloadV0{
		AnswerID:     "answer-ask-001",
		QuestionID:   "question-ask-001",
		Decision:     DirectorAnswerDecisionContinueV0,
		Summary:      "Continuar con la alternativa compacta y revisar en la siguiente fase.",
		EvidenceRefs: []string{"docs/contratos.md#AnswerDirectorQuestion"},
		Unblocks:     unblocks,
	}
}
