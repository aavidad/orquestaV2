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

func TestResolveBackendSoportaPostgres(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta?sslmode=disable")
	t.Setenv("ORQUESTA_DB", "")

	backend, cfg, err := resolveBackend()
	if err != nil {
		t.Fatalf("resolveBackend: %v", err)
	}
	if backend.Name() != "postgres" {
		t.Fatalf("backend inesperado: %s", backend.Name())
	}
	if configTarget(cfg) == "" {
		t.Fatalf("target vacío")
	}
}

func TestResolveBackendSoportaMySQL(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "mysql")
	t.Setenv("ORQUESTA_DB_BACKEND", "mysql")
	t.Setenv("ORQUESTA_DB_DSN", "usuario:pass@tcp(localhost:3306)/orquesta")

	backend, cfg, err := resolveBackend()
	if err != nil {
		t.Fatalf("resolveBackend: %v", err)
	}
	if backend.Name() != "mysql" {
		t.Fatalf("backend inesperado: %s", backend.Name())
	}
	if configTarget(cfg) == "" {
		t.Fatalf("target vacío")
	}
}

func TestResolveBackendFallaConBackendNoSoportado(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "oracle")
	t.Setenv("ORQUESTA_DB_BACKEND", "oracle")
	t.Setenv("ORQUESTA_DB_DSN", "oracle://user:pass@localhost/orquesta")

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

func TestCurrentStorageDisplayTargetRedactaDSN(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta?sslmode=disable")
	t.Setenv("ORQUESTA_DB", "")

	if got := CurrentStorageDisplayTarget(); got != "postgres://user:%2A%2A%2A@localhost/orquesta?sslmode=disable" {
		t.Fatalf("target visible inesperado: %s", got)
	}
}

func TestPostgresBackendBackupDevuelveErrorExplicito(t *testing.T) {
	err := (postgresBackend{}).Backup(nil, "/tmp/orquesta.postgres.bak")
	if err == nil {
		t.Fatalf("se esperaba error explicito para backup postgres")
	}
	if got := err.Error(); got == "" || got == "la base de datos no está inicializada" {
		t.Fatalf("mensaje de error poco especifico: %q", got)
	}
}

func TestMySQLBackendBackupDevuelveErrorExplicito(t *testing.T) {
	err := (mysqlBackend{}).Backup(nil, "/tmp/orquesta.mysql.bak")
	if err == nil {
		t.Fatalf("se esperaba error explicito para backup mysql")
	}
	if got := err.Error(); got == "" || got == "la base de datos no está inicializada" {
		t.Fatalf("mensaje de error poco especifico: %q", got)
	}
}
