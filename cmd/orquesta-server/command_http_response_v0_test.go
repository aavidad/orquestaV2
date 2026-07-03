package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestReadCommandHTTPResponseBodyV0ValidaJSONYContentType(t *testing.T) {
	response := commandHTTPResponseForTestV0(http.StatusOK, "application/json; charset=utf-8", `{"estado":"ok"}`)

	body, err := readCommandHTTPResponseBodyV0(response, "status")

	if err != nil {
		t.Fatalf("readCommandHTTPResponseBodyV0: %v", err)
	}
	if string(body) != `{"estado":"ok"}` {
		t.Fatalf("body=%s", string(body))
	}
}

func TestReadCommandHTTPResponseBodyV0AceptaLegacyTextPlainJSON(t *testing.T) {
	response := commandHTTPResponseForTestV0(http.StatusOK, "text/plain; charset=utf-8", `{"estado":"ok"}`)

	body, err := readCommandHTTPResponseBodyV0(response, "status")

	if err != nil {
		t.Fatalf("readCommandHTTPResponseBodyV0: %v", err)
	}
	if string(body) != `{"estado":"ok"}` {
		t.Fatalf("body=%s", string(body))
	}
}

func TestReadCommandHTTPResponseBodyV0NoPropagaBodyNo2xx(t *testing.T) {
	response := commandHTTPResponseForTestV0(http.StatusInternalServerError, "text/html", "secret_token=abc local_path=/tmp/private")

	_, err := readCommandHTTPResponseBodyV0(response, "run_status")

	if err == nil {
		t.Fatalf("error esperado")
	}
	if got := err.Error(); got != "run_status_http_500" {
		t.Fatalf("error=%q", got)
	}
}

func TestReadCommandHTTPResponseBodyV0PropagaCodigosPublicosJSONNo2xx(t *testing.T) {
	response := commandHTTPResponseForTestV0(
		http.StatusInternalServerError,
		"application/json; charset=utf-8",
		`{"estado":"error","errores_publicos":[{"code":"run_supervisor_execute_error"}],"diagnostics":[{"code":"supervisor_transition_error_but_agents_live"}]}`,
	)

	_, err := readCommandHTTPResponseBodyV0(response, "request")

	if err == nil {
		t.Fatalf("error esperado")
	}
	if got := err.Error(); got != "request_http_500:run_supervisor_execute_error,supervisor_transition_error_but_agents_live" {
		t.Fatalf("error=%q", got)
	}
}

func TestReadCommandHTTPResponseBodyAllowStatusV0PermiteConflictJSON(t *testing.T) {
	response := commandHTTPResponseForTestV0(
		http.StatusConflict,
		"application/json; charset=utf-8",
		`{"estado":"ok","status":"backend_still_running"}`,
	)

	body, err := readCommandHTTPResponseBodyAllowStatusV0(response, "shutdown", http.StatusConflict)

	if err != nil {
		t.Fatalf("readCommandHTTPResponseBodyAllowStatusV0: %v", err)
	}
	if string(body) != `{"estado":"ok","status":"backend_still_running"}` {
		t.Fatalf("body=%s", string(body))
	}
}

func TestReadCommandHTTPResponseBodyV0RechazaJSONConTrailingData(t *testing.T) {
	response := commandHTTPResponseForTestV0(http.StatusOK, "application/json", `{"estado":"ok"} {"extra":true}`)

	_, err := readCommandHTTPResponseBodyV0(response, "shutdown")

	if err == nil || err.Error() != "shutdown_response_trailing_data" {
		t.Fatalf("error=%v", err)
	}
}

func TestReadCommandHTTPResponseBodyV0RechazaContentTypeNoJSON(t *testing.T) {
	response := commandHTTPResponseForTestV0(http.StatusOK, "text/html", `{"estado":"ok"}`)

	_, err := readCommandHTTPResponseBodyV0(response, "status")

	if err == nil || err.Error() != "status_response_content_type" {
		t.Fatalf("error=%v", err)
	}
}

func TestReadCommandHTTPResponseBodyV0RechazaBodyGrande(t *testing.T) {
	response := commandHTTPResponseForTestV0(http.StatusOK, "application/json", strings.Repeat(" ", int(commandHTTPResponseMaxBytesV0)+1))

	_, err := readCommandHTTPResponseBodyV0(response, "status")

	if err == nil || err.Error() != "status_response_too_large" {
		t.Fatalf("error=%v", err)
	}
}

func commandHTTPResponseForTestV0(status int, contentType string, body string) *http.Response {
	header := http.Header{}
	if strings.TrimSpace(contentType) != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
