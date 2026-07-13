package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateOrchestrationCommandV0RejectsPayloadAboveBudgetV0(t *testing.T) {
	command := OrchestrationCommandV0{
		CommandID:      "cmd-payload-budget-001",
		CommandType:    OrchestrationCommandStartRunV0,
		RunID:          "run-payload-budget-001",
		IdempotencyKey: "idem-payload-budget-001",
		OccurredAt:     "2026-05-26T10:00:00Z",
		PayloadVersion: OrchestrationCommandPayloadVersionV0,
		Payload:        json.RawMessage(strings.Repeat("{", maxWorkflowCommandEventPayloadBytesV0+1)),
	}

	err := ValidateOrchestrationCommandV0(command)
	var publicErr OrchestrationCommandErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public command error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 || publicErr.Field != "payload" {
		t.Fatalf("error=%+v, want %s payload", publicErr, ErrPayloadInvalidoV0)
	}
}

func TestValidateOrchestrationEventV0RejectsPayloadAboveBudgetV0(t *testing.T) {
	event := OrchestrationEventV0{
		EventID:        "evt-payload-budget-001",
		EventType:      OrchestrationEventRunStartedV0,
		RunID:          "run-payload-budget-001",
		Sequence:       1,
		OccurredAt:     "2026-05-26T10:00:00Z",
		PayloadVersion: OrchestrationEventPayloadVersionV0,
		Payload:        json.RawMessage(strings.Repeat("{", maxWorkflowCommandEventPayloadBytesV0+1)),
	}

	err := ValidateOrchestrationEventV0(event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 || publicErr.Field != "payload" {
		t.Fatalf("error=%+v, want %s payload", publicErr, ErrPayloadInvalidoV0)
	}
}

func TestValidateOrchestrationEventPayloadBudgetV0RejectsPayloadAboveBudgetV0(t *testing.T) {
	event := OrchestrationEventV0{
		EventType: OrchestrationEventRunStartedV0,
		Payload:   json.RawMessage(strings.Repeat("{", maxWorkflowCommandEventPayloadBytesV0+1)),
	}

	err := ValidateOrchestrationEventPayloadBudgetV0(event)
	var publicErr OrchestrationEventErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected public event error, got %T %v", err, err)
	}
	if publicErr.Code != ErrPayloadInvalidoV0 || publicErr.Field != "payload" {
		t.Fatalf("error=%+v, want %s payload", publicErr, ErrPayloadInvalidoV0)
	}
}

func TestWorkflowPayloadBudgetUsesOutboxDefaultAndTypeOverridesV0(t *testing.T) {
	if maxWorkflowCommandEventPayloadBytesV0 != maxOutboxPayloadBytesV0 {
		t.Fatalf("workflow default budget=%d, want outbox budget=%d",
			maxWorkflowCommandEventPayloadBytesV0, maxOutboxPayloadBytesV0)
	}
	if workflowCommandPayloadMaxBytesV0(OrchestrationCommandStartRunV0) != maxOutboxPayloadBytesV0 {
		t.Fatalf("StartRun budget must use default workflow/outbox budget")
	}
	if workflowCommandPayloadMaxBytesV0(OrchestrationCommandCreateMicrotaskV0) !=
		maxCreateMicrotaskCommandPayloadBytesV0 {
		t.Fatalf("CreateMicrotask budget must preserve task wrapper override")
	}
	if workflowEventPayloadMaxBytesV0(OrchestrationEventMicrotaskCreatedV0) !=
		maxMicrotaskCreatedPayloadBytesV0 {
		t.Fatalf("MicrotaskCreated budget must preserve compact event override")
	}
}
