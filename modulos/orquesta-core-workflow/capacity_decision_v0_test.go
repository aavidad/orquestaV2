package orquestacoreworkflow

import (
	"errors"
	"reflect"
	"testing"
)

const defaultAgentCapacityRequestIDV0 = "capacity-request-010"

func TestHandleRegisterCapacityDecisionCommandV0ProjectsDecision(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustRequestCapacityCommandV0(t, "cmd-capacity-before-decision", "idem-capacity-before-decision", "capacity-request-001"))
	command := mustRegisterCapacityDecisionCommandV0(t, "cmd-capacity-decision-001", "idem-capacity-decision-001", "capacity-request-001")

	result, err := HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("handle RegisterCapacityDecision: %v", err)
	}
	assertSingleEventTypeV0(t, result, OrchestrationEventCapacityDecidedV0)
	if len(result.Outbox) != 0 {
		t.Fatalf("outbox=%d, want empty", len(result.Outbox))
	}
	decided := mustApplyReducerEventV0(t, run, result.Events[0])
	if !capacityDecisionAlreadyReflectedV0(decided, "capacity-request-001") {
		t.Fatalf("capacity decision not projected: %v", decided.CapacityDecisions)
	}
}

func TestApplyCapacityDecidedV0RequiresExistingRequest(t *testing.T) {
	run := mustReducerProgramacionRunV0(t)
	event := mustCapacityDecidedEventV0(t, "evt-capacity-decision-missing", run.LastSequence+1, "capacity-request-missing")

	_, err := ApplyEventV0(run, event)
	assertCapacityDecisionEventErrorV0(t, err, ErrSecuenciaInvalidaV0, "payload.capacity_request_id")
}

func TestReplayDurableEventsV0AcceptsCapacityDecidedAndExactDuplicate(t *testing.T) {
	events := []OrchestrationEventV0{
		mustReplayRunStartedEventWithKeyV0(t, "evt-durable-start-capacity-decision", 1, "idem-start-capacity-decision"),
		mustReplayPhaseEventWithKeyV0(t, "evt-durable-phase-capacity-decision", 2, "idem-phase-capacity-decision", OrchestrationPhaseProgramacionV0),
		mustCapacityRequestedEventWithKeyV0(t, "evt-durable-capacity-before-decision", 3, "idem-capacity-before-decision", "capacity-request-001"),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision", 4, "idem-capacity-decision", "capacity-request-001"),
		mustCapacityDecidedEventWithKeyV0(t, "evt-durable-capacity-decision", 4, "idem-capacity-decision", "capacity-request-001"),
	}

	got, err := ReplayDurableEventsV0(events)
	if err != nil {
		t.Fatalf("replay durable CapacityDecided: %v", err)
	}
	if got.LastSequence != 4 {
		t.Fatalf("last_sequence=%d, want 4", got.LastSequence)
	}
	if !capacityDecisionAlreadyReflectedV0(got, "capacity-request-001") {
		t.Fatalf("capacity_decisions=%v", got.CapacityDecisions)
	}
}

func TestRegisterCapacityDecisionCommandV0RepeatedDoesNotDuplicate(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustRequestCapacityCommandV0(t, "cmd-capacity-repeat-before-decision", "idem-capacity-repeat-before-decision", "capacity-request-repeat"))
	command := mustRegisterCapacityDecisionCommandV0(t, "cmd-capacity-decision-repeat", "idem-capacity-decision-repeat", "capacity-request-repeat")
	decided := mustApplySingleCommandEventV0(t, run, command)

	result, err := HandleCommandV0(decided, command)
	assertIdempotentNoEventsV0(t, result, err)
	if !reflect.DeepEqual(decided.CapacityDecisions, []string{capacityDecisionProjectionRefV0(validRegisterCapacityDecisionPayloadV0("capacity-request-repeat"))}) {
		t.Fatalf("capacity_decisions=%v", decided.CapacityDecisions)
	}
}

func TestRegisterCapacityDecisionCommandV0RejectsReflectedPayloadConflict(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustRequestCapacityCommandV0(t, "cmd-capacity-effect-before-decision", "idem-capacity-effect-before-decision", "capacity-request-effect"))
	payload := validRegisterCapacityDecisionPayloadV0("capacity-request-effect")
	command := mustRegisterCapacityDecisionCommandWithPayloadV0(t, "cmd-capacity-decision-effect", "idem-capacity-decision-effect", payload)
	decided := mustApplySingleCommandEventV0(t, run, command)

	payload.Summary = "Decision compacta de capacidad con otro resumen."
	conflicting := mustRegisterCapacityDecisionCommandWithPayloadV0(t, "cmd-capacity-decision-effect", "idem-capacity-decision-effect", payload)
	_, err := HandleCommandV0(decided, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload")
}

func TestRegisterCapacityDecisionCommandV0RejectsReflectedIdempotencyConflict(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustRequestCapacityCommandV0(t, "cmd-capacity-key-before-decision", "idem-capacity-key-before-decision", "capacity-request-key"))
	command := mustRegisterCapacityDecisionCommandV0(t, "cmd-capacity-decision-key", "idem-capacity-decision-key", "capacity-request-key")
	decided := mustApplySingleCommandEventV0(t, run, command)

	conflicting := mustRegisterCapacityDecisionCommandV0(t, "cmd-capacity-decision-key-2", "idem-capacity-decision-key-2", "capacity-request-key")
	_, err := HandleCommandV0(decided, conflicting)

	assertCommandErrorV0(t, err, ErrTransicionInvalidaV0, "idempotency_key")
}

func TestApplyCapacityDecidedV0RejectsReflectedEffectConflict(t *testing.T) {
	run := mustReducerProgramacionRunV0(t)
	run = mustApplyReducerEventV0(t, run, mustCapacityRequestedEventV0(t, "evt-capacity-effect-before-decision", run.LastSequence+1, "capacity-request-effect"))
	first := mustCapacityDecidedEventWithKeyV0(t, "evt-capacity-decision-effect", run.LastSequence+1, "idem-capacity-decision-effect", "capacity-request-effect")
	applied := mustApplyReducerEventV0(t, run, first)
	conflicting := mustCapacityDecidedEventWithKeyV0(t, "evt-capacity-decision-effect-conflict", applied.LastSequence+1, "idem-capacity-decision-effect-conflict", "capacity-request-effect")

	_, err := ApplyEventV0(applied, conflicting)

	assertEventErrorV0(t, err, ErrEventoConflictivoV0, "idempotency")
}

func TestRegisterCapacityDecisionCommandV0RejectsConflict(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	run = mustApplySingleCommandEventV0(t, run, mustRequestCapacityCommandV0(t, "cmd-capacity-conflict-before-decision", "idem-capacity-conflict-before-decision", "capacity-request-conflict"))
	decided := mustApplySingleCommandEventV0(t, run, mustRegisterCapacityDecisionCommandV0(t, "cmd-capacity-decision-conflict", "idem-capacity-decision-conflict", "capacity-request-conflict"))
	payload := validRegisterCapacityDecisionPayloadV0("capacity-request-conflict")
	payload.DecisionRef = "capacity-decision-conflicting"
	conflict, err := NewRegisterCapacityDecisionCommandV0(validCommandMetaV0("cmd-capacity-decision-conflicting", "idem-capacity-decision-conflicting"), payload)
	conflict = mustCommandV0(t, conflict, err)

	_, err = HandleCommandV0(decided, conflict)
	assertCapacityDecisionCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.decision_ref")
}

func TestRegisterCapacityDecisionCommandV0RejectsMissingRequest(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRegisterCapacityDecisionCommandV0(t, "cmd-capacity-decision-missing", "idem-capacity-decision-missing", "capacity-request-missing")

	_, err := HandleCommandV0(run, command)
	assertCapacityDecisionCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.capacity_request_id")
}

func TestRegisterCapacityDecisionCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRegisterCapacityDecisionPayloadV0("capacity-request-forbidden")
	payload.Summary = "client_secret=abc123"

	_, err := NewRegisterCapacityDecisionCommandV0(validCommandMetaV0("cmd-capacity-decision-forbidden", "idem-capacity-decision-forbidden"), payload)
	assertCapacityDecisionCommandErrorV0(t, err, ErrDetalleProhibidoV0, "payload")
}

func mustRunWithCapacityDecisionV0(t *testing.T, requestID string) OrchestrationRunV0 {
	t.Helper()
	return mustApplyCapacityDecisionToRunV0(t, mustHandlerProgramacionRunV0(t), requestID)
}

func mustApplyCapacityDecisionToRunV0(t *testing.T, run OrchestrationRunV0, requestID string) OrchestrationRunV0 {
	t.Helper()
	run = mustApplySingleCommandEventV0(t, run, mustRequestCapacityCommandV0(t, "cmd-capacity-"+requestID, "idem-capacity-"+requestID, requestID))
	return mustApplySingleCommandEventV0(t, run, mustRegisterCapacityDecisionCommandV0(t, "cmd-capacity-decision-"+requestID, "idem-capacity-decision-"+requestID, requestID))
}

func mustReducerRunWithCapacityDecisionV0(t *testing.T, requestID string) OrchestrationRunV0 {
	t.Helper()
	run := mustReducerProgramacionRunV0(t)
	run = mustApplyReducerEventV0(t, run, mustCapacityRequestedEventV0(t, "evt-capacity-"+requestID, run.LastSequence+1, requestID))
	return mustApplyReducerEventV0(t, run, mustCapacityDecidedEventV0(t, "evt-capacity-decision-"+requestID, run.LastSequence+1, requestID))
}

func mustRegisterCapacityDecisionCommandV0(t *testing.T, commandID string, idempotencyKey string, requestID string) OrchestrationCommandV0 {
	t.Helper()
	return mustRegisterCapacityDecisionCommandWithPayloadV0(t, commandID, idempotencyKey, validRegisterCapacityDecisionPayloadV0(requestID))
}

func mustRegisterCapacityDecisionCommandWithPayloadV0(t *testing.T, commandID string, idempotencyKey string, payload RegisterCapacityDecisionCommandPayloadV0) OrchestrationCommandV0 {
	t.Helper()
	command, err := NewRegisterCapacityDecisionCommandV0(validCommandMetaV0(commandID, idempotencyKey), payload)
	return mustCommandV0(t, command, err)
}

func mustCapacityDecidedEventV0(t *testing.T, eventID string, sequence int64, requestID string) OrchestrationEventV0 {
	t.Helper()
	return mustCapacityDecidedEventWithKeyV0(t, eventID, sequence, "idem-"+eventID, requestID)
}

func mustCapacityDecidedEventWithKeyV0(t *testing.T, eventID string, sequence int64, idempotencyKey string, requestID string) OrchestrationEventV0 {
	t.Helper()
	meta := reducerEventMetaV0(eventID, sequence)
	meta.IdempotencyKey = idempotencyKey
	event, err := NewCapacityDecidedEventV0(meta, capacityDecidedPayloadFromCommandV0(validRegisterCapacityDecisionPayloadV0(requestID)))
	return mustReducerEventV0(t, event, err)
}

func validRegisterCapacityDecisionPayloadV0(requestID string) RegisterCapacityDecisionCommandPayloadV0 {
	return RegisterCapacityDecisionCommandPayloadV0{
		CapacityRequestID: requestID,
		DecisionRef:       "capacity-decision-" + requestID,
		Tier:              OrchestrationCapacityHighV0,
		ReasoningEffort:   OrchestrationCapacityHighV0,
		Summary:           "Decision compacta de capacidad para la solicitud.",
		EvidenceRefs:      []string{"docs/contratos_capacidad.md#CapacityDecided"},
	}
}

func assertCapacityDecisionCommandErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want %s %s", publicErr, code, field)
	}
}

func assertCapacityDecisionEventErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want %s %s", publicErr, code, field)
	}
}
