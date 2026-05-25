package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestagovernance "orquesta/modulos/orquesta-governance"
)

func TestGovernanceCatalogRouteV0DevuelveErrorPublicoSinProvider(t *testing.T) {
	handler := withGovernanceCatalogRouteV0(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		orquestagovernance.GovernanceCatalogQueryHTTPPathV0,
		strings.NewReader(`{"request_id":"req-gov-server","correlation_id":"corr-gov-server","filters":{"module":"cmd/orquesta-server"}}`),
	)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response orquestagovernance.GovernanceCatalogPublicErrorResponseV0
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.RequestID != "req-gov-server" ||
		response.CorrelationID != "corr-gov-server" ||
		len(response.Errors) != 1 ||
		response.Errors[0].Code != "governance_catalog_source_unavailable" {
		t.Fatalf("response=%+v", response)
	}
}

func TestGovernanceCatalogRouteV0PreservaFallbackHandler(t *testing.T) {
	handler := withGovernanceCatalogRouteV0(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("fallback"))
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v0/director/stats", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted || rec.Body.String() != "fallback" {
		t.Fatalf("fallback status=%d body=%q", rec.Code, rec.Body.String())
	}
}
