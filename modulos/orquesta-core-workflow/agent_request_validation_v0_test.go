package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestApplyAgentRequestedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustReducerRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	event := mustAgentRequestedEventV0(t, "evt-agent-effect-first", run.LastSequence+1, "agent-request-effect")
	applied := mustApplyReducerEventV0(t, run, event)
	conflict := mustAgentRequestedEventWithKeyV0(t, "evt-agent-effect-second", applied.LastSequence+1, "idem-agent-effect-second", "agent-request-effect")

	_, err := ApplyEventV0(applied, conflict)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrEventoConflictivoV0 || publicErr.Field != "idempotency" {
		t.Fatalf("error=%+v, want evento_conflictivo idempotency", publicErr)
	}
}

func TestHandleRequestAgentCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	command := mustRequestAgentCommandV0(t, "cmd-agent-effect-conflict", "idem-agent-effect-conflict", "agent-request-effect-conflict")
	created := mustApplySingleCommandEventV0(t, run, command)
	payload := validRequestAgentPayloadV0("agent-request-effect-conflict")
	payload.Role = "revision"
	conflict, err := NewRequestAgentCommandV0(validCommandMetaV0("cmd-agent-effect-conflict", "idem-agent-effect-conflict"), payload)
	conflict = mustCommandV0(t, conflict, err)

	_, err = HandleCommandV0(created, conflict)
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestHandleRequestAgentCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	command := mustRequestAgentCommandV0(t, "cmd-agent-key-conflict", "idem-agent-key-conflict", "agent-request-key-conflict")
	created := mustApplySingleCommandEventV0(t, run, command)
	conflict := mustRequestAgentCommandV0(t, "cmd-agent-key-conflict-2", "idem-agent-key-conflict-2", "agent-request-key-conflict")

	_, err := HandleCommandV0(created, conflict)
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestRequestAgentCommandV0RequiresCapacityRequestRef(t *testing.T) {
	payload := validRequestAgentPayloadV0("agent-request-no-capacity-ref")
	payload.CapacityRequestRef = ""

	_, err := NewRequestAgentCommandV0(validCommandMetaV0("cmd-agent-no-capacity-ref", "idem-agent-no-capacity-ref"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 || publicErr.Field != "payload.capacity_request_ref" {
		t.Fatalf("error=%+v, want payload.capacity_request_ref", publicErr)
	}
}

func TestHandleRequestAgentCommandV0RequiresCapacityDecision(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRequestAgentCommandV0(t, "cmd-agent-no-capacity-decision", "idem-agent-no-capacity-decision", "agent-request-no-capacity-decision")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 || publicErr.Field != "payload.capacity_request_ref" {
		t.Fatalf("error=%+v, want transicion_invalida payload.capacity_request_ref", publicErr)
	}
}

func TestApplyAgentRequestedV0RequiresCapacityDecision(t *testing.T) {
	run := mustReducerProgramacionRunV0(t)
	event := mustAgentRequestedEventV0(t, "evt-agent-no-capacity-decision", run.LastSequence+1, "agent-request-no-capacity-decision")

	_, err := ApplyEventV0(run, event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrSecuenciaInvalidaV0 || publicErr.Field != "payload.capacity_request_ref" {
		t.Fatalf("error=%+v, want secuencia_invalida payload.capacity_request_ref", publicErr)
	}
}

func TestHandleRequestAgentCommandV0RejectsNonCurrentPhase(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustRequestAgentCommandV0(t, "cmd-agent-phase", "idem-agent-phase", "agent-request-phase")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 || publicErr.Field != "payload.phase_id" {
		t.Fatalf("error=%+v, want transicion_invalida payload.phase_id", publicErr)
	}
}

func TestRequestAgentCommandV0PermiteDetallesOperativosOpacos(t *testing.T) {
	cases := map[string]func(*RequestAgentCommandPayloadV0){
		"provider": func(payload *RequestAgentCommandPayloadV0) { payload.Summary = "usar provider externo" },
		"modelo":   func(payload *RequestAgentCommandPayloadV0) { payload.Role = "modelo" },
		"runtime":  func(payload *RequestAgentCommandPayloadV0) { payload.TaskRef = "runtime-task" },
		"HOME":     func(payload *RequestAgentCommandPayloadV0) { payload.EvidenceRefs = []string{"$HOME/estado"} },
		"OAuth":    func(payload *RequestAgentCommandPayloadV0) { payload.CapacityRequestRef = "oauth-ref" },
		"Codex":    func(payload *RequestAgentCommandPayloadV0) { payload.Summary = "usar Codex" },
		"Claude":   func(payload *RequestAgentCommandPayloadV0) { payload.Summary = "usar Claude" },
		"Ollama":   func(payload *RequestAgentCommandPayloadV0) { payload.Summary = "usar Ollama" },
		"vLLM":     func(payload *RequestAgentCommandPayloadV0) { payload.Summary = "usar vLLM" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			payload := validRequestAgentPayloadV0("agent-request-forbidden")
			mutate(&payload)

			if _, err := NewRequestAgentCommandV0(validCommandMetaV0("cmd-agent-opaque-"+strings.ToLower(name), "idem-agent-opaque-"+strings.ToLower(name)), payload); err != nil {
				t.Fatalf("detalle operativo opaco rechazado: %v", err)
			}
		})
	}
}

func TestRequestAgentCommandV0RejectsSensitiveDetails(t *testing.T) {
	payload := validRequestAgentPayloadV0("agent-request-sensitive")
	payload.EvidenceRefs = []string{"client_secret=abc123"}

	_, err := NewRequestAgentCommandV0(validCommandMetaV0("cmd-agent-sensitive", "idem-agent-sensitive"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestRequestAgentCommandV0RejectsInvalidSkillRefs(t *testing.T) {
	for name, refs := range map[string][]string{
		"with_space":          {"skill ref invalid"},
		"with_path_separator": {"skills/orquesta-programacion-autonoma"},
	} {
		t.Run(name, func(t *testing.T) {
			payload := validRequestAgentPayloadV0("agent-request-invalid-skill")
			payload.SkillRefs = refs

			_, err := NewRequestAgentCommandV0(
				validCommandMetaV0("cmd-agent-invalid-skill-"+name, "idem-agent-invalid-skill-"+name),
				payload,
			)
			var publicErr OrchestrationCommandErrorV0
			if !errors.As(err, &publicErr) {
				t.Fatalf("expected command error, got %T %v", err, err)
			}
			if publicErr.Code != ErrPayloadInvalidoV0 || publicErr.Field != "payload.skill_refs" {
				t.Fatalf("error=%+v, want payload.skill_refs", publicErr)
			}
		})
	}
}

func TestReplayDurableEventsV0RejectsForbiddenAgentRequested(t *testing.T) {
	event := mustAgentRequestedEventWithKeyV0(t, "evt-agent-forbidden", 2, "idem-agent-forbidden", "agent-request-forbidden")
	event.Payload = json.RawMessage(`{"agent_request_id":"agent-request-forbidden","phase_id":"programacion","capacity_request_ref":"capacity-request-010","role":"implementacion","summary":"client_secret=abc123"}`)
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-agent-forbidden", 1, "idem-start-agent-forbidden"),
		event,
	}

	_, err := ReplayDurableEventsV0(events)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestAgentRequestedOutboxPayloadDoesNotContainRuntimeProviderModelHome(t *testing.T) {
	run := mustRunWithCapacityDecisionV0(t, defaultAgentCapacityRequestIDV0)
	command := mustRequestAgentCommandV0(t, "cmd-agent-clean", "idem-agent-clean", "agent-request-clean")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RequestAgent: %v", err)
	}
	serialized := strings.ToLower(string(result.Outbox[0].Payload))
	for _, forbidden := range operationalSensitiveFragmentsForTestV0() {
		if containsForbiddenFragmentV0(serialized, forbidden) {
			t.Fatalf("outbox payload contains forbidden fragment %q: %s", forbidden, serialized)
		}
	}
}

func TestValidateLaunchRuntimeAgentOutboxRejectsInvalidPayload(t *testing.T) {
	message := OutboxMessageV0{
		MessageID:      "outbox-agent-invalid",
		MessageType:    OutboxMessageLaunchRuntimeAgentV0,
		RunID:          "run-001",
		IdempotencyKey: "idem-agent-invalid",
		TargetPort:     OutboxTargetAgentLauncherV0,
		PayloadVersion: OutboxPayloadVersionV0,
		Payload:        json.RawMessage(`{}`),
	}

	err := ValidateOutboxMessageV0(message)
	var publicErr OutboxMessageErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public outbox error, got %T %v", err, err)
	}
	if publicErr.Code != ErrOutboxPayloadInvalidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrOutboxPayloadInvalidoV0)
	}
}
