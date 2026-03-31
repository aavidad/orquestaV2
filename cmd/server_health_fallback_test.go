package cmd

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/internal/rpclocal"
)

func TestLoadServerInfoWithHealthFallbackUsaHealthzSiFaltaStatefile(t *testing.T) {
	dbPath := t.TempDir() + "/orquesta.db"
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", t.TempDir()+"/missing-state.json")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          "127.0.0.1:19999",
			PID:           4242,
			Kind:          "server",
			ScopeID:       rpclocal.CurrentScopeID(),
			DBPath:        dbPath,
			StorageDriver: db.CurrentStorageDriver(),
			StorageTarget: dbPath,
			StartedAt:     time.Date(2026, 3, 30, 0, 10, 0, 0, time.UTC),
			Version:       "test",
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	info, recovered, err := loadServerInfoWithHealthFallback(srv.URL)
	if err != nil {
		t.Fatalf("loadServerInfoWithHealthFallback: %v", err)
	}
	if !recovered {
		t.Fatalf("se esperaba recuperacion desde healthz")
	}
	if info.Addr != "127.0.0.1:19999" {
		t.Fatalf("addr inesperada: %q", info.Addr)
	}
	if info.PID != 4242 || info.ScopeID != rpclocal.CurrentScopeID() || info.DBPath != dbPath || info.StorageTarget != dbPath || info.StorageDriver != db.CurrentStorageDriver() || info.Kind != "server" || info.Version != "test" {
		t.Fatalf("info inesperada: %+v", info)
	}
}

func TestLoadServerInfoWithHealthFallbackRecuperaStatefileStale(t *testing.T) {
	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, "server.json")
	dbPath := filepath.Join(stateDir, "orquesta.db")
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", statePath)()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	stale := rpclocal.ServerInfo{
		Addr:          "127.0.0.1:1",
		PID:           1111,
		Kind:          "server",
		ScopeID:       rpclocal.CurrentScopeID(),
		DBPath:        dbPath,
		StorageDriver: db.CurrentStorageDriver(),
		StorageTarget: dbPath,
		StartedAt:     time.Date(2026, 3, 31, 18, 0, 0, 0, time.UTC),
		Version:       "stale",
	}
	if err := rpclocal.SaveServerInfo(stale); err != nil {
		t.Fatalf("SaveServerInfo stale: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          "127.0.0.1:19998",
			PID:           4243,
			Kind:          "server",
			ScopeID:       rpclocal.CurrentScopeID(),
			DBPath:        dbPath,
			StorageDriver: db.CurrentStorageDriver(),
			StorageTarget: dbPath,
			StartedAt:     time.Date(2026, 3, 31, 18, 5, 0, 0, time.UTC),
			Version:       "fresh",
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	info, recovered, err := loadServerInfoWithHealthFallback(srv.URL)
	if err != nil {
		t.Fatalf("loadServerInfoWithHealthFallback stale: %v", err)
	}
	if !recovered {
		t.Fatalf("se esperaba recuperacion desde healthz con statefile stale")
	}
	if info.PID != 4243 || info.Addr != "127.0.0.1:19998" || info.Version != "fresh" {
		t.Fatalf("info recuperada inesperada: %+v", info)
	}
}

func TestLoadServerInfoWithHealthFallbackRechazaScopeAjeno(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", t.TempDir()+"/missing-state.json")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          "127.0.0.1:19999",
			PID:           4242,
			ScopeID:       "scope-ajeno",
			StorageDriver: db.CurrentStorageDriver(),
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	_, _, err := loadServerInfoWithHealthFallback(srv.URL)
	if err == nil || !strings.Contains(err.Error(), "otro scope") {
		t.Fatalf("se esperaba error por scope ajeno, got %v", err)
	}
}

func TestLoadServerInfoWithHealthFallbackRechazaDBAjena(t *testing.T) {
	dbPath := t.TempDir() + "/orquesta.db"
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", t.TempDir()+"/missing-state.json")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          "127.0.0.1:19999",
			PID:           4242,
			ScopeID:       rpclocal.CurrentScopeID(),
			DBPath:        t.TempDir() + "/otra.db",
			StorageDriver: db.CurrentStorageDriver(),
			StorageTarget: t.TempDir() + "/otra.db",
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	_, _, err := loadServerInfoWithHealthFallback(srv.URL)
	if err == nil || !strings.Contains(err.Error(), "otro storage target") {
		t.Fatalf("se esperaba error por db ajena, got %v", err)
	}
}

func TestLoadServerInfoWithHealthFallbackRechazaDriverAjeno(t *testing.T) {
	dbPath := t.TempDir() + "/orquesta.db"
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", t.TempDir()+"/missing-state.json")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          "127.0.0.1:19999",
			PID:           4242,
			ScopeID:       rpclocal.CurrentScopeID(),
			DBPath:        dbPath,
			StorageDriver: "mysql",
			StorageTarget: dbPath,
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	_, _, err := loadServerInfoWithHealthFallback(srv.URL)
	if err == nil || !strings.Contains(err.Error(), "otro driver de persistencia") {
		t.Fatalf("se esperaba error por driver ajeno, got %v", err)
	}
}
