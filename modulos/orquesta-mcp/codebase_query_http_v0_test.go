package orquestamcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func TestMCPCodebaseQueryHTTPHandlerV0PostDelega(t *testing.T) {
	body, err := json.Marshal(validMCPCodebaseQueryInputTestV0())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, MCPCodebaseQueryHTTPPathV0, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewMCPCodebaseQueryHTTPHandlerV0(MCPCodebaseQueryToolExecutorV0{
		Broker: &fakeMCPCodeContextBrokerV0{},
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPCodebaseQueryToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestacontext.CodeContextEstadoOKV0 || len(result.Results) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPCodebaseQueryHTTPHandlerV0GetDevuelveCastellanoNoConfigured(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, MCPCodebaseQueryHTTPPathV0, nil)
	rec := httptest.NewRecorder()

	NewMCPCodebaseQueryHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != http.MethodPost+", OPTIONS" {
		t.Fatalf("allow=%q", rec.Header().Get("Allow"))
	}
}
