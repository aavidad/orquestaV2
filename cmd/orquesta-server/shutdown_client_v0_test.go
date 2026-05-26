package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRequestServerShutdownV0AceptaRespuestaJSONControlada(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v0/server/shutdown" {
			t.Fatalf("request inesperada: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(serverShutdownClientResultV0{
			Estado:        "ok",
			Status:        "ready",
			ShutdownReady: true,
		})
	}))
	defer server.Close()

	if err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://")); err != nil {
		t.Fatalf("requestServerShutdownV0: %v", err)
	}
}

func TestRequestServerShutdownV0NoPropagaBodyCrudoEnError(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.Error(w, "secret_token=abc local_path=/tmp/private", http.StatusInternalServerError)
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"))

	if err == nil {
		t.Fatalf("error esperado")
	}
	if got := err.Error(); got != "shutdown_http_500" {
		t.Fatalf("error=%q", got)
	}
	if strings.Contains(err.Error(), "secret_token") || strings.Contains(err.Error(), "/tmp/private") {
		t.Fatalf("error filtra body crudo: %v", err)
	}
}

func TestRequestServerShutdownV0RechazaJSONConTrailingData(t *testing.T) {
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"estado":"ok","shutdown_ready":true} {"extra":true}`))
	}))
	defer server.Close()

	err := requestServerShutdownV0(strings.TrimPrefix(server.URL, "http://"))

	if err == nil || err.Error() != "shutdown_response_trailing_data" {
		t.Fatalf("error=%v", err)
	}
}
