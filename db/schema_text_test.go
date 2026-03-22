package db

import (
	"strings"
	"testing"
)

func TestSchemaNoIncluyeCoordinacionPorFicherosObsoletos(t *testing.T) {
	t.Parallel()

	for _, prohibido := range []string{
		"Opinion.md",
		"Dudas.md",
		"ContaGrx/orquestacion.md",
		"en Opinion.md o en la BD",
		"PRAGMA journal_mode",
		"PRAGMA foreign_keys",
	} {
		if strings.Contains(Schema, prohibido) {
			t.Fatalf("Schema contiene referencia obsoleta: %q", prohibido)
		}
	}
}

func TestSchemaIncluyeCoordinacionMultiProyectoYMCP(t *testing.T) {
	t.Parallel()

	for _, requerido := range []string{
		"CREATE TABLE IF NOT EXISTS proyectos",
		"CREATE TABLE IF NOT EXISTS asignaciones",
		"CREATE TABLE IF NOT EXISTS conectores",
		"CREATE TABLE IF NOT EXISTS locks",
		"CREATE TABLE IF NOT EXISTS worktrees",
		"CREATE TABLE IF NOT EXISTS runtime_handles",
		"CREATE TABLE IF NOT EXISTS runtime_orders",
		"CREATE TABLE IF NOT EXISTS pools_capacidad",
		"CREATE TABLE IF NOT EXISTS pool_modelos",
		"CREATE TABLE IF NOT EXISTS politicas_modelo",
		"CREATE TABLE IF NOT EXISTS decisiones_proyecto",
		"CREATE TABLE IF NOT EXISTS documentos_externos",
		"CREATE TABLE IF NOT EXISTS git_merges",
		"external_session_id",
		"resumen_continuidad",
		"pool_id             INTEGER REFERENCES pools_capacidad(id)",
		"reasoning_effort TEXT NOT NULL DEFAULT ''",
	} {
		if !strings.Contains(Schema, requerido) {
			t.Fatalf("Schema no contiene el bloque esperado: %q", requerido)
		}
	}
}

func TestSchemaSeparadoEnDDLYSemillas(t *testing.T) {
	t.Parallel()

	ddl := schemaDDL()
	seeds := schemaSeedData()
	if ddl == "" {
		t.Fatalf("schema DDL vacio")
	}
	if seeds == "" {
		t.Fatalf("schema seeds vacio")
	}
	if strings.Contains(ddl, "INSERT OR IGNORE INTO agentes") {
		t.Fatalf("el DDL no deberia incluir semillas de agentes")
	}
	if strings.Contains(ddl, "INSERT OR IGNORE INTO config") {
		t.Fatalf("el DDL no deberia incluir semillas de config")
	}
	if strings.Contains(ddl, "INSERT OR IGNORE INTO reglas") {
		t.Fatalf("el DDL no deberia incluir semillas de reglas")
	}
	if !strings.Contains(seeds, "INSERT OR IGNORE INTO agentes") {
		t.Fatalf("las semillas deberian incluir agentes iniciales")
	}
	if !strings.Contains(seeds, "INSERT OR IGNORE INTO config") {
		t.Fatalf("las semillas deberian incluir config inicial")
	}
	if !strings.Contains(seeds, "INSERT OR IGNORE INTO reglas") {
		t.Fatalf("las semillas deberian incluir reglas iniciales")
	}
	if !strings.Contains(ddl, "CREATE TABLE IF NOT EXISTS reglas") {
		t.Fatalf("el DDL deberia incluir tablas de gobernanza")
	}
}
