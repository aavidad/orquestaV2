package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostMigrationStatementsSoloMigracionesIncrementales(t *testing.T) {
	t.Parallel()

	statements := postMigrationStatements()
	if len(statements) == 0 {
		t.Fatalf("postMigrationStatements vacio")
	}

	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "CREATE TABLE") ||
			strings.HasPrefix(upper, "CREATE TRIGGER") ||
			strings.HasPrefix(upper, "CREATE UNIQUE INDEX") {
			t.Fatalf("postMigrationStatements no debe incluir DDL de esquema: %q", trimmed)
		}
		if !strings.HasPrefix(upper, "ALTER TABLE ") && !strings.HasPrefix(upper, "UPDATE ") {
			t.Fatalf("postMigrationStatements contiene sentencia inesperada: %q", trimmed)
		}
	}

	for _, required := range []string{
		"ALTER TABLE agentes ADD COLUMN estado_sesion TEXT DEFAULT NULL",
		"ALTER TABLE sesiones ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)",
		"UPDATE reglas",
		"UPDATE workflows",
	} {
		found := false
		for _, stmt := range statements {
			if strings.Contains(stmt, required) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no se encontró la migración esperada: %q", required)
		}
	}
}

func TestOpenBootstrapCreaTriggersEIndicesDesdeSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	for _, obj := range []struct {
		typ  string
		name string
	}{
		{typ: "trigger", name: "trig_proyectos_updated"},
		{typ: "trigger", name: "trig_asignaciones_updated"},
		{typ: "trigger", name: "trig_conectores_updated"},
		{typ: "trigger", name: "trig_locks_updated"},
		{typ: "trigger", name: "trig_worktrees_updated"},
		{typ: "trigger", name: "trig_git_merges_updated"},
		{typ: "trigger", name: "trig_runtime_handles_updated"},
		{typ: "trigger", name: "trig_pools_capacidad_updated"},
		{typ: "trigger", name: "trig_pool_modelos_updated"},
		{typ: "trigger", name: "trig_politicas_modelo_updated"},
		{typ: "trigger", name: "trig_decisiones_proyecto_updated"},
		{typ: "trigger", name: "trig_documentos_externos_updated"},
		{typ: "index", name: "idx_locks_scope_activo"},
		{typ: "index", name: "idx_runtime_handles_agente_activo"},
	} {
		exists, err := SchemaObjectExists(obj.typ, obj.name)
		if err != nil {
			t.Fatalf("consultando %s %s: %v", obj.typ, obj.name, err)
		}
		if !exists {
			t.Fatalf("%s %s no existe tras bootstrap", obj.typ, obj.name)
		}
	}
}

func TestPostMigracionesLimpianCerradaAtEnPropuestasAbiertas(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open inicial: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO propuestas (codigo, titulo, descripcion, tipo, estado, propuesto_por, distribuidor, cerrada_at)
		VALUES ('OP-996', 'Inconsistencia heredada', '', 'arquitectura', 'abierta', 'Codex1', 'Codex1', CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert propuesta inconsistente: %v", err)
	}

	var antes int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM propuestas WHERE codigo='OP-996' AND cerrada_at IS NOT NULL`).Scan(&antes); err != nil {
		t.Fatalf("count antes: %v", err)
	}
	if antes != 1 {
		t.Fatalf("esperaba propuesta inconsistente antes de reabrir, got=%d", antes)
	}

	Close()
	if err := Open(); err != nil {
		t.Fatalf("Open segunda: %v", err)
	}

	var despues int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM propuestas WHERE codigo='OP-996' AND cerrada_at IS NULL`).Scan(&despues); err != nil {
		t.Fatalf("count despues: %v", err)
	}
	if despues != 1 {
		t.Fatalf("post-migraciones no limpiaron cerrada_at en propuesta abierta, got=%d", despues)
	}
}
