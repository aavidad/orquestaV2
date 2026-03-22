package storage

import (
	"strings"
	"testing"
)

func TestResolveConfigSQLitePorDefectoUsaPathDelCallback(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_DSN", "")
	t.Setenv("ORQUESTA_DB", "")
	t.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", "")
	t.Setenv("ORQUESTA_DB_BOOTSTRAP", "")

	cfg, err := ResolveConfig(func() string { return "/tmp/orquesta.db" })
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if cfg.Driver != "sqlite" {
		t.Fatalf("driver inesperado: %s", cfg.Driver)
	}
	if cfg.Path != "/tmp/orquesta.db" {
		t.Fatalf("path inesperado: %s", cfg.Path)
	}
	if cfg.DSN != "/tmp/orquesta.db?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000" {
		t.Fatalf("dsn inesperado: %s", cfg.DSN)
	}
	if cfg.MaxOpenConns != 1 {
		t.Fatalf("MaxOpenConns inesperado: %d", cfg.MaxOpenConns)
	}
	if !cfg.BootstrapSchema {
		t.Fatalf("sqlite deberia arrancar con bootstrap activo por defecto")
	}
}

func TestResolveConfigUsaDriverYDSNDesdeEntorno(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta?sslmode=disable")
	t.Setenv("ORQUESTA_DB_BOOTSTRAP", "false")
	t.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", "16")

	cfg, err := ResolveConfig(func() string { return "/tmp/ignorado.db" })
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if cfg.Driver != "postgres" {
		t.Fatalf("driver inesperado: %s", cfg.Driver)
	}
	if cfg.DSN != "postgres://user:pass@localhost/orquesta?sslmode=disable" {
		t.Fatalf("dsn inesperado: %s", cfg.DSN)
	}
	if cfg.Path != "" {
		t.Fatalf("path no deberia usarse en postgres: %s", cfg.Path)
	}
	if cfg.MaxOpenConns != 16 {
		t.Fatalf("MaxOpenConns inesperado: %d", cfg.MaxOpenConns)
	}
	if cfg.BootstrapSchema {
		t.Fatalf("bootstrap no deberia activarse para postgres en este test")
	}
}

func TestResolveConfigNormalizaSQLite3ASQLite(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "sqlite3")
	t.Setenv("ORQUESTA_DB_DSN", "")
	t.Setenv("ORQUESTA_DB", "/tmp/orquesta.db")

	cfg, err := ResolveConfig(func() string { return "/tmp/ignorado.db" })
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if cfg.Driver != "sqlite" {
		t.Fatalf("driver inesperado: %s", cfg.Driver)
	}
	if cfg.MaxOpenConns != 1 {
		t.Fatalf("MaxOpenConns inesperado: %d", cfg.MaxOpenConns)
	}
	if !cfg.BootstrapSchema {
		t.Fatalf("sqlite deberia arrancar con bootstrap activo")
	}
}

func TestResolveConfigFallaSinDSNParaMySQLYPostgres(t *testing.T) {
	for _, driver := range []string{"mysql", "postgres"} {
		t.Run(driver, func(t *testing.T) {
			t.Setenv("ORQUESTA_DB_DRIVER", driver)
			t.Setenv("ORQUESTA_DB_DSN", "")
			t.Setenv("ORQUESTA_DB", "")

			if _, err := ResolveConfig(func() string { return "/tmp/orquesta.db" }); err == nil {
				t.Fatalf("esperaba error para driver %s sin DSN", driver)
			}
		})
	}
}

func TestSQLiteDSNAniadeParametrosSinRomperQueryExistente(t *testing.T) {
	got := SQLiteDSN("/tmp/orquesta.db?cache=shared")
	want := "/tmp/orquesta.db?cache=shared&_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"
	if got != want {
		t.Fatalf("dsn inesperado: %s", got)
	}
}

func TestOpenFallaConDriverExternoNoEnlazado(t *testing.T) {
	for _, driver := range []string{"postgres", "mysql"} {
		t.Run(driver, func(t *testing.T) {
			_, err := Open(Config{
				Driver: driver,
				DSN:    "dsn://usuario:clave@localhost/orquesta",
			})
			if err == nil {
				t.Fatalf("esperaba error para driver %s no enlazado", driver)
			}
			if !strings.Contains(err.Error(), "no está enlazado en el binario") {
				t.Fatalf("error no explicito para %s: %v", driver, err)
			}
		})
	}
}
