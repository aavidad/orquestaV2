package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestSendDirectorQuestionOutboxV0SerializesWithoutAdaptersOrSecrets(t *testing.T) {
	message := mustSendDirectorQuestionOutboxV0(t)

	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal outbox: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("outbox JSON is invalid: %s", data)
	}
	assertOutboxNoForbiddenFragmentsV0(t, string(data))
	if err := ValidateOutboxMessageV0(message); err != nil {
		t.Fatalf("valid outbox rejected: %v", err)
	}
}

func TestValidateOutboxMessageV0RejectsUnknownType(t *testing.T) {
	message := mustSendDirectorQuestionOutboxV0(t)
	message.MessageType = "SendHTTPRequest"

	err := ValidateOutboxMessageV0(message)
	var publicErr OutboxMessageErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public outbox error, got %T %v", err, err)
	}
	if publicErr.Code != ErrOutboxTipoNoSoportadoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrOutboxTipoNoSoportadoV0)
	}
}

func TestDirectorQuestionV0RequiresSummaryAndSource(t *testing.T) {
	tests := []struct {
		name  string
		field string
		edit  func(DirectorQuestionV0) DirectorQuestionV0
	}{
		{
			name:  "summary",
			field: "summary",
			edit: func(question DirectorQuestionV0) DirectorQuestionV0 {
				question.Summary = " "
				return question
			},
		},
		{
			name:  "source",
			field: "source_group",
			edit: func(question DirectorQuestionV0) DirectorQuestionV0 {
				question.SourceGroup = " "
				return question
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDirectorQuestionV0(tt.edit(validDirectorQuestionV0()))
			var publicErr DirectorQuestionErrorV0
			if !errors.As(err, &publicErr) {
				t.Fatalf("expected public director question error, got %T %v", err, err)
			}
			if publicErr.Code != ErrDirectorQuestionInvalidaV0 || publicErr.Field != tt.field {
				t.Fatalf("error=%+v, want code=%s field=%s", publicErr, ErrDirectorQuestionInvalidaV0, tt.field)
			}
		})
	}
}

func TestValidateOutboxMessageV0WrapsDirectorQuestionErrors(t *testing.T) {
	message := mustSendDirectorQuestionOutboxV0(t)
	message.Payload = json.RawMessage(`{
		"question_id":"question-001",
		"run_id":"run-001",
		"source_group":"",
		"summary":"Falta decidir si este contrato local se promueve.",
		"requested_at":"2026-05-04T10:00:00Z"
	}`)

	err := ValidateOutboxMessageV0(message)
	var publicErr OutboxMessageErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected outbox public error, got %T %v", err, err)
	}
	if publicErr.Code != ErrOutboxPayloadInvalidoV0 || publicErr.Field != "payload.source_group" {
		t.Fatalf("error=%+v, want payload.source_group", publicErr)
	}
}

func TestValidateOutboxMessageV0RejectsMassiveContextPayload(t *testing.T) {
	message := mustSendDirectorQuestionOutboxV0(t)
	message.Payload = json.RawMessage(`{
		"question_id":"question-001",
		"run_id":"run-001",
		"source_group":"workflow",
		"target_group":"director",
		"summary":"Falta decidir si este contrato local se promueve.",
		"requested_at":"2026-05-04T10:00:00Z",
		"massive_context":"` + strings.Repeat("x", 128) + `"
	}`)

	err := ValidateOutboxMessageV0(message)
	var publicErr OutboxMessageErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public outbox error, got %T %v", err, err)
	}
	if publicErr.Code != ErrOutboxDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrOutboxDetalleProhibidoV0)
	}
}

func TestValidateOutboxMessageV0DoesNotRejectLegitimateWordsContainingForbiddenToken(t *testing.T) {
	question := validDirectorQuestionV0()
	question.Summary = "Consulta legitima para cerrar una decision local."
	message, err := NewSendDirectorQuestionOutboxV0(validOutboxMetaV0(), question)
	if err != nil {
		t.Fatalf("constructor rejected legitimate summary: %v", err)
	}

	if err := ValidateOutboxMessageV0(message); err != nil {
		t.Fatalf("valid outbox rejected by substring false positive: %v", err)
	}
}

func TestLaunchAgentOutboxTargetDoesNotUseForbiddenRuntimeWord(t *testing.T) {
	if containsForbiddenOutboxTextV0(OutboxTargetAgentLauncherV0) {
		t.Fatalf("agent launcher target contains forbidden detail: %q", OutboxTargetAgentLauncherV0)
	}
	if got := expectedOutboxTargetPortV0(OutboxMessageLaunchRuntimeAgentV0); got != OutboxTargetAgentLauncherV0 {
		t.Fatalf("target=%q, want %q", got, OutboxTargetAgentLauncherV0)
	}
}

func TestSendDirectorQuestionOutboxV0PayloadIsCompact(t *testing.T) {
	message := mustSendDirectorQuestionOutboxV0(t)

	if len(message.Payload) > maxOutboxPayloadBytesV0 {
		t.Fatalf("payload size=%d, max=%d", len(message.Payload), maxOutboxPayloadBytesV0)
	}
	var payload map[string]any
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	for _, key := range forbiddenOutboxPayloadKeysV0 {
		if _, exists := payload[key]; exists {
			t.Fatalf("payload includes massive-context key %q: %s", key, string(message.Payload))
		}
	}
}

func mustSendDirectorQuestionOutboxV0(t *testing.T) OutboxMessageV0 {
	t.Helper()
	message, err := NewSendDirectorQuestionOutboxV0(validOutboxMetaV0(), validDirectorQuestionV0())
	if err != nil {
		t.Fatalf("outbox constructor failed: %v", err)
	}
	return message
}

func validOutboxMetaV0() OutboxMessageMetaV0 {
	return OutboxMessageMetaV0{
		MessageID:        "outbox-001",
		RunID:            "run-001",
		IdempotencyKey:   "idem-outbox-001",
		CorrelationID:    "corr-outbox-001",
		CausationEventID: "evt-director-question-001",
	}
}

func validDirectorQuestionV0() DirectorQuestionV0 {
	return DirectorQuestionV0{
		QuestionID:   "question-001",
		RunID:        "run-001",
		SourceGroup:  "workflow",
		TargetGroup:  "director",
		Summary:      "Falta decidir si este contrato local se promueve o se mantiene interno.",
		Options:      []string{"Mantenerlo local", "Elevarlo como contrato global"},
		EvidenceRefs: []string{"docs/contratos.md#OutboxMessageV0"},
		Blocking:     true,
		RequestedAt:  "2026-05-04T10:00:00Z",
	}
}

func assertOutboxNoForbiddenFragmentsV0(t *testing.T, serialized string) {
	t.Helper()
	forbidden := append([]string{}, forbiddenOutboxFragmentsV0...)
	forbidden = append(forbidden, forbiddenOutboxPayloadKeysV0...)
	forbidden = append(forbidden, "db", "database")

	lower := strings.ToLower(serialized)
	for _, fragment := range forbidden {
		if strings.Contains(lower, fragment) {
			t.Fatalf("serialized outbox contains forbidden fragment %q: %s", fragment, serialized)
		}
	}
}
