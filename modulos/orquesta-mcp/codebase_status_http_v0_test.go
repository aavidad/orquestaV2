package orquestamcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestMCPCodebaseStatusHTTPHandlerV0PostDelega(t *testing.T) {
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	body, err := json.Marshal(validMCPCodebaseStatusInputTestV0(started))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPCodebaseStatusHTTPPathV0, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewMCPCodebaseStatusHTTPHandlerV0(MCPCodebaseStatusToolExecutorV0{
		Leases: codebaseStatusLeaseStoreTestV0(t, started),
		Clock:  func() time.Time { return started.Add(time.Minute) },
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPCodebaseStatusToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestacontext.CodeContextToolingEstadoAttentionRequiredV0 ||
		result.StopRequested != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPCodebaseStatusHTTPHandlerV0ObservacionesEvitanParadaConPeticionesActivas(t *testing.T) {
	started := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	leases := codebaseStatusLeaseStoreTestV0(t, started)
	lease := codebaseStatusActiveLeaseTestV0(t, leases)
	input := validMCPCodebaseStatusInputTestV0(started)
	input.Observations = []MCPCodebaseStatusToolObservationInputV0{{
		ObservationRef: "observation-ref-codebase-owner-marker-active",
		LeaseRef:       lease.LeaseRef,
		ObservedAt:     started.Add(time.Minute).Format(time.RFC3339),
		CPUPercent:     95,
		ActiveRequests: 2,
		EvidenceRefs:   []string{"evidence-ref-codebase-owner-marker-active-requests"},
	}}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPCodebaseStatusHTTPPathV0, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewMCPCodebaseStatusHTTPHandlerV0(MCPCodebaseStatusToolExecutorV0{
		Leases: leases,
		Clock:  func() time.Time { return started.Add(time.Minute) },
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPCodebaseStatusToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
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
	if containsMCPCodebaseStatusStringTestV0(result.NextActions, orquestacontext.CodeContextToolingActionStopExpiredLeaseV0) {
		t.Fatalf("next_actions no deben pedir parada: %+v", result.NextActions)
	}
	if !containsMCPCodebaseStatusStringTestV0(entry.EvidenceRefs, "evidence-ref-codebase-owner-marker-active-requests") {
		t.Fatalf("evidence refs=%+v", entry.EvidenceRefs)
	}
}

func TestMCPCodebaseStatusHTTPHandlerV0GetDevuelveCastellanoNoConfigured(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, MCPCodebaseStatusHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPCodebaseStatusHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != http.MethodPost+", OPTIONS" {
		t.Fatalf("allow=%q", rec.Header().Get("Allow"))
	}
}

func containsMCPCodebaseStatusStringTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
