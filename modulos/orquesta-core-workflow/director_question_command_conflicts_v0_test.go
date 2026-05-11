package orquestacoreworkflow

import "testing"

func TestHandleAskDirectorCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAskDirectorCommandV0(t, "cmd-ask-conflict", "idem-ask-conflict", false)
	first, err := HandleAskDirectorCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle first AskDirector: %v", err)
	}
	questioned := mustApplyReducerEventV0(t, run, first.Events[0])
	payload := validAskDirectorPayloadV0(false)
	payload.Summary = "Falta decidir si la consulta queda como decision local documentada."
	conflicting := mustAskDirectorCommandWithPayloadV0(t, "cmd-ask-conflict", "idem-ask-conflict", payload)

	result, err := HandleAskDirectorCommandV0(questioned, conflicting)

	if len(result.Events) != 0 || len(result.Outbox) != 0 {
		t.Fatalf("conflict emitted result: %+v", result)
	}
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestHandleAskDirectorCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAskDirectorCommandV0(t, "cmd-ask-key-conflict", "idem-ask-key-conflict", false)
	first, err := HandleAskDirectorCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle first AskDirector: %v", err)
	}
	questioned := mustApplyReducerEventV0(t, run, first.Events[0])
	conflicting := mustAskDirectorCommandV0(t, "cmd-ask-key-conflict-2", "idem-ask-key-conflict-2", false)

	_, err = HandleAskDirectorCommandV0(questioned, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyDirectorQuestionRaisedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	first := mustDirectorQuestionRaisedEventWithKeyV0(t, "evt-question-effect", run.LastSequence+1, "idem-question-effect", "question-effect", false)
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustDirectorQuestionRaisedEventWithKeyV0(t, "evt-question-effect-conflict", applied.LastSequence+1, "idem-question-effect-conflict", "question-effect", false)

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func mustDirectorQuestionRaisedEventWithKeyV0(
	t *testing.T,
	eventID string,
	sequence int64,
	idempotencyKey string,
	questionID string,
	blocking bool,
) OrchestrationEventV0 {
	t.Helper()
	payload := validAskDirectorPayloadV0(blocking)
	payload.QuestionID = questionID
	eventPayload := questionRaisedPayloadV0(DirectorQuestionV0{
		QuestionID:   payload.QuestionID,
		RunID:        validCommandMetaV0("cmd-question-event", idempotencyKey).RunID,
		SourceGroup:  payload.SourceGroup,
		TargetGroup:  payload.TargetGroup,
		Summary:      payload.Summary,
		Options:      payload.Options,
		EvidenceRefs: payload.EvidenceRefs,
		Blocking:     payload.Blocking,
		RequestedAt:  validCommandMetaV0("cmd-question-event", idempotencyKey).OccurredAt,
	})
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewDirectorQuestionRaisedEventV0(meta, eventPayload)
	return mustReducerEventV0(t, event, err)
}
