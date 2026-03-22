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
