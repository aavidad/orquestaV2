package orquestamcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMCPAutoprogrammingHTTPV0StatusYSuperviseUsanPerfilAutoprogrammingV0(t *testing.T) {
	body := `{"request_id":"request-ref-autop-bounds","operator_advice":"` +
		strings.Repeat("a", int(mcpPublicHTTPJSONControlMaxBytesV0)+1024) + `"}`
	for _, tc := range []struct {
		name    string
		path    string
		handler http.Handler
	}{
		{
			name:    "status",
			path:    MCPAutoprogrammingStatusHTTPPathV0,
			handler: NewMCPAutoprogrammingStatusHTTPHandlerV0(nil),
		},
		{
			name:    "supervise",
			path:    MCPAutoprogrammingSuperviseHTTPPathV0,
			handler: NewMCPAutoprogrammingSuperviseHTTPHandlerV0(nil),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			tc.handler.ServeHTTP(rec, req)

			if rec.Code == http.StatusBadRequest || strings.Contains(rec.Body.String(), "request_body_too_large") {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
