package orquestacoreworkflow

import (
	"errors"
	"testing"
)

func TestAgentStopRequestProjectionV0ParseaMotivoCompacto(t *testing.T) {
	payload := AgentStopRequestedPayloadV0{
		AgentRequestID: "agent-request-projection-001",
		ReasonCode:     "loop_detected",
		Summary:        "Detener por bucle detectado.",
	}

	ref := AgentStopRequestProjectionRefV0(payload)
	projection, ok := ParseAgentStopRequestProjectionV0(ref)

	if !ok ||
		ref != "agent-request-projection-001#reason:loop_detected" ||
		projection.AgentRequestID != payload.AgentRequestID ||
		projection.ReasonCode != payload.ReasonCode {
		t.Fatalf("projection=%+v ok=%v ref=%q", projection, ok, ref)
	}
}

func TestStopAgentCommandV0RejectsUnsafeProjectionSeparator(t *testing.T) {
	payload := validStopAgentPayloadV0("agent-request-projection-unsafe")
	payload.ReasonCode = "loop#detected"

	_, err := NewStopAgentCommandV0(validCommandMetaV0("cmd-stop-projection-unsafe", "idem-stop-projection-unsafe"), payload)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 || publicErr.Field != "payload" {
		t.Fatalf("error=%+v, want payload_invalido payload", publicErr)
	}
}
