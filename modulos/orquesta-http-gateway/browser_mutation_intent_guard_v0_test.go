package orquestahttpgateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrowserMutationIntentGuardV0BloqueaFormBrowserSinOrigenNiToken(t *testing.T) {
	handler := NewBrowserMutationIntentGuardV0(BrowserMutationIntentGuardConfigV0{}, markerHandler("ok"))
	req := httptest.NewRequest(http.MethodPost, RouteRunControlV0, strings.NewReader("run_ref=run-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden ||
		rec.Header().Get(BrowserMutationIntentDecisionHeaderV0) != BrowserMutationDecisionIntentMissingV0 {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestBrowserMutationIntentGuardV0PermiteOrigenSameOrigin(t *testing.T) {
	handler := NewBrowserMutationIntentGuardV0(BrowserMutationIntentGuardConfigV0{}, markerHandler("ok"))
	req := httptest.NewRequest(http.MethodPost, RouteRunControlV0, strings.NewReader("run_ref=run-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get(BrowserMutationIntentDecisionHeaderV0) != BrowserMutationDecisionOriginAllowedV0 {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestBrowserMutationIntentGuardV0BloqueaOriginCruzado(t *testing.T) {
	handler := NewBrowserMutationIntentGuardV0(BrowserMutationIntentGuardConfigV0{}, markerHandler("ok"))
	req := httptest.NewRequest(http.MethodPost, RouteRunControlV0, strings.NewReader(`{"run_ref":"run-1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden ||
		rec.Header().Get(BrowserMutationIntentDecisionHeaderV0) != BrowserMutationDecisionOriginBlockedV0 ||
		strings.Contains(rec.Body.String(), "evil.example") {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestBrowserMutationIntentGuardV0PermiteTokenDeIntencion(t *testing.T) {
	handler := NewBrowserMutationIntentGuardV0(BrowserMutationIntentGuardConfigV0{IntentToken: "token-intent-001"}, markerHandler("ok"))
	req := httptest.NewRequest(http.MethodPost, RouteRunControlV0, strings.NewReader("run_ref=run-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set(BrowserMutationIntentTokenHeaderV0, "token-intent-001")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get(BrowserMutationIntentReasonHeaderV0) != "intent_token" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestBrowserMutationIntentGuardV0PermiteClienteNoBrowser(t *testing.T) {
	handler := NewBrowserMutationIntentGuardV0(BrowserMutationIntentGuardConfigV0{}, markerHandler("ok"))
	req := httptest.NewRequest(http.MethodPost, RouteRunControlV0, strings.NewReader(`{"run_ref":"run-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK ||
		rec.Header().Get(BrowserMutationIntentDecisionHeaderV0) != BrowserMutationDecisionNonBrowserV0 {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
}
