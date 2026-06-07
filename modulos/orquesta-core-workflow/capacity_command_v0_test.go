package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestHandleRequestCapacityCommandV0ReturnsEventAndOutbox(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRequestCapacityCommandV0(t, "cmd-capacity-001", "idem-capacity-001", "capacity-request-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RequestCapacity: %v", err)
	}

	assertSingleEventTypeV0(t, result, OrchestrationEventCapacityRequestedV0)
	if len(result.Outbox) != 1 {
		t.Fatalf("outbox=%d, want 1", len(result.Outbox))
	}
	outbox := result.Outbox[0]
	if outbox.MessageType != OutboxMessageRequestCapacityDecisionV0 || outbox.TargetPort != OutboxTargetCapacityV0 {
		t.Fatalf("unexpected outbox envelope: %+v", outbox)
	}
	if outbox.CausationEventID != result.Events[0].EventID {
		t.Fatalf("causation_event_id=%q, want %q", outbox.CausationEventID, result.Events[0].EventID)
	}
	var payload CapacityDecisionRequestV0
	if err := json.Unmarshal(outbox.Payload, &payload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if payload.CapacityRequestID != "capacity-request-001" || payload.RunID != command.RunID {
		t.Fatalf("unexpected outbox payload: %+v", payload)
	}
	if payload.MinimumRecommendedCapacity != OrchestrationCapacityHighV0 {
		t.Fatalf("minimum capacity=%q, want high", payload.MinimumRecommendedCapacity)
	}
}

func TestHandleRequestCapacityCommandV0PermiteTaskRefDeAPIHTTP(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	payload := validRequestCapacityPayloadV0("capacity-request-http-api")
	payload.TaskRef = "task-programacion-http-api"
	command, err := NewRequestCapacityCommandV0(
		validCommandMetaV0("cmd-capacity-http-api", "idem-capacity-http-api"),
		payload,
	)
	if err != nil {
		t.Fatalf("NewRequestCapacityCommandV0: %v", err)
	}

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RequestCapacity con task http: %v", err)
	}
	if len(result.Outbox) != 1 {
		t.Fatalf("outbox=%d, want 1", len(result.Outbox))
	}
}

func TestApplyCapacityRequestedV0ProjectsRefOnce(t *testing.T) {
	run := mustReducerProgramacionRunV0(t)
	event := mustCapacityRequestedEventV0(t, "evt-capacity-reducer-001", run.LastSequence+1, "capacity-request-001")

	got, err := ApplyEventV0(run, event)
	if err != nil {
		t.Fatalf("apply CapacityRequested: %v", err)
	}
	if !reflect.DeepEqual(got.CapacityRequests, []string{"capacity-request-001"}) {
		t.Fatalf("capacity_requests=%v, want [capacity-request-001]", got.CapacityRequests)
	}
	again := mustApplyReducerEventV0(t, got, event)
	if !reflect.DeepEqual(again.CapacityRequests, got.CapacityRequests) {
		t.Fatalf("capacity requests duplicated: %v", again.CapacityRequests)
	}
}

func TestApplyCapacityRequestedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustReducerProgramacionRunV0(t)
	event := mustCapacityRequestedEventV0(t, "evt-capacity-effect-first", run.LastSequence+1, "capacity-request-effect")
	applied := mustApplyReducerEventV0(t, run, event)
	conflict := mustCapacityRequestedEventWithKeyV0(t, "evt-capacity-effect-second", applied.LastSequence+1, "idem-capacity-effect-second", "capacity-request-effect")

	_, err := ApplyEventV0(applied, conflict)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrEventoConflictivoV0 || publicErr.Field != "idempotency" {
		t.Fatalf("error=%+v, want evento_conflictivo idempotency", publicErr)
	}
}

func TestReplayDurableEventsV0AcceptsCapacityRequested(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-capacity", 1, "idem-start-capacity"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-capacity", 2, "idem-phase-capacity", OrchestrationPhaseProgramacionV0),
		mustCapacityRequestedEventWithKeyV0(t, "evt-durable-capacity", 3, "idem-capacity", "capacity-request-001"),
		mustCapacityRequestedEventWithKeyV0(t, "evt-durable-capacity", 3, "idem-capacity", "capacity-request-001"),
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable CapacityRequested: %v", err)
	}
	if got.LastSequence != 3 {
		t.Fatalf("last_sequence=%d, want 3", got.LastSequence)
	}
	if !reflect.DeepEqual(got.CapacityRequests, []string{"capacity-request-001"}) {
		t.Fatalf("capacity_requests=%v, want [capacity-request-001]", got.CapacityRequests)
	}
}

func TestHandleRequestCapacityCommandV0RetriesPendingOutboxAfterPartialPersist(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRequestCapacityCommandV0(t, "cmd-capacity-repeat", "idem-capacity-repeat", "capacity-request-001")
	created := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(created, command)
	if err != nil {
		t.Fatalf("retry RequestCapacity after partial persist: %v", err)
	}
	if len(result.Events) != 0 || len(result.Outbox) != 1 {
		t.Fatalf("result=%+v, want no events and one outbox", result)
	}
	if result.Outbox[0].MessageType != OutboxMessageRequestCapacityDecisionV0 ||
		result.Outbox[0].TargetPort != OutboxTargetCapacityV0 {
		t.Fatalf("unexpected outbox retry: %+v", result.Outbox[0])
	}
}

func TestHandleRequestCapacityCommandV0RepeatedAfterDecisionDoesNotDuplicate(t *testing.T) {
	command := mustRequestCapacityCommandV0(t, "cmd-capacity-repeat-decided", "idem-capacity-repeat-decided", "capacity-request-repeat-decided")
	run := mustHandlerProgramacionRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, command)
	run = mustApplySingleCommandEventV0(t, run, mustRegisterCapacityDecisionCommandV0(t, "cmd-capacity-decision-repeat-decided", "idem-capacity-decision-repeat-decided", "capacity-request-repeat-decided"))

	result, err := HandleCommandV0(run, command)
	assertIdempotentNoEventsV0(t, result, err)
}

func TestHandleRequestCapacityCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRequestCapacityCommandV0(t, "cmd-capacity-conflict-effect", "idem-capacity-conflict-effect", "capacity-request-effect-conflict")
	created := mustApplySingleCommandEventV0(t, run, command)
	payload := validRequestCapacityPayloadV0("capacity-request-effect-conflict")
	payload.MinimumRecommendedCapacity = OrchestrationCapacityLowV0
	conflict, err := NewRequestCapacityCommandV0(validCommandMetaV0("cmd-capacity-conflict-effect", "idem-capacity-conflict-effect"), payload)
	conflict = mustCommandV0(t, conflict, err)

	_, err = HandleCommandV0(created, conflict)
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestHandleRequestCapacityCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRequestCapacityCommandV0(t, "cmd-capacity-conflict-key", "idem-capacity-conflict-key", "capacity-request-key-conflict")
	created := mustApplySingleCommandEventV0(t, run, command)
	conflict := mustRequestCapacityCommandV0(t, "cmd-capacity-conflict-key-2", "idem-capacity-conflict-key-2", "capacity-request-key-conflict")

	_, err := HandleCommandV0(created, conflict)
	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestHandleRequestCapacityCommandV0RejectsNonCurrentPhase(t *testing.T) {
	run := mustHandlerStartedRunV0(t)
	command := mustRequestCapacityCommandV0(t, "cmd-capacity-phase", "idem-capacity-phase", "capacity-request-phase")

	_, err := HandleCommandV0(run, command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrTransicionInvalidaV0 || publicErr.Field != "payload.phase_id" {
		t.Fatalf("error=%+v, want transicion_invalida payload.phase_id", publicErr)
	}
}

func TestRequestCapacityCommandV0PermiteDetallesOperativosOpacos(t *testing.T) {
	cases := map[string]func(*RequestCapacityCommandPayloadV0){
		"provider": func(payload *RequestCapacityCommandPayloadV0) { payload.Summary = "decidir provider externo" },
		"modelo":   func(payload *RequestCapacityCommandPayloadV0) { payload.ReasonCode = "modelo_requerido" },
		"runtime":  func(payload *RequestCapacityCommandPayloadV0) { payload.TaskRef = "runtime-task" },
		"HOME":     func(payload *RequestCapacityCommandPayloadV0) { payload.EvidenceRefs = []string{"$HOME/estado"} },
		"Codex":    func(payload *RequestCapacityCommandPayloadV0) { payload.Summary = "usar Codex" },
		"Claude":   func(payload *RequestCapacityCommandPayloadV0) { payload.Summary = "usar Claude" },
		"Ollama":   func(payload *RequestCapacityCommandPayloadV0) { payload.Summary = "usar Ollama" },
		"vLLM":     func(payload *RequestCapacityCommandPayloadV0) { payload.Summary = "usar vLLM" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			payload := validRequestCapacityPayloadV0("capacity-request-forbidden")
			mutate(&payload)

			if _, err := NewRequestCapacityCommandV0(validCommandMetaV0("cmd-capacity-opaque-"+strings.ToLower(name), "idem-capacity-opaque-"+strings.ToLower(name)), payload); err != nil {
				t.Fatalf("detalle operativo opaco rechazado: %v", err)
			}
		})
	}
}

func TestRequestCapacityCommandV0RejectsSensitiveDetails(t *testing.T) {
	payload := validRequestCapacityPayloadV0("capacity-request-sensitive")
	payload.EvidenceRefs = []string{"client_secret=abc123"}

	_, err := NewRequestCapacityCommandV0(validCommandMetaV0("cmd-capacity-sensitive", "idem-capacity-sensitive"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestCapacityRequestedEventV0RejectsForbiddenDetails(t *testing.T) {
	payload := capacityRequestedPayloadFromCommandV0(validRequestCapacityPayloadV0("capacity-request-event-forbidden"))
	payload.Summary = "authorization: Bearer abc123"

	_, err := NewCapacityRequestedEventV0(reducerEventMetaV0("evt-capacity-forbidden", 2), payload)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrDetalleProhibidoV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, ErrDetalleProhibidoV0)
	}
}

func TestCapacityRequestedOutboxPayloadDoesNotContainProviderModelRuntimeHome(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRequestCapacityCommandV0(t, "cmd-capacity-clean", "idem-capacity-clean", "capacity-request-clean")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RequestCapacity: %v", err)
	}
	serialized := strings.ToLower(string(result.Outbox[0].Payload))
	for _, forbidden := range operationalSensitiveFragmentsForTestV0() {
		if containsForbiddenFragmentV0(serialized, forbidden) {
			t.Fatalf("outbox payload contains forbidden fragment %q: %s", forbidden, serialized)
		}
	}
}

func TestValidateRequestCapacityDecisionOutboxRejectsInvalidPayload(t *testing.T) {
	message := OutboxMessageV0{
		MessageID:      "outbox-capacity-invalid",
		MessageType:    OutboxMessageRequestCapacityDecisionV0,
		RunID:          "run-001",
		IdempotencyKey: "idem-capacity-invalid",
		TargetPort:     OutboxTargetCapacityV0,
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

func TestHandleRequestCapacityCommandV0DoesNotBreakStartRun(t *testing.T) {
	command := mustStartRunCommandV0(t, "cmd-start-after-capacity", "idem-start-after-capacity")

	result, err := HandleCommandV0(OrchestrationRunV0{}, command)
	if err != nil {
		t.Fatalf("handle StartRun after adding capacity command: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventRunStartedV0)
}

func mustRequestCapacityCommandV0(t *testing.T, commandID string, idempotencyKey string, requestID string) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRequestCapacityCommandV0(validCommandMetaV0(commandID, idempotencyKey), validRequestCapacityPayloadV0(requestID))
	return mustCommandV0(t, command, err)
}

func mustCapacityRequestedEventV0(t *testing.T, eventID string, sequence int64, requestID string) OrchestrationEventV0 {
	t.Helper()
	return mustCapacityRequestedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, requestID)
}

func mustCapacityRequestedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, requestID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewCapacityRequestedEventV0(meta, capacityRequestedPayloadFromCommandV0(validRequestCapacityPayloadV0(requestID)))
	return mustReducerEventV0(t, event, err)
}

func validRequestCapacityPayloadV0(requestID string) RequestCapacityCommandPayloadV0 {
	return RequestCapacityCommandPayloadV0{
		CapacityRequestID:          requestID,
		PhaseID:                    string(OrchestrationPhaseProgramacionV0),
		TaskRef:                    "task-ncw-010",
		ReasonCode:                 "complejidad_alta",
		Summary:                    "La fase requiere mas capacidad de razonamiento para reducir riesgo.",
		MinimumRecommendedCapacity: OrchestrationCapacityHighV0,
		EvidenceRefs:               []string{"docs/contratos_capacidad.md#RequestCapacity"},
	}
}
