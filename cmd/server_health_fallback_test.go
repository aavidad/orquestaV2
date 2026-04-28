package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

func TestLoadServerInfoWithHealthFallbackEliminaStatefileMuerto(t *testing.T) {
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
		PID:           99999999,
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

	_, _, err := loadServerInfoWithHealthFallback("http://127.0.0.1:1")
	if err == nil {
		t.Fatal("deberia fallar sin healthz")
	}
	if _, statErr := os.Stat(statePath); !os.IsNotExist(statErr) {
		t.Fatalf("deberia eliminar statefile stale, statErr=%v", statErr)
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

func TestEnsureServerInfoFromHealthRecuperaYPublicaStatefile(t *testing.T) {
	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, "server.json")
	dbPath := filepath.Join(stateDir, "orquesta.db")
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", statePath)()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          "127.0.0.1:18888",
			PID:           5555,
			Kind:          "server",
			ScopeID:       rpclocal.CurrentScopeID(),
			DBPath:        dbPath,
			StorageDriver: db.CurrentStorageDriver(),
			StorageTarget: dbPath,
			StartedAt:     time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC),
			Version:       "test",
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	info, err := ensureServerInfoFromHealth(srv.URL)
	if err != nil {
		t.Fatalf("ensureServerInfoFromHealth: %v", err)
	}
	if info == nil || info.PID != 5555 || info.Addr != "127.0.0.1:18888" {
		t.Fatalf("info inesperada: %+v", info)
	}

	state, err := rpclocal.LoadServerInfo()
	if err != nil {
		t.Fatalf("LoadServerInfo tras recovery: %v", err)
	}
	if state.PID != 5555 || state.Addr != "127.0.0.1:18888" {
		t.Fatalf("statefile inesperado: %+v", state)
	}
}

func TestFindExistingServerForCurrentStorageEncuentraServidorDeOtroScope(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "orquesta.db")
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	stateDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(stateDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          "127.0.0.1:18887",
			PID:           7777,
			Kind:          "server",
			ScopeID:       "scope-ajeno",
			DBPath:        dbPath,
			StorageDriver: db.CurrentStorageDriver(),
			StorageTarget: dbPath,
			StartedAt:     time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
			Version:       "test",
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	statePath := filepath.Join(os.TempDir(), "orquesta-localrpc-scope-ajeno-test.json")
	t.Cleanup(func() { _ = os.Remove(statePath) })
	if err := os.WriteFile(statePath, []byte(fmt.Sprintf(`{
  "addr": "%s",
  "pid": 7777,
  "kind": "server",
  "scope_id": "scope-ajeno",
  "db_path": %q,
  "storage_driver": %q,
  "storage_target": %q,
  "started_at": "2026-04-22T09:00:00Z",
  "version": "test"
}`, strings.TrimPrefix(strings.TrimPrefix(srv.URL, "http://"), "https://"), dbPath, db.CurrentStorageDriver(), dbPath)), 0o600); err != nil {
		t.Fatalf("WriteFile state: %v", err)
	}

	info, err := findExistingServerForCurrentStorage()
	if err != nil {
		t.Fatalf("findExistingServerForCurrentStorage: %v", err)
	}
	if info != nil {
		t.Fatalf("no deberia aceptar un servidor de scope ajeno: %+v", info)
	}
}

func TestFindExistingServerForCurrentStorageIgnoraStorageAjeno(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "orquesta.db")
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:            true,
			Addr:          "127.0.0.1:18886",
			PID:           6666,
			Kind:          "server",
			ScopeID:       "scope-ajeno",
			DBPath:        filepath.Join(t.TempDir(), "otra.db"),
			StorageDriver: db.CurrentStorageDriver(),
			StorageTarget: filepath.Join(t.TempDir(), "otra.db"),
			StartedAt:     time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
			Version:       "test",
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	statePath := filepath.Join(os.TempDir(), "orquesta-localrpc-scope-ajeno-otra-db-test.json")
	t.Cleanup(func() { _ = os.Remove(statePath) })
	if err := os.WriteFile(statePath, []byte(fmt.Sprintf(`{
  "addr": "%s",
  "pid": 6666,
  "kind": "server",
  "scope_id": "scope-ajeno",
  "db_path": %q,
  "storage_driver": %q,
  "storage_target": %q,
  "started_at": "2026-04-22T09:00:00Z",
  "version": "test"
}`, strings.TrimPrefix(strings.TrimPrefix(srv.URL, "http://"), "https://"), filepath.Join(t.TempDir(), "otra.db"), db.CurrentStorageDriver(), filepath.Join(t.TempDir(), "otra.db"))), 0o600); err != nil {
		t.Fatalf("WriteFile state: %v", err)
	}

	info, err := findExistingServerForCurrentStorage()
	if err != nil {
		t.Fatalf("findExistingServerForCurrentStorage: %v", err)
	}
	if info != nil {
		t.Fatalf("no deberia reutilizar servidor de otro storage: %+v", info)
	}
}

func TestFindExistingServerForCurrentStorageEliminaStatefileMuerto(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "orquesta.db")
	defer cambiarEnv(t, "ORQUESTA_DB", dbPath)()
	defer cambiarEnv(t, "ORQUESTA_DB_DRIVER", "")()
	defer cambiarEnv(t, "ORQUESTA_DB_DSN", "")()

	statePath := filepath.Join(os.TempDir(), "orquesta-localrpc-dead-pid-test.json")
	t.Cleanup(func() { _ = os.Remove(statePath) })
	if err := os.WriteFile(statePath, []byte(fmt.Sprintf(`{
  "addr": "127.0.0.1:1",
  "pid": 99999999,
  "kind": "server",
  "scope_id": %q,
  "db_path": %q,
  "storage_driver": %q,
  "storage_target": %q,
  "started_at": "2026-04-22T09:00:00Z",
  "version": "test"
}`, rpclocal.CurrentScopeID(), dbPath, db.CurrentStorageDriver(), dbPath)), 0o600); err != nil {
		t.Fatalf("WriteFile state: %v", err)
	}

	info, err := findExistingServerForCurrentStorage()
	if err != nil {
		t.Fatalf("findExistingServerForCurrentStorage: %v", err)
	}
	if info != nil {
		t.Fatalf("no deberia devolver servidor para state muerto: %+v", info)
	}
	if _, statErr := os.Stat(statePath); !os.IsNotExist(statErr) {
		t.Fatalf("deberia eliminar state muerto, statErr=%v", statErr)
	}
}
