package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestHandleAskDirectorCommandV0EmitsQuestionEventAndDirectorOutbox(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAskDirectorCommandV0(t, "cmd-ask-001", "idem-ask-001", false)

	result, err := HandleAskDirectorCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle AskDirector: %v", err)
	}

	if len(result.Events) != 1 {
		t.Fatalf("events=%d, want 1", len(result.Events))
	}
	event := result.Events[0]
	if event.EventType != OrchestrationEventDirectorQuestionRaisedV0 {
		t.Fatalf("event_type=%q, want %q", event.EventType, OrchestrationEventDirectorQuestionRaisedV0)
	}
	if err := ValidateDirectorQuestionRaisedEventV0(event); err != nil {
		t.Fatalf("director question event rejected: %v", err)
	}
	assertSingleDirectorOutboxV0(t, result, event.EventID)
}

func TestHandleAskDirectorCommandV0BlockingAlsoEmitsRunBlocked(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAskDirectorCommandV0(t, "cmd-ask-002", "idem-ask-002", true)

	result, err := HandleAskDirectorCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle blocking AskDirector: %v", err)
	}

	if len(result.Events) != 2 {
		t.Fatalf("events=%d, want 2", len(result.Events))
	}
	if result.Events[0].EventType != OrchestrationEventDirectorQuestionRaisedV0 {
		t.Fatalf("first event=%q, want question", result.Events[0].EventType)
	}
	if result.Events[1].EventType != OrchestrationEventRunBlockedV0 {
		t.Fatalf("second event=%q, want RunBlocked", result.Events[1].EventType)
	}
	if result.Events[1].Sequence != result.Events[0].Sequence+1 {
		t.Fatalf("block sequence=%d, want %d", result.Events[1].Sequence, result.Events[0].Sequence+1)
	}
	questioned := mustApplyReducerEventV0(t, run, result.Events[0])
	next := mustApplyReducerEventV0(t, questioned, result.Events[1])
	if !directorQuestionRefAlreadyReflectedV0(next, "question-ask-001") {
		t.Fatalf("question not reflected: %+v", next.DirectorQuestions)
	}
	if !blockerAlreadyReflectedV0(next, "director-question-question-ask-001") {
		t.Fatalf("blocking question not reflected as blocker: %+v", next.Blockers)
	}
	assertSingleDirectorOutboxV0(t, result, result.Events[0].EventID)
}

func TestHandleAskDirectorCommandV0KeepsQuestionPayloadCompact(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAskDirectorCommandV0(t, "cmd-ask-003", "idem-ask-003", true)

	result, err := HandleAskDirectorCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle AskDirector: %v", err)
	}

	var question DirectorQuestionV0
	if err := json.Unmarshal(result.Outbox[0].Payload, &question); err != nil {
		t.Fatalf("unmarshal outbox payload: %v", err)
	}
	if len(question.Summary) > maxDirectorQuestionSummaryLenV0 {
		t.Fatalf("summary len=%d, max=%d", len(question.Summary), maxDirectorQuestionSummaryLenV0)
	}
	if len(question.Options) != 2 || len(question.EvidenceRefs) != 1 {
		t.Fatalf("unexpected compact question payload: %+v", question)
	}
}

func TestHandleAskDirectorCommandV0ReturnsPublicErrors(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAskDirectorCommandV0(t, "cmd-ask-004", "idem-ask-004", false)
	command.Payload = json.RawMessage(`{"question_id":"question-ask-001","source_group":"workflow","summary":" "}`)

	_, err := HandleAskDirectorCommandV0(run, command)
	var questionErr DirectorQuestionErrorV0
	if !errors.As(err, &questionErr) {
		t.Fatalf("expected public director question error, got %T %v", err, err)
	}
	if questionErr.Code != ErrDirectorQuestionInvalidaV0 || questionErr.Field != "summary" {
		t.Fatalf("error=%+v, want summary question error", questionErr)
	}

	command = mustAskDirectorCommandV0(t, "cmd-ask-005", "idem-ask-005", false)
	_, err = HandleAskDirectorCommandV0(OrchestrationRunV0{}, command)
	var commandErr OrchestrationCommandErrorV0
	if !errors.As(err, &commandErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if commandErr.Code != ErrTransicionInvalidaV0 {
		t.Fatalf("code=%q, want %q", commandErr.Code, ErrTransicionInvalidaV0)
	}
}

func TestValidateOrchestrationCommandV0WrapsAskDirectorPayloadErrors(t *testing.T) {
	command := mustAskDirectorCommandV0(t, "cmd-ask-wrap", "idem-ask-wrap", false)
	command.Payload = json.RawMessage(`{"question_id":"question-ask-001","source_group":"workflow","summary":" "}`)

	err := ValidateOrchestrationCommandV0(command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 || publicErr.Field != "payload.summary" {
		t.Fatalf("error=%+v, want payload.summary", publicErr)
	}
}

func TestHandleAskDirectorCommandV0RetriesOutboxWhenBlockingQuestionAlreadyReflected(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAskDirectorCommandV0(t, "cmd-ask-006", "idem-ask-006", true)
	first, err := HandleAskDirectorCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle first AskDirector: %v", err)
	}
	questioned := mustApplyReducerEventV0(t, run, first.Events[0])
	blocked := mustApplyReducerEventV0(t, questioned, first.Events[1])

	result, err := HandleAskDirectorCommandV0(blocked, command)
	if err != nil {
		t.Fatalf("retry AskDirector: %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("events=%d, want none", len(result.Events))
	}
	assertSingleDirectorOutboxV0(t, result, first.Events[0].EventID)
}

func TestHandleAskDirectorCommandV0RepairsQuestionMissingButBlockerPresent(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	run.Status = OrchestrationRunStatusBlockedV0
	run.Blockers = []string{"director-question-question-ask-001"}
	command := mustAskDirectorCommandV0(t, "cmd-ask-repair", "idem-ask-repair", true)

	result, err := HandleAskDirectorCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle repair AskDirector: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventDirectorQuestionRaisedV0)
	assertSingleDirectorOutboxV0(t, result, result.Events[0].EventID)
}

func TestAskDirectorCommandV0IsSupportedByGenericHandler(t *testing.T) {
	start := mustStartRunCommandV0(t, "cmd-start-still-valid", "idem-start-still-valid")
	if err := ValidateOrchestrationCommandV0(start); err != nil {
		t.Fatalf("StartRun validation regressed: %v", err)
	}

	ask := mustAskDirectorCommandV0(t, "cmd-ask-007", "idem-ask-007", false)
	if err := ValidateOrchestrationCommandV0(ask); err != nil {
		t.Fatalf("generic command validator rejected AskDirector: %v", err)
	}
	result, err := HandleCommandV0(mustHandlerStartedRunV0(t), ask)
	if err != nil {
		t.Fatalf("generic handler rejected AskDirector: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventDirectorQuestionRaisedV0)
}

func TestHandleAskDirectorCommandV0RetriesOutboxWhenNonBlockingQuestionAlreadyReflected(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustAskDirectorCommandV0(t, "cmd-ask-008", "idem-ask-008", false)
	first, err := HandleAskDirectorCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle first AskDirector: %v", err)
	}
	questioned := mustApplyReducerEventV0(t, run, first.Events[0])

	result, err := HandleAskDirectorCommandV0(questioned, command)
	if err != nil {
		t.Fatalf("retry AskDirector: %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("events=%d, want none", len(result.Events))
	}
	assertSingleDirectorOutboxV0(t, result, first.Events[0].EventID)
}

func TestHandleAskDirectorCommandV0DoesNotReblockAnsweredQuestion(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	ask := mustAskDirectorCommandV0(t, "cmd-ask-answer-repeat", "idem-ask-answer-repeat", true)
	first, err := HandleAskDirectorCommandV0(run, ask)
	if err != nil {
		t.Fatalf("handle first AskDirector: %v", err)
	}
	for _, event := range first.Events {
		run = mustApplyReducerEventV0(t, run, event)
	}
	answer := mustAnswerDirectorQuestionCommandV0(t, "cmd-answer-before-ask-repeat", "idem-answer-before-ask-repeat", true)
	answered := mustApplySingleCommandEventV0(t, run, answer)

	result, err := HandleAskDirectorCommandV0(answered, ask)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestValidateAskDirectorCommandV0RejectsForbiddenDetails(t *testing.T) {
	cases := map[string]func(*AskDirectorCommandPayloadV0){
		"api_key": func(payload *AskDirectorCommandPayloadV0) { payload.Summary = "depende de api_key=valor" },
		"authorization": func(payload *AskDirectorCommandPayloadV0) {
			payload.EvidenceRefs = []string{"authorization: bearer valor"}
		},
		"client_secret": func(payload *AskDirectorCommandPayloadV0) { payload.Options = []string{"client_secret=valor"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			payload := validAskDirectorPayloadV0(false)
			mutate(&payload)

			_, err := NewAskDirectorCommandV0(validCommandMetaV0("cmd-ask-forbidden-"+name, "idem-ask-forbidden-"+name), payload)
			var publicErr OrchestrationCommandErrorV0
			if !errors.As(err, &publicErr) {
				t.Fatalf("expected public command error, got %T %v", err, err)
			}
			if publicErr.Code != ErrDetalleProhibidoV0 {
				t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
			}
		})
	}
}

func mustAskDirectorCommandV0(t *testing.T, commandID string, idempotencyKey string, blocking bool) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewAskDirectorCommandV0(validCommandMetaV0(commandID, idempotencyKey), validAskDirectorPayloadV0(blocking))
	if err != nil {
		t.Fatalf("AskDirector constructor failed: %v", err)
	}
	return command
}

func validAskDirectorPayloadV0(blocking bool) AskDirectorCommandPayloadV0 {
	return AskDirectorCommandPayloadV0{
		QuestionID:   "question-ask-001",
		SourceGroup:  "workflow",
		TargetGroup:  "director",
		Summary:      "Falta decidir si la consulta bloquea el avance de esta fase.",
		Options:      []string{"Continuar sin bloquear", "Bloquear hasta respuesta"},
		EvidenceRefs: []string{"docs/contratos.md#AskDirector"},
		Blocking:     blocking,
	}
}

func assertSingleDirectorOutboxV0(t *testing.T, result OrchestrationCommandResultV0, causationEventID string) {
	t.Helper()
	if len(result.Outbox) != 1 {
		t.Fatalf("outbox=%d, want 1", len(result.Outbox))
	}
	message := result.Outbox[0]
	if message.MessageType != OutboxMessageSendDirectorQuestionV0 {
		t.Fatalf("message_type=%q, want %q", message.MessageType, OutboxMessageSendDirectorQuestionV0)
	}
	if message.TargetPort != OutboxTargetDirectorV0 {
		t.Fatalf("target_port=%q, want %q", message.TargetPort, OutboxTargetDirectorV0)
	}
	if message.CausationEventID != causationEventID {
		t.Fatalf("causation_event_id=%q, want %q", message.CausationEventID, causationEventID)
	}
	if err := ValidateOutboxMessageV0(message); err != nil {
		t.Fatalf("outbox rejected: %v", err)
	}
}
