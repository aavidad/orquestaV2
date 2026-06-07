package orquestamcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPRuntimeModelsHTTPHandlerV0DelegaEnPuerto(t *testing.T) {
	port := &fakeMCPRuntimeModelsPortV0{}
	body := bytes.NewBuffer(nil)
	_ = json.NewEncoder(body).Encode(MCPRuntimeModelsToolInputV0{
		Action: "list",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPRuntimeModelsHTTPPathV0, body)

	NewMCPRuntimeModelsHTTPHandlerV0(port).ServeHTTP(rec, req)

	var result MCPRuntimeModelsToolResultV0
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if rec.Code != http.StatusOK ||
		result.Estado != MCPRuntimeModelsEstadoOKV0 ||
		result.ListResult == nil ||
		len(result.ListResult.Models) != 1 {
		t.Fatalf("status=%d result=%+v body=%s", rec.Code, result, rec.Body.String())
	}
}

func TestMCPRuntimeModelsHTTPHandlerV0MetodoYPuertoNil(t *testing.T) {
	for _, tc := range []struct {
		name    string
		method  string
		handler http.Handler
		want    int
	}{
		{name: "metodo", method: http.MethodGet, handler: NewMCPRuntimeModelsHTTPHandlerV0(&fakeMCPRuntimeModelsPortV0{}), want: http.StatusMethodNotAllowed},
		{name: "nil", method: http.MethodPost, handler: NewMCPRuntimeModelsHTTPHandlerV0(nil), want: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, MCPRuntimeModelsHTTPPathV0, bytes.NewBufferString(`{}`))

			tc.handler.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
