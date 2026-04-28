/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package storage

import "testing"

func TestResolveConfigSQLitePorDefectoUsaPathDelCallback(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
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
	if cfg.DSN != "/tmp/orquesta.db?_journal_mode=WAL&_synchronous=NORMAL&_wal_autocheckpoint=100&_foreign_keys=on&_busy_timeout=30000" {
		t.Fatalf("dsn inesperado: %s", cfg.DSN)
	}
	if cfg.MaxOpenConns != 1 {
		t.Fatalf("MaxOpenConns inesperado: %d", cfg.MaxOpenConns)
	}
	if !cfg.BootstrapSchema {
		t.Fatalf("sqlite deberia arrancar con bootstrap activo por defecto")
	}
}

func TestResolveConfigFallaSinConectorNiTargetExplicito(t *testing.T) {
	t.Setenv("ORQUESTA_PERSISTENCE_CONNECTOR", "")
	t.Setenv("ORQUESTA_DB_CONNECTOR", "")
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "")
	t.Setenv("ORQUESTA_DB", "")

	if _, err := ResolveConfig(func() string { return "" }); err == nil {
		t.Fatalf("se esperaba error sin conector ni target explicito")
	}
}

func TestResolveConfigRequireExplicitPersistenceNoUsaFallbackLocal(t *testing.T) {
	t.Setenv("ORQUESTA_REQUIRE_EXPLICIT_PERSISTENCE", "1")
	t.Setenv("ORQUESTA_PERSISTENCE_CONNECTOR", "")
	t.Setenv("ORQUESTA_DB_CONNECTOR", "")
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "")
	t.Setenv("ORQUESTA_DB", "")

	if _, err := ResolveConfig(func() string { return "/tmp/orquesta.db" }); err == nil {
		t.Fatalf("se esperaba error al exigir persistencia explicita")
	}
}

func TestResolveConfigRequireExplicitPersistenceAceptaPostgresPorDSN(t *testing.T) {
	t.Setenv("ORQUESTA_REQUIRE_EXPLICIT_PERSISTENCE", "1")
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta?sslmode=disable")
	t.Setenv("ORQUESTA_DB", "")

	cfg, err := ResolveConfig(func() string { return "/tmp/orquesta.db" })
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if cfg.Driver != "postgres" {
		t.Fatalf("driver inesperado: %s", cfg.Driver)
	}
}

func TestResolveConfigAceptaBackendLegacy(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "sqlite3")
	t.Setenv("ORQUESTA_DB_DSN", "")
	t.Setenv("ORQUESTA_DB", "/tmp/orquesta.db")

	cfg, err := ResolveConfig(func() string { return "/tmp/ignorado.db" })
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if cfg.Driver != "sqlite" {
		t.Fatalf("driver inesperado: %s", cfg.Driver)
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

func TestResolveConfigInfierePostgresDesdeDSNYActivaBootstrapPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "")
	t.Setenv("ORQUESTA_DB_BACKEND", "")
	t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta?sslmode=disable")
	t.Setenv("ORQUESTA_DB", "")
	t.Setenv("ORQUESTA_DB_BOOTSTRAP", "")

	cfg, err := ResolveConfig(func() string { return "" })
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if cfg.Driver != "postgres" {
		t.Fatalf("driver inesperado: %s", cfg.Driver)
	}
	if !cfg.BootstrapSchema {
		t.Fatalf("postgres deberia arrancar con bootstrap activo por defecto")
	}
}

func TestResolveConfigPostgresUsaPoolMasAmplioPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta?sslmode=disable")
	t.Setenv("ORQUESTA_DB", "")
	t.Setenv("ORQUESTA_DB_BOOTSTRAP", "")
	t.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", "")

	cfg, err := ResolveConfig(func() string { return "" })
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if cfg.MaxOpenConns != 32 {
		t.Fatalf("MaxOpenConns inesperado: %d", cfg.MaxOpenConns)
	}
}

func TestResolveConfigNormalizaDrivers(t *testing.T) {
	cases := []struct {
		envValue string
		want     string
	}{
		{envValue: "sqlite3", want: "sqlite"},
		{envValue: "postgresql", want: "postgres"},
	}

	for _, tc := range cases {
		t.Run(tc.envValue, func(t *testing.T) {
			t.Setenv("ORQUESTA_DB_DRIVER", tc.envValue)
			t.Setenv("ORQUESTA_DB_DSN", "postgres://user:pass@localhost/orquesta?sslmode=disable")
			t.Setenv("ORQUESTA_DB", "/tmp/orquesta.db")

			cfg, err := ResolveConfig(func() string { return "/tmp/ignorado.db" })
			if err != nil {
				t.Fatalf("ResolveConfig: %v", err)
			}
			if cfg.Driver != tc.want {
				t.Fatalf("driver inesperado: %s", cfg.Driver)
			}
		})
	}
}

func TestResolveConfigFallaSinDSNParaDriversExternos(t *testing.T) {
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
	want := "/tmp/orquesta.db?cache=shared&_journal_mode=WAL&_synchronous=NORMAL&_wal_autocheckpoint=100&_foreign_keys=on&_busy_timeout=30000"
	if got != want {
		t.Fatalf("dsn inesperado: %s", got)
	}
}

func TestOpenFallaConDriverNoEnlazado(t *testing.T) {
	_, err := Open(Config{
		Driver: "oracle",
		DSN:    "dsn://usuario:clave@localhost/orquesta",
	})
	if err == nil {
		t.Fatalf("esperaba error para driver no enlazado")
	}
}

func TestDisplayTargetRedactaPasswordEnDSNURL(t *testing.T) {
	cfg := Config{
		Driver: "postgres",
		DSN:    "postgres://user:secret@localhost/orquesta?sslmode=disable",
	}

	got := DisplayTarget(cfg)
	if got != "postgres://user:%2A%2A%2A@localhost/orquesta?sslmode=disable" {
		t.Fatalf("display target inesperado: %s", got)
	}
}

func TestDisplayTargetRedactaPasswordEnDSNNoURL(t *testing.T) {
	cfg := Config{
		Driver: "mysql",
		DSN:    "user:secret@tcp(localhost:3306)/orquesta?parseTime=true",
	}

	got := DisplayTarget(cfg)
	if got != "user:***@tcp(localhost:3306)/orquesta?parseTime=true" {
		t.Fatalf("display target inesperado: %s", got)
	}
}

func TestDisplayTargetMantienePathEnSQLite(t *testing.T) {
	cfg := Config{
		Driver: "sqlite",
		Path:   "/tmp/orquesta.db",
		DSN:    "/tmp/orquesta.db?_busy_timeout=30000",
	}

	got := DisplayTarget(cfg)
	if got != "/tmp/orquesta.db" {
		t.Fatalf("display target inesperado: %s", got)
	}
}
