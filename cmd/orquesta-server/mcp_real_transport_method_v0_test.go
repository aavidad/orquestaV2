package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestMCPRealTransportV0MetodoYOptionsPublicosV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	for _, tc := range []struct {
		name       string
		method     string
		wantStatus int
		wantBody   string
	}{
		{name: "method", method: http.MethodGet, wantStatus: http.StatusMethodNotAllowed, wantBody: "mcp_method_not_allowed"},
		{name: "options", method: http.MethodOptions, wantStatus: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, mcpRealHTTPPathV0, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("Allow"); got != serverPublicHTTPAllowHeaderV0(http.MethodPost) {
				t.Fatalf("allow=%q", got)
			}
			if tc.wantBody == "" && rec.Body.Len() != 0 {
				t.Fatalf("body=%q", rec.Body.String())
			}
			if tc.wantBody != "" && !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body=%q want=%q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}
