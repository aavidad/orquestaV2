package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"orquesta/internal/rpclocal"
)

func TestServerReadinessOKExigeStatusDecodificable(t *testing.T) {
	var calls int32
	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatalf("ResponseWriter sin hijack")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"agentes":[],"conteoTareas":{}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if err := serverReadinessOK(srv.URL); err == nil {
		t.Fatalf("la primera lectura deberia fallar por EOF")
	}
	if err := serverReadinessOK(srv.URL); err != nil {
		t.Fatalf("la segunda lectura deberia pasar: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got < 2 {
		t.Fatalf("se esperaban al menos 2 llamadas a /api/status, got=%d", got)
	}
}

func TestServerReadinessOKFallaConHTTPNoOK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	err := serverReadinessOK(srv.URL)
	if err == nil || err.Error() != fmt.Sprintf("status %s", http.StatusText(http.StatusServiceUnavailable)) && err.Error() != "status 503 Service Unavailable" {
		if err == nil {
			t.Fatalf("se esperaba error readiness")
		}
		t.Fatalf("error inesperado: %v", err)
	}
}
