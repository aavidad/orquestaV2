package orquestaappgateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAppGatewayOpsDashboardRouteV0(t *testing.T) {
	handler := NewHTTPHandlerV0(ConfigV0{Timeout: time.Second})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ops", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Orquesta Ops") ||
		!strings.Contains(body, "/api/v0/autoprogramming/status") ||
		!strings.Contains(body, "/api/v0/server/resources") {
		t.Fatalf("ops html incompleto: %s", body)
	}
}
