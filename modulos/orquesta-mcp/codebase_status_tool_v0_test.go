package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestMCPCodebaseStatusTransportV0RegistradoYDelegado(t *testing.T) {
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	leases := codebaseStatusLeaseStoreTestV0(t, started)
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		CodebaseStatus: MCPCodebaseStatusToolExecutorV0{
			Leases: leases,
			Clock:  func() time.Time { return started.Add(time.Minute) },
		},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	raw, err := transport.CallToolV0(context.Background(), MCPCodebaseStatusToolNameV0, validMCPCodebaseStatusInputTestV0(started))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	var result MCPCodebaseStatusToolResultV0
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, raw)
	}
	if result.Estado != orquestacontext.CodeContextToolingEstadoAttentionRequiredV0 ||
		result.StopRequested != 1 ||
		len(result.Entries) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Entries[0].ReasonCode != orquestacontext.CodeContextToolLeaseReasonLeaseExpiredV0 {
		t.Fatalf("entries=%+v", result.Entries)
	}
}

func TestMCPCodebaseStatusInputSchemaV0ExponeCampos(t *testing.T) {
	fields, ok := MCPTransportToolInputFieldsV0(MCPCodebaseStatusToolNameV0)
	if !ok {
		t.Fatalf("schema no encontrado")
	}
	seen := map[string]bool{}
	for _, field := range fields {
		seen[field.Name] = true
	}
	for _, want := range []string{"request_ref", "repository_ref", "tool_ref", "observed_at", "include_terminal", "default_cpu_high_percent", "observations"} {
		if !seen[want] {
			t.Fatalf("campo %q no encontrado en %+v", want, fields)
		}
	}
}

func TestMCPCodebaseStatusToolExecutorV0NoConfigurado(t *testing.T) {
	result, err := (MCPCodebaseStatusToolExecutorV0{}).Execute(context.Background(), validMCPCodebaseStatusInputTestV0(time.Now().UTC()))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != orquestacontext.CodeContextToolingEstadoErrorV0 || len(result.Issues) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func codebaseStatusLeaseStoreTestV0(
	t *testing.T,
	started time.Time,
) *orquestacontext.InMemoryCodeContextToolLeaseStoreV0 {
	t.Helper()
	leases := orquestacontext.NewInMemoryCodeContextToolLeaseStoreV0()
	if _, err := leases.BeginCodeContextToolLeaseV0(context.Background(), orquestacontext.CodeContextToolLeaseRequestV0{
		RequestRef:      "request-ref-codebase-status-test",
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "code-context-sha256-status-test",
		ToolRef:         "provider-ref-orquesta-code-context-central",
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 10,
	}); err != nil {
		t.Fatalf("begin lease: %v", err)
	}
	return leases
}

func codebaseStatusActiveLeaseTestV0(
	t *testing.T,
	leases *orquestacontext.InMemoryCodeContextToolLeaseStoreV0,
) orquestacontext.CodeContextToolLeaseV0 {
	t.Helper()
	active, err := leases.ListCodeContextToolLeasesV0(context.Background(), orquestacontext.CodeContextToolLeaseListFilterV0{
		Status: orquestacontext.CodeContextToolLeaseStatusActiveV0,
	})
	if err != nil {
		t.Fatalf("list active leases: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("active leases=%+v", active)
	}
	return active[0]
}

func validMCPCodebaseStatusInputTestV0(started time.Time) MCPCodebaseStatusToolInputV0 {
	return MCPCodebaseStatusToolInputV0{
		SchemaVersion: orquestacontext.CodeContextToolingStatusSchemaVersionV0,
		RequestRef:    "request-ref-mcp-codebase-status-test",
		RepositoryRef: "repo-ref-orquesta",
		ObservedAt:    started.Add(time.Minute).Format(time.RFC3339),
	}
}
