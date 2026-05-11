package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestRegisterAgentLeaseExpiredCommandV0RejectsMissingAgent(t *testing.T) {
	run := mustHandlerProgramacionRunV0(t)
	command := mustRegisterAgentLeaseExpiredCommandV0(t, "cmd-lease-missing-agent", "idem-lease-missing-agent", "agent-request-missing", "lease-ref-missing-agent")

	_, err := HandleCommandV0(run, command)
	assertAgentLeaseExpiredCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.agent_request_id")
}

func TestRegisterAgentLeaseExpiredCommandV0RejectsRunRefMismatch(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-run-mismatch")
	payload := validRegisterAgentLeaseExpiredPayloadV0("agent-request-run-mismatch", "lease-ref-run-mismatch")
	payload.RunRef = "run-other"
	command, err := NewRegisterAgentLeaseExpiredCommandV0(validCommandMetaV0("cmd-lease-run-mismatch", "idem-lease-run-mismatch"), payload)
	if err != nil {
		t.Fatalf("command constructor: %v", err)
	}

	_, err = HandleCommandV0(run, command)
	assertAgentLeaseExpiredCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.run_ref")
}

func TestRegisterAgentLeaseExpiredCommandV0RejectsInvalidAction(t *testing.T) {
	payload := validRegisterAgentLeaseExpiredPayloadV0("agent-request-invalid-action", "lease-ref-invalid-action")
	payload.RecommendedAction = AgentLeaseRecommendedActionV0("continue")

	_, err := NewRegisterAgentLeaseExpiredCommandV0(validCommandMetaV0("cmd-lease-invalid-action", "idem-lease-invalid-action"), payload)
	assertAgentLeaseExpiredCommandErrorV0(t, err, ErrPayloadInvalidoV0, "payload.recommended_action")
}

func TestRegisterAgentLeaseExpiredCommandV0RejectsInvalidObservedAt(t *testing.T) {
	payload := validRegisterAgentLeaseExpiredPayloadV0("agent-request-invalid-time", "lease-ref-invalid-time")
	payload.ObservedAt = "2026-05-04T10:00:00+02:00"

	_, err := NewRegisterAgentLeaseExpiredCommandV0(validCommandMetaV0("cmd-lease-invalid-time", "idem-lease-invalid-time"), payload)
	assertAgentLeaseExpiredCommandErrorV0(t, err, ErrPayloadInvalidoV0, "payload.observed_at")
}

func TestRegisterAgentLeaseExpiredCommandV0RejectsConflictingLeaseRef(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-conflict")
	command := mustRegisterAgentLeaseExpiredCommandV0(t, "cmd-lease-conflict", "idem-lease-conflict", "agent-request-conflict", "lease-ref-conflict")
	created := mustApplySingleCommandEventV0(t, run, command)
	conflict := validRegisterAgentLeaseExpiredPayloadV0("agent-request-conflict", "lease-ref-conflict")
	conflict.RecommendedAction = AgentLeaseActionAskDirectorV0
	conflictCommand, err := NewRegisterAgentLeaseExpiredCommandV0(validCommandMetaV0("cmd-lease-conflict-2", "idem-lease-conflict-2"), conflict)
	if err != nil {
		t.Fatalf("command constructor: %v", err)
	}

	_, err = HandleCommandV0(created, conflictCommand)
	assertAgentLeaseExpiredCommandErrorV0(t, err, ErrTransicionInvalidaV0, "payload.lease_ref")
}

func TestApplyAgentLeaseExpiredEventV0RejectsConflictingLeaseRef(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-event-conflict")
	payload := validAgentLeaseExpiredPayloadV0("agent-request-event-conflict", "lease-ref-event-conflict")
	applied := mustApplyReducerEventV0(t, run, mustAgentLeaseExpiredEventV0(t, "evt-lease-event-conflict", run.LastSequence+1, payload))
	conflict := payload
	conflict.RecommendedAction = AgentLeaseActionAskDirectorV0
	event := mustAgentLeaseExpiredEventV0(t, "evt-lease-event-conflict-2", applied.LastSequence+1, conflict)

	_, err := ApplyEventV0(applied, event)
	assertAgentLeaseExpiredEventErrorV0(t, err, ErrSecuenciaInvalidaV0, "payload.lease_ref")
}

func TestRegisterAgentLeaseExpiredCommandV0RejectsForbiddenDetails(t *testing.T) {
	payload := validRegisterAgentLeaseExpiredPayloadV0("agent-request-forbidden", "lease-ref-forbidden")
	payload.EvidenceRefs = []string{"provider-detail"}

	_, err := NewRegisterAgentLeaseExpiredCommandV0(validCommandMetaV0("cmd-lease-forbidden", "idem-lease-forbidden"), payload)
	assertAgentLeaseExpiredCommandErrorV0(t, err, ErrDetalleProhibidoV0, "payload")
}

func TestValidateRunV0RejectsDanglingAgentLeaseExpiration(t *testing.T) {
	run := mustRunWithRequestedAgentV0(t, "agent-request-valid-lease")
	run.AgentLeaseExpirations = []string{agentLeaseExpiredProjectionRefV0(validAgentLeaseExpiredPayloadV0("agent-request-missing", "lease-ref-dangling"))}

	issues := ValidateOrchestrationRunV0(run)
	if len(issues) == 0 {
		t.Fatalf("expected run validation issue")
	}
	if issues[0].Code != OrchestrationEstadoInconsistenteV0 || issues[0].Field != "agent_lease_expirations" {
		t.Fatalf("issue=%+v, want estado_inconsistente agent_lease_expirations", issues[0])
	}
}

func assertAgentLeaseExpiredCommandErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want %s %s", publicErr, code, field)
	}
}

func assertAgentLeaseExpiredEventErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want %s %s", publicErr, code, field)
	}
}
