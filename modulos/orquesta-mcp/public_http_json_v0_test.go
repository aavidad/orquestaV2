package orquestamcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMCPPublicHTTPJSONV0DistinguePerfilesDeLimiteV0(t *testing.T) {
	if mcpPublicHTTPJSONMaxBytesV0(mcpPublicHTTPJSONProfileControlV0) >=
		mcpPublicHTTPJSONMaxBytesV0(mcpPublicHTTPJSONProfileDomainWorkV0) {
		t.Fatalf("domain_work debe admitir mas body que control")
	}
	if mcpPublicHTTPJSONMaxBytesV0(mcpPublicHTTPJSONProfileAutoprogrammingV0) ==
		mcpPublicHTTPJSONMaxBytesV0(mcpPublicHTTPJSONProfileDomainWorkV0) {
		t.Fatalf("autoprogramacion y domain_work no deben compartir numero magico")
	}
	if got := mcpPublicHTTPJSONUnknownFieldsPolicyForProfileV0(mcpPublicHTTPJSONProfileControlV0); got != mcpPublicHTTPJSONUnknownFieldsLegacyV0 {
		t.Fatalf("unknown fields policy=%q", got)
	}
}

func TestMCPPublicHTTPJSONV0RechazaTooLargeYTrailingV0(t *testing.T) {
	var input map[string]string
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", strings.NewReader(`{} {}`))
	if code := decodeMCPPublicHTTPJSONV0(httptest.NewRecorder(), req, &input); code != "request_body_trailing_data" {
		t.Fatalf("code trailing=%q", code)
	}

	largeBody := strings.NewReader(`{"x":"` + strings.Repeat("a", int(mcpPublicHTTPJSONControlMaxBytesV0)+1) + `"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", largeBody)
	if code := decodeMCPPublicHTTPJSONV0(httptest.NewRecorder(), req, &input); code != "request_body_too_large" {
		t.Fatalf("code large=%q", code)
	}
}

func TestMCPPublicHTTPJSONV0RechazaContentTypeNoJSONYDeclaraLegacyExtraFieldsV0(t *testing.T) {
	var input struct {
		RequestID string `json:"request_id,omitempty"`
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", strings.NewReader(`{"request_id":"req-1"}`))
	req.Header.Set("Content-Type", "text/plain")
	if code := decodeMCPPublicHTTPJSONV0(httptest.NewRecorder(), req, &input); code != "request_content_type_invalido" {
		t.Fatalf("content-type code=%q", code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", strings.NewReader(`{"request_id":"req-1","legacy_extra":true}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if code := decodeMCPPublicHTTPJSONV0(httptest.NewRecorder(), req, &input); code != "" {
		t.Fatalf("legacy extra fields code=%q", code)
	}
	if input.RequestID != "req-1" {
		t.Fatalf("request_id=%q", input.RequestID)
	}
}
