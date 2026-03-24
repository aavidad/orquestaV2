package db

import (
	"path/filepath"
	"testing"
)

func TestResolveBackendPorDefectoSQLite(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB", filepath.Join(t.TempDir(), "orquesta.db"))

	backend, cfg, err := resolveBackend()
	if err != nil {
		t.Fatalf("resolveBackend: %v", err)
	}
	if backend.Name() != "sqlite" {
		t.Fatalf("backend inesperado: %s", backend.Name())
	}
	if configTarget(cfg) == "" {
		t.Fatalf("target vacío")
	}
}

func TestResolveBackendRespetaORQUESTADBDRIVER(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "sqlite3")
	t.Setenv("ORQUESTA_DB_BACKEND", "mysql")
	t.Setenv("ORQUESTA_DB", filepath.Join(t.TempDir(), "orquesta.db"))

	backend, _, err := resolveBackend()
	if err != nil {
		t.Fatalf("resolveBackend: %v", err)
	}
	if backend.Name() != "sqlite" {
		t.Fatalf("backend inesperado: %s", backend.Name())
	}
}

func TestResolveBackendFallaConBackendNoSoportado(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "mysql")
	t.Setenv("ORQUESTA_DB_BACKEND", "mysql")
	t.Setenv("ORQUESTA_DB_DSN", "usuario:pass@tcp(localhost:3306)/orquesta")

	_, _, err := resolveBackend()
	if err == nil {
		t.Fatalf("se esperaba error por backend no soportado")
	}
}

func TestBackupToRequiereBackendInicializado(t *testing.T) {
	prevDB := DB
	prevBackend := currentBackend
	DB = nil
	currentBackend = nil
	t.Cleanup(func() {
		DB = prevDB
		currentBackend = prevBackend
	})

	err := BackupTo(filepath.Join(t.TempDir(), "x.db"))
	if err == nil {
		t.Fatalf("se esperaba error sin DB inicializada")
	}
}

func TestRegisterBackendIgnoraNil(t *testing.T) {
	prev := len(backendRegistry)
	RegisterBackend(nil)
	if len(backendRegistry) != prev {
		t.Fatalf("register nil no debe alterar el registro")
	}
}

func TestBackupFilenameSuffixSQLitePorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "")
	t.Setenv("ORQUESTA_DB", filepath.Join(t.TempDir(), "orquesta.db"))

	if got := BackupFilenameSuffix(); got != "_orquesta.db.bak" {
		t.Fatalf("sufijo inesperado: %s", got)
	}
}

func TestBackupFilenameSuffixUsaDriverActual(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta")
	t.Setenv("ORQUESTA_DB", "")

	if got := BackupFilenameSuffix(); got != "_orquesta.postgres.bak" {
		t.Fatalf("sufijo inesperado: %s", got)
	}
}
