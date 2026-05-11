package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestRecordConcurrencyGateCommandV0RejectsRunRefMismatch(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	payload := validConcurrencyGatePayloadV0("gate-ref-run-mismatch", ConcurrencyGateDecisionAllowRequestAgentV0)
	payload.RunRef = "run-other"
	command, err := NewRecordConcurrencyGateCommandV0(validCommandMetaV0("cmd-concurrency-run-mismatch", "idem-concurrency-run-mismatch"), payload)
	if err != nil {
		t.Fatalf("command constructor: %v", err)
	}

	_, err = HandleCommandV0(run, command)
	assertConcurrencyGateCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.run_ref")
}

func TestRecordConcurrencyGateCommandV0RejectsInvalidDecision(t *testing.T) {
	payload := validConcurrencyGatePayloadV0("gate-ref-invalid-decision", ConcurrencyGateDecisionAllowRequestAgentV0)
	payload.Decision = ConcurrencyGateDecisionV0("launch_anyway")

	_, err := NewRecordConcurrencyGateCommandV0(validCommandMetaV0("cmd-concurrency-invalid-decision", "idem-concurrency-invalid-decision"), payload)
	assertConcurrencyGateCommandErrorV0(t, err, ErrPayloadInvalidoV0, "payload.decision")
}

func TestRecordConcurrencyGateCommandV0RejectsAllowWhenSubjectNotReady(t *testing.T) {
	payload := validConcurrencyGatePayloadV0("gate-ref-subject-not-ready", ConcurrencyGateDecisionAllowRequestAgentV0)
	payload.ReadyClaimRefs = []string{"claim-ref-other"}

	_, err := NewRecordConcurrencyGateCommandV0(validCommandMetaV0("cmd-concurrency-subject-not-ready", "idem-concurrency-subject-not-ready"), payload)
	assertConcurrencyGateCommandErrorV0(t, err, ErrPayloadInvalidoV0, "payload.subject_claim_refs")
}

func TestRecordConcurrencyGateCommandV0RejectsBlockWithoutBlockedSubject(t *testing.T) {
	payload := validConcurrencyGatePayloadV0("gate-ref-block-without-subject", ConcurrencyGateDecisionBlockRequestAgentV0)
	payload.BlockedClaimRefs = []string{"claim-ref-other"}

	_, err := NewRecordConcurrencyGateCommandV0(validCommandMetaV0("cmd-concurrency-block-without-subject", "idem-concurrency-block-without-subject"), payload)
	assertConcurrencyGateCommandErrorV0(t, err, ErrPayloadInvalidoV0, "payload.subject_claim_refs")
}

func TestRecordConcurrencyGateCommandV0RejectsConflictingGateRef(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRecordConcurrencyGateCommandV0(t, "cmd-concurrency-conflict", "idem-concurrency-conflict", validConcurrencyGatePayloadV0("gate-ref-conflict", ConcurrencyGateDecisionAllowRequestAgentV0))
	created := mustApplySingleCommandEventV0(t, run, command)
	conflict := validConcurrencyGatePayloadV0("gate-ref-conflict", ConcurrencyGateDecisionBlockRequestAgentV0)
	conflict.PlanRef = "plan-ref-gate-ref-conflict"
	conflictCommand, err := NewRecordConcurrencyGateCommandV0(validCommandMetaV0("cmd-concurrency-conflict-2", "idem-concurrency-conflict-2"), conflict)
	if err != nil {
		t.Fatalf("command constructor: %v", err)
	}

	_, err = HandleCommandV0(created, conflictCommand)
	assertConcurrencyGateCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.gate_ref")
}

func TestApplyConcurrencyGateRecordedEventV0RejectsConflictingGateRef(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	payload := validConcurrencyGateRecordedPayloadV0("gate-ref-event-conflict", ConcurrencyGateDecisionAllowRequestAgentV0)
	applied := mustApplyReducerEventV0(t, run, mustConcurrencyGateRecordedEventV0(t, "evt-concurrency-event-conflict", run.LastSequence+1, payload))
	conflict := validConcurrencyGateRecordedPayloadV0("gate-ref-event-conflict", ConcurrencyGateDecisionBlockRequestAgentV0)
	conflict.PlanRef = payload.PlanRef
	event := mustConcurrencyGateRecordedEventV0(t, "evt-concurrency-event-conflict-2", applied.LastSequence+1, conflict)

	_, err := ApplyEventV0(applied, event)
	assertConcurrencyGateEventErrorV0(t, err, ErrSecuenciaInvalidaV0, "payload.gate_ref")
}

func TestRecordConcurrencyGateCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validConcurrencyGatePayloadV0("gate-ref-forbidden", ConcurrencyGateDecisionAllowRequestAgentV0)
	payload.EvidenceRefs = []string{"runtime-detail"}

	_, err := NewRecordConcurrencyGateCommandV0(validCommandMetaV0("cmd-concurrency-forbidden", "idem-concurrency-forbidden"), payload)
	assertConcurrencyGateCommandErrorV0(t, err, ErrDetalleProhibidoV0, "payload")
}

func TestValidateRunV0RejectsInvalidConcurrencyGateProjection(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	run.ConcurrencyGates = []string{"invalid-concurrency-gate"}

	issues := ValidateOrchestrationRunV0(run)
	if len(issues) == 0 {
		t.Fatalf("expected run validation issue")
	}
	if issues[0].Code != OrchestrationEstadoInconsistenteV0 || issues[0].Field != "concurrency_gates" {
		t.Fatalf("issue=%+v, want estado_inconsistente concurrency_gates", issues[0])
	}
}

func assertConcurrencyGateCommandErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want %s %s", publicErr, code, field)
	}
}

func assertConcurrencyGateEventErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want %s %s", publicErr, code, field)
	}
}
