package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/internal/rpclocal"
)

func TestServerAPIReadyOKExigeServerDecodificable(t *testing.T) {
	var calls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/server", func(w http.ResponseWriter, r *http.Request) {
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
		_, _ = w.Write([]byte(`{"name":"orquesta","storageDriver":"sqlite"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if err := serverAPIReadyOK(srv.URL); err == nil {
		t.Fatalf("la primera lectura deberia fallar por EOF")
	}
	if err := serverAPIReadyOK(srv.URL); err != nil {
		t.Fatalf("la segunda lectura deberia pasar: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got < 2 {
		t.Fatalf("se esperaban al menos 2 llamadas a /api/server, got=%d", got)
	}
}

func TestServerAPIReadyOKFallaConHTTPNoOK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/server", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	err := serverAPIReadyOK(srv.URL)
	if err == nil || err.Error() != fmt.Sprintf("status %s", http.StatusText(http.StatusServiceUnavailable)) && err.Error() != "status 503 Service Unavailable" {
		if err == nil {
			t.Fatalf("se esperaba error readiness")
		}
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestWaitLocalServerStableExigeStatefilePublicada(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "server.json")
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", statePath)()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()
	dbPath := filepath.Join(t.TempDir(), "orquesta.db")
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          strings.TrimPrefix(strings.TrimPrefix(srvURLForRequest(r), "http://"), "https://"),
			PID:           4242,
			Kind:          "server",
			ScopeID:       rpclocal.CurrentScopeID(),
			StorageDriver: db.CurrentStorageDriver(),
			StorageTarget: dbPath,
			DBPath:        dbPath,
		})
	})
	mux.HandleFunc("/api/server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"orquesta"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	go func() {
		time.Sleep(150 * time.Millisecond)
		info := rpclocal.ServerInfo{
			Addr:          strings.TrimPrefix(strings.TrimPrefix(srv.URL, "http://"), "https://"),
			PID:           4242,
			Kind:          "server",
			ScopeID:       rpclocal.CurrentScopeID(),
			StorageDriver: db.CurrentStorageDriver(),
			StorageTarget: dbPath,
			DBPath:        dbPath,
		}
		_ = rpclocal.SaveServerInfo(info)
	}()

	if err := waitLocalServerStable(srv.URL, 2*time.Second); err != nil {
		t.Fatalf("waitLocalServerStable deberia esperar al statefile: %v", err)
	}
}

func TestWaitLocalServerStableExigeDosReadinessConsecutivos(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "server.json")
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", statePath)()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()
	dbPath := filepath.Join(t.TempDir(), "orquesta.db")
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	var statusCalls int32
	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          strings.TrimPrefix(strings.TrimPrefix(srvURLForRequest(r), "http://"), "https://"),
			PID:           4242,
			Kind:          "server",
			ScopeID:       rpclocal.CurrentScopeID(),
			StorageDriver: db.CurrentStorageDriver(),
			StorageTarget: dbPath,
			DBPath:        dbPath,
		})
	})
	mux.HandleFunc("/api/server", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&statusCalls, 1)
		if n == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"orquesta"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"orquesta"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	info := rpclocal.ServerInfo{
		Addr:          strings.TrimPrefix(strings.TrimPrefix(srv.URL, "http://"), "https://"),
		PID:           4242,
		Kind:          "server",
		ScopeID:       rpclocal.CurrentScopeID(),
		StorageDriver: db.CurrentStorageDriver(),
		StorageTarget: dbPath,
		DBPath:        dbPath,
	}
	_ = rpclocal.SaveServerInfo(info)

	if err := waitLocalServerStable(srv.URL, 2*time.Second); err != nil {
		t.Fatalf("waitLocalServerStable deberia exigir estabilidad real: %v", err)
	}
	if got := atomic.LoadInt32(&statusCalls); got < 2 {
		t.Fatalf("deberia exigir al menos dos readiness consecutivos, got=%d", got)
	}
}

func TestWaitLocalServerStableFallaSinStatefile(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "missing-server.json")
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", statePath)()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{OK: true})
	})
	mux.HandleFunc("/api/server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"orquesta"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if err := waitLocalServerStable(srv.URL, 250*time.Millisecond); err == nil {
		t.Fatalf("deberia fallar si el statefile no llega a publicarse")
	}
}

func srvURLForRequest(r *http.Request) string {
	if r == nil || r.Host == "" {
		return ""
	}
	scheme := "http://"
	if r.TLS != nil {
		scheme = "https://"
	}
	return scheme + r.Host
}
