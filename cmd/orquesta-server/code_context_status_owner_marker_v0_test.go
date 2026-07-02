package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestServerCodebaseStatusOwnerMarkerExecutorV0InyectaPeticionesActivas(t *testing.T) {
	started := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	observedAt := started.Add(2 * time.Second)
	stateDir, store, lease := codebaseStatusOwnerMarkerFixtureV0(t, started)

	registry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
	if err := registry.WriteCodeContextToolOwnerMarkerV0(serverCodeContextToolOwnerMarkerV0{
		OwnerRef:        lease.OwnerRef,
		ToolRef:         lease.ToolRef,
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		PID:             0,
		StartedAt:       started.Format(time.RFC3339),
		LastHeartbeatAt: observedAt.Format(time.RFC3339),
		CPUPercent:      95,
		ActiveRequests:  2,
		EvidenceRefs:    []string{"evidence-ref-codebase-owner-marker-active-requests"},
	}); err != nil {
		t.Fatalf("WriteCodeContextToolOwnerMarkerV0: %v", err)
	}
	executor := serverCodebaseStatusExecutorWithOwnerMarkersV0(
		orquestamcp.MCPCodebaseStatusToolExecutorV0{
			Leases: store,
			Clock:  func() time.Time { return observedAt },
		},
		store,
		serverFileCodeContextToolOwnerObserverV0{Registry: registry},
		func() time.Time { return observedAt },
	)

	result, err := executor.Execute(context.Background(), orquestamcp.MCPCodebaseStatusToolInputV0{
		RequestRef: "request-ref-codebase-status-owner-marker",
		ObservedAt: observedAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	assertCodebaseStatusOwnerMarkerActiveRequestsV0(t, result)
}

func TestBuildServerAppHandlerV0CodebaseStatusPublicoUsaOwnerMarkersFileBased(t *testing.T) {
	started := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	observedAt := started.Add(2 * time.Second)
	stateDir, store, lease := codebaseStatusOwnerMarkerFixtureV0(t, started)

	registry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
	if err := registry.WriteCodeContextToolOwnerMarkerV0(serverCodeContextToolOwnerMarkerV0{
		OwnerRef:        lease.OwnerRef,
		ToolRef:         lease.ToolRef,
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		StartedAt:       started.Format(time.RFC3339),
		LastHeartbeatAt: observedAt.Format(time.RFC3339),
		CPUPercent:      95,
		ActiveRequests:  2,
		EvidenceRefs:    []string{"evidence-ref-codebase-owner-marker-active-requests"},
	}); err != nil {
		t.Fatalf("WriteCodeContextToolOwnerMarkerV0: %v", err)
	}
	statusExecutor := serverCodebaseStatusExecutorWithOwnerMarkersV0(
		orquestamcp.MCPCodebaseStatusToolExecutorV0{
			Leases: store,
			Clock:  func() time.Time { return observedAt },
		},
		store,
		serverFileCodeContextToolOwnerObserverV0{Registry: registry},
		func() time.Time { return observedAt },
	)
	handler, err := buildServerAppHandlerV0(orquestaappcodexstack.StackV0{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "fallback-status-route-not-used", http.StatusTeapot)
		}),
		MCPTransportBindings: orquestamcp.MCPTransportBindingsV0{
			CodebaseStatus: statusExecutor,
		},
	})
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	body, err := json.Marshal(orquestamcp.MCPCodebaseStatusToolInputV0{
		RequestRef: "request-ref-codebase-status-public-owner-marker",
		ObservedAt: observedAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, orquestamcp.MCPCodebaseStatusHTTPPathV0, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPCodebaseStatusToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertCodebaseStatusOwnerMarkerActiveRequestsV0(t, result)
}

func codebaseStatusOwnerMarkerFixtureV0(
	t *testing.T,
	started time.Time,
) (string, *serverFileCodeContextToolLeaseStoreV0, orquestacontext.CodeContextToolLeaseV0) {
	t.Helper()
	stateDir := t.TempDir()
	store := newServerFileCodeContextToolLeaseStoreV0(filepath.Join(stateDir, serverCodeContextLeasesFileV0))
	lease, err := store.BeginCodeContextToolLeaseV0(context.Background(), orquestacontext.CodeContextToolLeaseRequestV0{
		RepositoryRef:   "repo-ref-orquesta",
		QueryHash:       "query-hash-owner-marker-status",
		ToolRef:         "tool-ref-codebase-central",
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		OwnerRef:        "owner-ref-codebase-status-active",
		StartedAt:       started.Format(time.RFC3339),
		LeaseTTLSeconds: 1,
	})
	if err != nil {
		t.Fatalf("BeginCodeContextToolLeaseV0: %v", err)
	}
	return stateDir, store, lease
}

func assertCodebaseStatusOwnerMarkerActiveRequestsV0(
	t *testing.T,
	result orquestacontext.CodeContextToolingStatusV0,
) {
	t.Helper()
	if result.Estado != orquestacontext.CodeContextToolingEstadoOKV0 ||
		result.StopRequested != 0 ||
		result.HighCPUStopRequested != 0 ||
		len(result.Entries) != 1 {
		t.Fatalf("result=%+v", result)
	}
	entry := result.Entries[0]
	if entry.Decision != orquestacontext.CodeContextToolLeaseDecisionContinueV0 ||
		entry.ReasonCode != orquestacontext.CodeContextToolLeaseReasonActiveRequestsV0 ||
		entry.CPUPercent != 95 ||
		entry.ActiveRequests != 2 ||
		entry.ShouldRequestStop {
		t.Fatalf("entry=%+v", entry)
	}
	if codeContextEvidenceContainsForTestV0(result.NextActions, orquestacontext.CodeContextToolingActionStopExpiredLeaseV0) {
		t.Fatalf("next_actions no deben pedir parada: %+v", result.NextActions)
	}
	if !codeContextEvidenceContainsForTestV0(entry.EvidenceRefs, "evidence-ref-codebase-owner-marker-active-requests") {
		t.Fatalf("entry evidence=%+v", entry.EvidenceRefs)
	}
}
