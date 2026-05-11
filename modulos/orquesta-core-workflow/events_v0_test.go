package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestOrchestrationEventV0SerializesWithoutForbiddenDetails(t *testing.T) {
	events := []OrchestrationEventV0{
		mustRunStartedEventV0(t, validEventMetaV0("evt-run-started-001", 1), RunStartedPayloadV0{
			ProjectRef:  "project:ventas",
			AppSpecRef:  "appspec:req-001",
			RequestedBy: "director",
		}),
		mustPhaseOpenedEventV0(t, validEventMetaV0("evt-phase-opened-001", 2), PhaseOpenedPayloadV0{
			PhaseID:  "descubrimiento",
			OpenedBy: "director",
			Reason:   "inicio de fase canonica",
		}),
		mustRunBlockedEventV0(t, validEventMetaV0("evt-run-blocked-001", 3), RunBlockedPayloadV0{
			BlockerID:    "blocker-001",
			ReasonCode:   "consulta_director_requerida",
			Summary:      "Falta una decision de contrato antes de continuar.",
			SourceGroup:  "workflow",
			EvidenceRefs: []string{"docs/contratos.md#OrchestrationEventV0"},
		}),
	}

	for _, event := range events {
		t.Run(event.EventType, func(t *testing.T) {
			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("marshal event: %v", err)
			}
			assertNoForbiddenFragmentsV0(t, string(data))
			if !json.Valid(data) {
				t.Fatalf("event JSON is invalid: %s", data)
			}
		})
	}
}

func TestValidateOrchestrationEventV0RejectsUnknownEvent(t *testing.T) {
	event := mustRunStartedEventV0(t, validEventMetaV0("evt-run-started-002", 1), RunStartedPayloadV0{
		ProjectRef: "project:ventas",
		AppSpecRef: "appspec:req-002",
	})
	event.EventType = "AgentUnknown"

	err := ValidateOrchestrationEventV0(event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public error, got %T %v", err, err)
	}
	if publicErr.Code != ErrEventoNoSoportadoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrEventoNoSoportadoV0)
	}
}

func TestValidateOrchestrationEventV0RejectsForbiddenPayloadDetails(t *testing.T) {
	event := mustRunBlockedEventV0(t, validEventMetaV0("evt-run-blocked-002", 1), RunBlockedPayloadV0{
		BlockerID:  "blocker-002",
		ReasonCode: "consulta_director_requerida",
		Summary:    "Bloqueado hasta recibir criterio de arquitectura.",
	})
	event.Payload = json.RawMessage(`{"blocker_id":"blocker-002","reason_code":"x","summary":"ver transcript completo"}`)

	err := ValidateOrchestrationEventV0(event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestValidateOrchestrationEventV0DoesNotRejectLegitimateWordsContainingForbiddenToken(t *testing.T) {
	event := mustRunBlockedEventV0(t, validEventMetaV0("evt-run-blocked-003", 1), RunBlockedPayloadV0{
		BlockerID:  "blocker-003",
		ReasonCode: "consulta_director_requerida",
		Summary:    "Decision legitima pendiente de criterio del director.",
	})

	if err := ValidateOrchestrationEventV0(event); err != nil {
		t.Fatalf("valid event rejected by substring false positive: %v", err)
	}
}

func TestEventConstructorsRequireMinimalPayload(t *testing.T) {
	_, err := NewPhaseOpenedEventV0(validEventMetaV0("evt-phase-opened-002", 1), PhaseOpenedPayloadV0{})
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 || publicErr.Field != "payload.phase_id" {
		t.Fatalf("error=%+v, want code=%s field=payload.phase_id", publicErr, ErrPayloadInvalidoV0)
	}
}

func validEventMetaV0(eventID string, sequence int64) OrchestrationEventMetaV0 {
	return OrchestrationEventMetaV0{
		EventID:        eventID,
		RunID:          "run-001",
		Sequence:       sequence,
		IdempotencyKey: "idem-001",
		CorrelationID:  "corr-001",
		CausationID:    "cmd-001",
		OccurredAt:     "2026-05-04T10:00:00Z",
	}
}

func mustRunStartedEventV0(t *testing.T, meta OrchestrationEventMetaV0, payload RunStartedPayloadV0) OrchestrationEventV0 {
	t.Helper()
	event, err := NewRunStartedEventV0(meta, payload)
	return mustEventV0(t, event, err)
}

func mustPhaseOpenedEventV0(t *testing.T, meta OrchestrationEventMetaV0, payload PhaseOpenedPayloadV0) OrchestrationEventV0 {
	t.Helper()
	event, err := NewPhaseOpenedEventV0(meta, payload)
	return mustEventV0(t, event, err)
}

func mustRunBlockedEventV0(t *testing.T, meta OrchestrationEventMetaV0, payload RunBlockedPayloadV0) OrchestrationEventV0 {
	t.Helper()
	event, err := NewRunBlockedEventV0(meta, payload)
	return mustEventV0(t, event, err)
}

func mustEventV0(t *testing.T, event OrchestrationEventV0, err error) OrchestrationEventV0 {
	t.Helper()
	if err != nil {
		t.Fatalf("event constructor failed: %v", err)
	}
	return event
}

func assertNoForbiddenFragmentsV0(t *testing.T, serialized string) {
	t.Helper()
	lower := strings.ToLower(serialized)
	for _, fragment := range forbiddenEventFragmentsV0 {
		if strings.Contains(lower, fragment) {
			t.Fatalf("serialized event contains forbidden fragment %q: %s", fragment, serialized)
		}
	}
}
