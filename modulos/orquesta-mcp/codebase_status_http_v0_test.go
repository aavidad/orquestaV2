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
