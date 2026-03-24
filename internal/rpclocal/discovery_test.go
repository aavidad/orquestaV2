package rpclocal

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestDefaultAddr(t *testing.T) {
	prev := os.Getenv(envAddr)
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv(envAddr)
			return
		}
		_ = os.Setenv(envAddr, prev)
	})

	if err := os.Setenv(envAddr, "127.0.0.1:19000"); err != nil {
		t.Fatalf("Setenv: %v", err)
	}
	if got := DefaultAddr(); got != "http://127.0.0.1:19000" {
		t.Fatalf("DefaultAddr inesperado: %s", got)
	}
}

func TestSaveLoadAndRemoveState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc-state.json")
	want := &State{
		Addr:      "127.0.0.1:16543",
		PID:       42,
		ScopeID:   CurrentScopeID(),
		DBPath:    "/tmp/orquesta.db",
		StartedAt: time.Date(2026, 3, 22, 20, 0, 0, 0, time.UTC),
		Version:   "test",
	}
	if err := SaveState(path, want); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	got, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.Addr != want.Addr || got.PID != want.PID || got.ScopeID != want.ScopeID || got.DBPath != want.DBPath || !got.StartedAt.Equal(want.StartedAt) || got.Version != want.Version {
		t.Fatalf("state inesperado: %+v", got)
	}

	if err := RemoveState(path); err != nil {
		t.Fatalf("RemoveState: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("se esperaba eliminar %s", path)
	}
}

func TestSaveStateUsesPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permisos POSIX no aplican en windows")
	}

	path := filepath.Join(t.TempDir(), "rpc-state.json")
	if err := SaveState(path, &State{
		Addr:      "127.0.0.1:16543",
		PID:       42,
		ScopeID:   CurrentScopeID(),
		StartedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("permisos inesperados: %o", mode)
	}
}

func TestBaseURL(t *testing.T) {
	if got := BaseURL("127.0.0.1:16543"); got != "http://127.0.0.1:16543" {
		t.Fatalf("BaseURL addr inesperado: %s", got)
	}
	if got := BaseURL("http://127.0.0.1:16543/"); got != "http://127.0.0.1:16543" {
		t.Fatalf("BaseURL url inesperado: %s", got)
	}
}

func TestDefaultStatePathScopedByDB(t *testing.T) {
	prev := os.Getenv("ORQUESTA_DB")
	prevDriver := os.Getenv("ORQUESTA_DB_DRIVER")
	prevDSN := os.Getenv("ORQUESTA_DB_DSN")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prev)
		}
		if prevDriver == "" {
			_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
		} else {
			_ = os.Setenv("ORQUESTA_DB_DRIVER", prevDriver)
		}
		if prevDSN == "" {
			_ = os.Unsetenv("ORQUESTA_DB_DSN")
		} else {
			_ = os.Setenv("ORQUESTA_DB_DSN", prevDSN)
		}
	})
	_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
	_ = os.Unsetenv("ORQUESTA_DB_DSN")

	if err := os.Setenv("ORQUESTA_DB", "/tmp/orquesta-a.db"); err != nil {
		t.Fatalf("Setenv a: %v", err)
	}
	pathA := DefaultStatePath()
	if err := os.Setenv("ORQUESTA_DB", "/tmp/orquesta-b.db"); err != nil {
		t.Fatalf("Setenv b: %v", err)
	}
	pathB := DefaultStatePath()
	if pathA == pathB {
		t.Fatalf("se esperaban statefiles distintos por DB: %s", pathA)
	}
}

func TestDefaultStatePathScopedByDSN(t *testing.T) {
	prevDB := os.Getenv("ORQUESTA_DB")
	prevDriver := os.Getenv("ORQUESTA_DB_DRIVER")
	prevDSN := os.Getenv("ORQUESTA_DB_DSN")
	t.Cleanup(func() {
		if prevDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prevDB)
		}
		if prevDriver == "" {
			_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
		} else {
			_ = os.Setenv("ORQUESTA_DB_DRIVER", prevDriver)
		}
		if prevDSN == "" {
			_ = os.Unsetenv("ORQUESTA_DB_DSN")
		} else {
			_ = os.Setenv("ORQUESTA_DB_DSN", prevDSN)
		}
	})

	_ = os.Unsetenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB_DRIVER", "postgres"); err != nil {
		t.Fatalf("Setenv driver: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta_a"); err != nil {
		t.Fatalf("Setenv dsn a: %v", err)
	}
	pathA := DefaultStatePath()
	if err := os.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta_b"); err != nil {
		t.Fatalf("Setenv dsn b: %v", err)
	}
	pathB := DefaultStatePath()
	if pathA == pathB {
		t.Fatalf("se esperaban statefiles distintos por DSN: %s", pathA)
	}
}

func TestLoadStateScopeMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rpc-state.json")
	if err := SaveState(path, &State{
		Addr:      "127.0.0.1:17899",
		PID:       42,
		ScopeID:   "otro-scope",
		StartedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	if _, err := LoadState(path); err == nil {
		t.Fatalf("se esperaba error por scope mismatch")
	}
}
