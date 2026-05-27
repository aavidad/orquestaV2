package orquestaobservability

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeOrquestaEventV0FixtureValido(t *testing.T) {
	event, err := DecodeOrquestaEventV0(readOrquestaEventFixtureV0(t, "core_event_minimo_valido.json"))
	if err != nil {
		t.Fatalf("fixture should validate: %v", err)
	}
	if event.SchemaVersion != OrquestaEventSchemaVersionV0 {
		t.Fatalf("schema_version=%q", event.SchemaVersion)
	}
	if event.SourceArea != OrquestaEventSourceAreaCoreV0 || !strings.HasPrefix(event.EventType, event.SourceArea+".") {
		t.Fatalf("source_area/event_type mismatch: %+v", event)
	}
	if event.Privacy.ContainsSecret || event.Privacy.ContainsTranscript {
		t.Fatalf("privacy flags should be false: %+v", event.Privacy)
	}
}

func TestDecodeOrquestaEventV0FixturesInvalidos(t *testing.T) {
	tests := []struct {
		name string
		file string
		code string
	}{
		{
			name: "transcript completo",
			file: "transcript_completo_invalido.json",
			code: ErrTranscriptNoPermitidoV0,
		},
		{
			name: "secreto detectado",
			file: "secreto_detectado_invalido.json",
			code: ErrSecretoDetectadoV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeOrquestaEventV0(readOrquestaEventFixtureV0(t, test.file))
			assertOrquestaEventIssueV0(t, err, test.code)
		})
	}
}

func TestValidatePublishOrquestaEventRequestV0AcceptedSinSink(t *testing.T) {
	event := validOrquestaEventV0(t)
	request := PublishOrquestaEventRequestV0{
		RequestID:      "req_20260504_000010",
		CorrelationID:  event.Correlation.CorrelationID,
		IdempotencyKey: "idem_20260504_000010",
		Event:          event,
		DryRun:         true,
	}

	accepted, err := AcceptPublishOrquestaEventRequestV0(request)
	if err != nil {
		t.Fatalf("request should validate: %v", err)
	}
	if !accepted.Accepted {
		t.Fatalf("accepted should be true")
	}
	if accepted.EventID != event.EventID || accepted.CorrelationID != request.CorrelationID {
		t.Fatalf("accepted echo mismatch: %+v", accepted)
	}
	if accepted.Stored || accepted.SinkReceiptID != "" {
		t.Fatalf("microcut must not store or emit sink receipt: %+v", accepted)
	}
}

func TestValidatePublishOrquestaEventRequestV0EnvelopeRequerido(t *testing.T) {
	event := validOrquestaEventV0(t)

	t.Run("correlation_id requerido", func(t *testing.T) {
		request := PublishOrquestaEventRequestV0{
			IdempotencyKey: "idem_20260504_000011",
			Event:          event,
		}
		assertOrquestaEventIssueV0(t, ValidatePublishOrquestaEventRequestV0(request), ErrCorrelationIDRequeridoV0)
	})

	t.Run("idempotency_key requerida", func(t *testing.T) {
		request := PublishOrquestaEventRequestV0{
			CorrelationID: event.Correlation.CorrelationID,
			Event:         event,
		}
		assertOrquestaEventIssueV0(t, ValidatePublishOrquestaEventRequestV0(request), ErrIdempotencyKeyRequeridaV0)
	})

	t.Run("correlacion incompatible", func(t *testing.T) {
		request := PublishOrquestaEventRequestV0{
			CorrelationID:  "corr_20260504_distinta",
			IdempotencyKey: "idem_20260504_000012",
			Event:          event,
		}
		assertOrquestaEventIssueV0(t, ValidatePublishOrquestaEventRequestV0(request), ErrOrquestaEventInvalidoV0)
	})
}

func TestValidateOrquestaEventV0SourceAreaEventType(t *testing.T) {
	base := validOrquestaEventV0(t)

	t.Run("source_area no soportada", func(t *testing.T) {
		event := cloneOrquestaEventV0(t, base)
		event.SourceArea = "deploy"
		assertOrquestaEventIssueV0(t, ValidateOrquestaEventV0(event), ErrSourceAreaNoSoportadaV0)
	})

	t.Run("event_type incompatible", func(t *testing.T) {
		event := cloneOrquestaEventV0(t, base)
		event.SourceArea = OrquestaEventSourceAreaRuntimeV0
		event.EventType = "core.project_draft.accepted"
		assertOrquestaEventIssueV0(t, ValidateOrquestaEventV0(event), ErrEventTypeIncompatibleV0)
	})

	t.Run("event_type no compacto", func(t *testing.T) {
		event := cloneOrquestaEventV0(t, base)
		event.EventType = "core.ProjectDraft.Accepted"
		assertOrquestaEventIssueV0(t, ValidateOrquestaEventV0(event), ErrOrquestaEventInvalidoV0)
	})
}

func TestValidateOrquestaEventV0PrivacyFalse(t *testing.T) {
	base := validOrquestaEventV0(t)

	t.Run("secret false obligatorio", func(t *testing.T) {
		event := cloneOrquestaEventV0(t, base)
		event.Privacy.ContainsSecret = true
		assertOrquestaEventIssueV0(t, ValidateOrquestaEventV0(event), ErrSecretoDetectadoV0)
	})

	t.Run("transcript false obligatorio", func(t *testing.T) {
		event := cloneOrquestaEventV0(t, base)
		event.Privacy.ContainsTranscript = true
		assertOrquestaEventIssueV0(t, ValidateOrquestaEventV0(event), ErrTranscriptNoPermitidoV0)
	})
}

func TestValidateOrquestaEventV0PayloadCompacto(t *testing.T) {
	base := validOrquestaEventV0(t)

	t.Run("demasiadas claves", func(t *testing.T) {
		event := cloneOrquestaEventV0(t, base)
		event.Payload = map[string]any{}
		for index := 0; index < maxPayloadPropertiesV0+1; index++ {
			event.Payload["item_"+string(rune('a'+index))] = index
		}
		assertOrquestaEventIssueV0(t, ValidateOrquestaEventV0(event), ErrEventoDemasiadoExtensoV0)
	})

	t.Run("string demasiado largo", func(t *testing.T) {
		event := cloneOrquestaEventV0(t, base)
		event.Payload["status"] = strings.Repeat("a", maxPayloadStringRunesV0+1)
		assertOrquestaEventIssueV0(t, ValidateOrquestaEventV0(event), ErrEventoDemasiadoExtensoV0)
	})

	t.Run("array demasiado largo", func(t *testing.T) {
		event := cloneOrquestaEventV0(t, base)
		values := make([]any, maxPayloadArrayItemsV0+1)
		for index := range values {
			values[index] = index
		}
		event.Payload["durations_ms"] = values
		assertOrquestaEventIssueV0(t, ValidateOrquestaEventV0(event), ErrEventoDemasiadoExtensoV0)
	})
}

func TestValidateOrquestaEventV0RechazaClavesProhibidas(t *testing.T) {
	tests := []struct {
		key  string
		code string
	}{
		{key: "api_key_hint", code: ErrSecretoDetectadoV0},
		{key: "access_token_value", code: ErrSecretoDetectadoV0},
		{key: "transcript_text", code: ErrTranscriptNoPermitidoV0},
		{key: "prompt_text", code: ErrOrquestaEventInvalidoV0},
		{key: "completion_text", code: ErrOrquestaEventInvalidoV0},
		{key: "sql_query", code: ErrOrquestaEventInvalidoV0},
		{key: "dsn_value", code: ErrOrquestaEventInvalidoV0},
		{key: "tabla_name", code: ErrOrquestaEventInvalidoV0},
		{key: "home_path", code: ErrOrquestaEventInvalidoV0},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			event := validOrquestaEventV0(t)
			event.Payload[test.key] = "ref_20260504_000001"
			assertOrquestaEventIssueV0(t, ValidateOrquestaEventV0(event), test.code)
		})
	}
}

func TestValidateOrquestaEventV0PermiteRefsPrivacidadOpacas(t *testing.T) {
	for _, key := range []string{"token_policy_ref", "payload_redaction_ref", "transcript_ref", "home_ref", "dsn_ref"} {
		t.Run(key, func(t *testing.T) {
			event := validOrquestaEventV0(t)
			event.Payload[key] = "privacy-policy-ref-20260504-000001"
			if err := ValidateOrquestaEventV0(event); err != nil {
				t.Fatalf("event should allow opaque privacy ref %q: %v", key, err)
			}
		})
	}
}

func readOrquestaEventFixtureV0(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("docs", "fixtures", "orquesta_event_v0", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func validOrquestaEventV0(t *testing.T) OrquestaEventV0 {
	t.Helper()
	event, err := DecodeOrquestaEventV0(readOrquestaEventFixtureV0(t, "core_event_minimo_valido.json"))
	if err != nil {
		t.Fatalf("decode valid fixture: %v", err)
	}
	return event
}

func cloneOrquestaEventV0(t *testing.T, event OrquestaEventV0) OrquestaEventV0 {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	var clone OrquestaEventV0
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	return clone
}

func assertOrquestaEventIssueV0(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	var validationErr OrquestaEventValidationErrorV0
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type=%T, want OrquestaEventValidationErrorV0", err)
	}
	for _, issue := range validationErr.Issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("missing issue %q in %+v", want, validationErr.Issues)
}
