package db

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"orquesta/storage"
)

func TestOpenPostgresBootstrapLimpio(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ORQUESTA_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("ORQUESTA_TEST_POSTGRES_DSN no definido; se omite el test de integración con Postgres")
	}

	Close()

	adminDB, err := storage.Open(storage.Config{
		Driver:       "postgres",
		DSN:          dsn,
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("abriendo conexion de limpieza: %v", err)
	}
	defer adminDB.Close()

	if err := adminDB.Ping(); err != nil {
		t.Fatalf("ping conexion de limpieza: %v", err)
	}
	schemaName := fmt.Sprintf("orquesta_test_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(`CREATE SCHEMA ` + quotePostgresIdent(schemaName)); err != nil {
		t.Fatalf("creando schema de prueba: %v", err)
	}
	defer func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS ` + quotePostgresIdent(schemaName) + ` CASCADE`)
	}()

	schemaDSN, err := postgresTestDSNWithSearchPath(dsn, schemaName)
	if err != nil {
		t.Fatalf("construyendo dsn con search_path: %v", err)
	}

	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_DSN", schemaDSN)
	t.Setenv("ORQUESTA_DB_BOOTSTRAP", "true")
	t.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", "1")

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer Close()

	if got := DriverName(); got != "postgres" {
		t.Fatalf("DriverName=%q, want postgres", got)
	}
	if !BootstrapSchemaEnabled() {
		t.Fatalf("BootstrapSchemaEnabled deberia estar activo para esta prueba")
	}

	for _, table := range []string{
		"agentes",
		"config",
		"reglas",
		"skills",
		"workflows",
		"proyectos",
		"sesiones",
		"runtime_handles",
		"runtime_orders",
		"pools_capacidad",
		"pool_modelos",
		"decisiones_proyecto",
		"documentos_externos",
		"git_merges",
	} {
		exists, err := TableExists(table)
		if err != nil {
			t.Fatalf("TableExists(%s): %v", table, err)
		}
		if !exists {
			t.Fatalf("tabla %s no creada por el bootstrap de postgres", table)
		}
	}

	for _, table := range []string{"reglas", "skills", "workflows"} {
		got, err := tableRowCount(table)
		if err != nil {
			t.Fatalf("tableRowCount(%s): %v", table, err)
		}
		if got == 0 {
			t.Fatalf("bootstrap postgres deberia dejar semillas minimas en %s", table)
		}
	}

	var version string
	if err := DB.QueryRow(`SELECT valor FROM config WHERE clave = ?`, "version").Scan(&version); err != nil {
		t.Fatalf("leyendo config semilla version: %v", err)
	}
	if version != "1.0.0" {
		t.Fatalf("valor version inesperado: %s", version)
	}
}

func tableRowCount(table string) (int64, error) {
	var count int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM ` + quotePostgresIdent(table)).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func seedRowCount(table string) (int, bool) {
	for _, group := range schemaSeedGroups {
		if group.table == table {
			return len(group.rows), true
		}
	}
	return 0, false
}

func quotePostgresIdent(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

func postgresTestDSNWithSearchPath(baseDSN, schema string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseDSN))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
