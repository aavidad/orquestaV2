package orquestaagentprocessregistrymemory

import (
	"context"
	"errors"
	"testing"

	orquestaagentprocessregistry "orquesta/modulos/orquesta-agent-process-registry"
)

func TestInMemoryAgentProcessRegistryV0RecordsAndResolves(t *testing.T) {
	registry := NewInMemoryAgentProcessRegistryV0()
	record := validAgentProcessRecordV0()
	record.RunID = "  run-process-registry-001  "
	record.AgentRequestID = "  agent-process-registry-001  "
	record.EvidenceRefs = []string{" evidence-ref-registry-001 ", "evidence-ref-registry-001"}

	if err := registry.RecordAgentProcessV0(context.Background(), record); err != nil {
		t.Fatalf("record agent process: %v", err)
	}
	got, err := registry.ResolveAgentProcessV0(
		context.Background(),
		"run-process-registry-001",
		"agent-process-registry-001",
	)
	if err != nil {
		t.Fatalf("resolve agent process: %v", err)
	}

	if got.RunID != "run-process-registry-001" ||
		got.AgentRequestID != "agent-process-registry-001" ||
		got.ProcessRef != record.ProcessRef ||
		got.SessionRef != record.SessionRef ||
		got.LaunchRef != record.LaunchRef ||
		got.ReadinessRef != record.ReadinessRef ||
		len(got.EvidenceRefs) != 1 ||
		got.EvidenceRefs[0] != "evidence-ref-registry-001" {
		t.Fatalf("record=%+v", got)
	}
}

func TestInMemoryAgentProcessRegistryV0RejectsForbiddenDetail(t *testing.T) {
	registry := NewInMemoryAgentProcessRegistryV0()
	record := validAgentProcessRecordV0()
	record.ProcessRef = "process://raw-detail"

	err := registry.RecordAgentProcessV0(context.Background(), record)

	assertAgentProcessRegistryErrorV0(t, err, orquestaagentprocessregistry.ErrAgentProcessRegistryInvalidV0, "agent_process.process_ref")
}

func TestInMemoryAgentProcessRegistryV0ReplayIdempotente(t *testing.T) {
	registry := NewInMemoryAgentProcessRegistryV0()
	record := validAgentProcessRecordV0()

	if err := registry.RecordAgentProcessV0(context.Background(), record); err != nil {
		t.Fatalf("record agent process: %v", err)
	}
	if err := registry.RecordAgentProcessV0(context.Background(), record); err != nil {
		t.Fatalf("replay idempotente: %v", err)
	}
}

func TestInMemoryAgentProcessRegistryV0ConflictoNoSobrescribe(t *testing.T) {
	registry := NewInMemoryAgentProcessRegistryV0()
	record := validAgentProcessRecordV0()
	conflict := record
	conflict.ProcessRef = "process-ref-registry-999"

	if err := registry.RecordAgentProcessV0(context.Background(), record); err != nil {
		t.Fatalf("record agent process: %v", err)
	}
	err := registry.RecordAgentProcessV0(context.Background(), conflict)

	assertAgentProcessRegistryErrorV0(t, err, orquestaagentprocessregistry.ErrAgentProcessRegistryInvalidV0, "agent_process_registry")
	got, err := registry.ResolveAgentProcessV0(
		context.Background(),
		record.RunID,
		record.AgentRequestID,
	)
	if err != nil {
		t.Fatalf("resolve agent process: %v", err)
	}
	if got.ProcessRef != record.ProcessRef {
		t.Fatalf("process_ref sobrescrito: got=%q want=%q", got.ProcessRef, record.ProcessRef)
	}
}

func TestInMemoryAgentProcessRegistryV0RejectsInvalidLookupRefs(t *testing.T) {
	registry := NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), validAgentProcessRecordV0()); err != nil {
		t.Fatalf("record agent process: %v", err)
	}

	_, err := registry.ResolveAgentProcessV0(context.Background(), "run-process-registry-001", "agent/request/raw")

	assertAgentProcessRegistryErrorV0(t, err, orquestaagentprocessregistry.ErrAgentProcessRegistryInvalidV0, "agent_process.agent_request_id")
}

func TestInMemoryAgentProcessRegistryV0ReturnsStoreErrorWhenMissing(t *testing.T) {
	registry := NewInMemoryAgentProcessRegistryV0()

	_, err := registry.ResolveAgentProcessV0(
		context.Background(),
		"run-process-registry-001",
		"agent-process-registry-001",
	)

	assertAgentProcessRegistryErrorV0(t, err, orquestaagentprocessregistry.ErrAgentProcessRegistryNotFoundV0, "agent_process_registry")
}

func TestInMemoryAgentProcessRegistryV0HonorsContextCancellation(t *testing.T) {
	registry := NewInMemoryAgentProcessRegistryV0()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := registry.RecordAgentProcessV0(ctx, validAgentProcessRecordV0())

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v, want context.Canceled", err)
	}
}

func validAgentProcessRecordV0() AgentProcessRecordV0 {
	return AgentProcessRecordV0{
		RunID:          "run-process-registry-001",
		AgentRequestID: "agent-process-registry-001",
		ProcessRef:     "process-ref-registry-001",
		SessionRef:     "session-ref-registry-001",
		LaunchRef:      "launch-ref-registry-001",
		ReadinessRef:   "readiness-ref-registry-001",
	}
}

func assertAgentProcessRegistryErrorV0(
	t *testing.T,
	err error,
	code string,
	field string,
) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var publicErr orquestaagentprocessregistry.ErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("unexpected error type %T: %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}
