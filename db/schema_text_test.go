/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

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
		"CREATE INDEX IF NOT EXISTS idx_asignaciones_agente_estado_proyecto_id",
		"CREATE INDEX IF NOT EXISTS idx_asignaciones_proyecto_estado_agente_id",
		"CREATE INDEX IF NOT EXISTS idx_sesiones_activa_id",
		"CREATE INDEX IF NOT EXISTS idx_sesiones_agente_activa_id",
		"CREATE INDEX IF NOT EXISTS idx_sesiones_agente_id",
		"CREATE TABLE IF NOT EXISTS conectores",
		"CREATE TABLE IF NOT EXISTS locks",
		"CREATE TABLE IF NOT EXISTS worktrees",
		"CREATE TABLE IF NOT EXISTS runtime_handles",
		"CREATE TABLE IF NOT EXISTS runtime_orders",
		"CREATE INDEX IF NOT EXISTS idx_runtime_mailbox_estado_id",
		"CREATE INDEX IF NOT EXISTS idx_tareas_agente_estado_id",
		"CREATE INDEX IF NOT EXISTS idx_tareas_estado_id",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_accion_id",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_agente_id",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_entidad_entidadid_id",
		"CREATE INDEX IF NOT EXISTS idx_runtime_instances_agente_updated_id",
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
	if strings.Contains(seeds, "INSERT OR IGNORE INTO agentes") {
		t.Fatalf("las semillas no deberian incluir agentes con INSERT OR IGNORE")
	}
	if strings.Contains(seeds, "INSERT OR IGNORE INTO config") {
		t.Fatalf("las semillas no deberian incluir config con INSERT OR IGNORE")
	}
	if strings.Contains(seeds, "INSERT OR IGNORE INTO reglas") {
		t.Fatalf("las semillas no deberian incluir reglas con INSERT OR IGNORE")
	}
	if strings.Contains(seeds, "INSERT OR IGNORE INTO skills") {
		t.Fatalf("las semillas no deberian incluir skills con INSERT OR IGNORE")
	}
	if strings.Contains(seeds, "INSERT OR IGNORE INTO workflows") {
		t.Fatalf("las semillas no deberian incluir workflows con INSERT OR IGNORE")
	}
	if strings.Contains(seeds, "INSERT OR IGNORE INTO") {
		t.Fatalf("las semillas no deberian usar INSERT OR IGNORE fijo")
	}
	if strings.Contains(seeds, "INSERT INTO agentes") {
		t.Fatalf("las semillas no deberian incluir agentes legacy")
	}
	if !strings.Contains(seeds, "INSERT INTO config") {
		t.Fatalf("las semillas deberian incluir config inicial")
	}
	if !strings.Contains(seeds, "ON CONFLICT(clave) DO NOTHING") {
		t.Fatalf("las semillas deberian usar conflicto por clave para config")
	}
	if !strings.Contains(seeds, "INSERT INTO reglas") {
		t.Fatalf("las semillas deberian incluir reglas iniciales")
	}
	if !strings.Contains(seeds, "ON CONFLICT(tipo_agente, titulo) DO NOTHING") {
		t.Fatalf("las semillas deberian usar conflicto compuesto para reglas")
	}
	if !strings.Contains(seeds, "INSERT INTO skills") {
		t.Fatalf("las semillas deberian incluir skills iniciales")
	}
	if !strings.Contains(seeds, "ON CONFLICT(tipo_agente, nombre) DO NOTHING") {
		t.Fatalf("las semillas deberian usar conflicto compuesto para skills")
	}
	if !strings.Contains(seeds, "INSERT INTO workflows") {
		t.Fatalf("las semillas deberian incluir workflows iniciales")
	}
	if !strings.Contains(seeds, "ON CONFLICT(nombre) DO NOTHING") {
		t.Fatalf("las semillas deberian usar conflicto por nombre para workflows")
	}
	if !strings.Contains(ddl, "CREATE TABLE IF NOT EXISTS reglas") {
		t.Fatalf("el DDL deberia incluir tablas de gobernanza")
	}
	for _, required := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_locks_scope_activo",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_handles_agente_activo",
		"CREATE INDEX IF NOT EXISTS idx_sesiones_activa_id",
		"CREATE INDEX IF NOT EXISTS idx_sesiones_agente_activa_id",
		"CREATE INDEX IF NOT EXISTS idx_sesiones_agente_id",
		"CREATE INDEX IF NOT EXISTS idx_presupuestos_sesion_sesion_checked_id",
		"CREATE INDEX IF NOT EXISTS idx_presupuestos_sesion_sesion_fuente_checked_id",
		"CREATE INDEX IF NOT EXISTS idx_propuestas_estado_proyecto_id",
		"CREATE INDEX IF NOT EXISTS idx_votos_agente_posicion_propuesta",
		"CREATE INDEX IF NOT EXISTS idx_tareas_agente_estado_id",
		"CREATE INDEX IF NOT EXISTS idx_tareas_estado_id",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_accion_id",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_agente_id",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_entidad_entidadid_id",
		"CREATE INDEX IF NOT EXISTS idx_runtime_instances_agente_updated_id",
	} {
		if !strings.Contains(ddl, required) {
			t.Fatalf("el DDL deberia incluir el indice auxiliar %q", required)
		}
	}
	if count := strings.Count(ddl, "CREATE TRIGGER IF NOT EXISTS trig_"); count != len(updatedAtTables()) {
		t.Fatalf("sqlite deberia declarar %d triggers updated_at; obtuvo %d", len(updatedAtTables()), count)
	}
}

func TestBootstrapSchemaSpecRenderPorDriver(t *testing.T) {
	t.Parallel()

	specs := bootstrapSchemaSpecs()
	if got := len(specs); got != 2 {
		t.Fatalf("bootstrapSchemaSpecs deberia tener 2 tablas; obtuvo %d", got)
	}
	if specs[0].Name != "config" {
		t.Fatalf("la primera tabla del bootstrap deberia ser config")
	}
	if specs[1].Name != "agentes" {
		t.Fatalf("la segunda tabla del bootstrap deberia ser agentes")
	}

	sqliteDDL := renderBootstrapSectionDDLForDriver("sqlite")
	postgresDDL := renderBootstrapSectionDDLForDriver("postgres")
	if sqliteDDL == "" || postgresDDL == "" {
		t.Fatalf("el bootstrap renderizado no deberia ser vacio")
	}
	if schemaBootstrapDDL != sqliteDDL {
		t.Fatalf("schemaBootstrapDDL deberia coincidir con el render sqlite del spec")
	}
	for _, ddl := range []string{sqliteDDL, postgresDDL} {
		for _, required := range []string{
			"CREATE TABLE IF NOT EXISTS config",
			"CREATE TABLE IF NOT EXISTS agentes",
			"CHECK (rol IN ('programador','documentador','admin'))",
			"DEFAULT NULL",
		} {
			if !strings.Contains(ddl, required) {
				t.Fatalf("bootstrap renderizado no contiene %q", required)
			}
		}
	}
	if !strings.Contains(sqliteDDL, "ultima_sesion DATETIME") {
		t.Fatalf("sqlite deberia conservar DATETIME en ultima_sesion")
	}
	if !strings.Contains(postgresDDL, "ultima_sesion TIMESTAMP") {
		t.Fatalf("postgres deberia usar TIMESTAMP en ultima_sesion")
	}
	if strings.Contains(postgresDDL, "DATETIME") {
		t.Fatalf("postgres no deberia conservar DATETIME en bootstrap")
	}
}

func TestRuntimeSchemaSpecRenderPorDriver(t *testing.T) {
	t.Parallel()

	specs := runtimeSchemaSpecs()
	if got := len(specs); got != 2 {
		t.Fatalf("runtimeSchemaSpecs deberia tener 2 tablas; obtuvo %d", got)
	}
	if specs[0].Name != "runtime_handles" {
		t.Fatalf("la primera tabla runtime deberia ser runtime_handles")
	}
	if specs[1].Name != "runtime_orders" {
		t.Fatalf("la segunda tabla runtime deberia ser runtime_orders")
	}

	sqliteDDL := renderRuntimeSectionDDLForDriver("sqlite")
	postgresDDL := renderRuntimeSectionDDLForDriver("postgres")
	if sqliteDDL == "" || postgresDDL == "" {
		t.Fatalf("el runtime renderizado no deberia ser vacio")
	}
	if schemaRuntimeDDL != sqliteDDL {
		t.Fatalf("schemaRuntimeDDL deberia coincidir con el render sqlite del spec")
	}
	for _, ddl := range []string{sqliteDDL, postgresDDL} {
		for _, required := range []string{
			"CREATE TABLE IF NOT EXISTS runtime_handles",
			"CREATE TABLE IF NOT EXISTS runtime_orders",
			"REFERENCES agentes(nombre)",
			"REFERENCES sesiones(id) ON DELETE SET NULL",
			"CHECK (estado IN ('activo','pausado','cerrado','fallido'))",
			"CHECK (tipo IN ('enviar_instruccion','pausar','continuar','handoff'))",
		} {
			if !strings.Contains(ddl, required) {
				t.Fatalf("runtime renderizado no contiene %q", required)
			}
		}
	}
	if !strings.Contains(sqliteDDL, "id INTEGER PRIMARY KEY AUTOINCREMENT") {
		t.Fatalf("sqlite deberia conservar AUTOINCREMENT en runtime")
	}
	if !strings.Contains(postgresDDL, "id INTEGER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY") {
		t.Fatalf("postgres deberia usar identity en runtime")
	}
	if strings.Contains(postgresDDL, "AUTOINCREMENT") {
		t.Fatalf("postgres no deberia conservar AUTOINCREMENT en runtime")
	}
	if !strings.Contains(postgresDDL, "created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP") {
		t.Fatalf("postgres deberia convertir DATETIME a TIMESTAMP en runtime")
	}
}

func TestCapacitySchemaSpecRenderPorDriver(t *testing.T) {
	t.Parallel()

	specs := capacitySchemaSpecs()
	if got := len(specs); got != 3 {
		t.Fatalf("capacitySchemaSpecs deberia tener 3 tablas; obtuvo %d", got)
	}
	if specs[0].Name != "pools_capacidad" {
		t.Fatalf("la primera tabla capacity deberia ser pools_capacidad")
	}
	if specs[1].Name != "pool_modelos" {
		t.Fatalf("la segunda tabla capacity deberia ser pool_modelos")
	}
	if specs[2].Name != "politicas_modelo" {
		t.Fatalf("la tercera tabla capacity deberia ser politicas_modelo")
	}

	sqliteDDL := renderCapacitySectionDDLForDriver("sqlite")
	postgresDDL := renderCapacitySectionDDLForDriver("postgres")
	if sqliteDDL == "" || postgresDDL == "" {
		t.Fatalf("el capacity renderizado no deberia ser vacio")
	}
	if schemaCapacityDDL != sqliteDDL {
		t.Fatalf("schemaCapacityDDL deberia coincidir con el render sqlite del spec")
	}
	for _, ddl := range []string{sqliteDDL, postgresDDL} {
		for _, required := range []string{
			"CREATE TABLE IF NOT EXISTS pools_capacidad",
			"CREATE TABLE IF NOT EXISTS pool_modelos",
			"CREATE TABLE IF NOT EXISTS politicas_modelo",
			"slug TEXT NOT NULL UNIQUE",
			"REFERENCES pools_capacidad(id) ON DELETE CASCADE",
			"UNIQUE(pool_id, model_slug)",
			"coste_relativo REAL NOT NULL DEFAULT 1.0",
			"CHECK (scope_tipo IN ('global','perfil','proyecto','fase','tarea'))",
		} {
			if !strings.Contains(ddl, required) {
				t.Fatalf("capacity renderizado no contiene %q", required)
			}
		}
	}
	if !strings.Contains(sqliteDDL, "id INTEGER PRIMARY KEY AUTOINCREMENT") {
		t.Fatalf("sqlite deberia conservar AUTOINCREMENT en capacity")
	}
	if !strings.Contains(postgresDDL, "id INTEGER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY") {
		t.Fatalf("postgres deberia usar identity en capacity")
	}
	if strings.Contains(postgresDDL, "AUTOINCREMENT") {
		t.Fatalf("postgres no deberia conservar AUTOINCREMENT en capacity")
	}
	if !strings.Contains(postgresDDL, "updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP") {
		t.Fatalf("postgres deberia convertir DATETIME a TIMESTAMP en capacity")
	}
}

func TestSchemaSeedDataForDriverMySQLUsaInsertIgnore(t *testing.T) {
	t.Parallel()

	seeds := schemaSeedDataForDriver("mysql")
	if seeds == "" {
		t.Fatalf("schema seeds vacio para mysql")
	}
	if !strings.Contains(seeds, "INSERT IGNORE INTO config") {
		t.Fatalf("mysql deberia usar INSERT IGNORE para config")
	}
	if strings.Contains(seeds, "INSERT IGNORE INTO agentes") {
		t.Fatalf("mysql no deberia sembrar agentes legacy")
	}
	if strings.Contains(seeds, "ON CONFLICT(") {
		t.Fatalf("mysql no deberia renderizar ON CONFLICT")
	}
}

func TestSchemaDDLForDriverPostgresReduceSintaxisSQLite(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"postgres", "postgresql"} {
		ddl := schemaDDLForDriver(driver)
		if ddl == "" {
			t.Fatalf("schema DDL vacio para %s", driver)
		}
		if strings.Contains(ddl, "AUTOINCREMENT") {
			t.Fatalf("%s no deberia conservar AUTOINCREMENT", driver)
		}
		if strings.Contains(ddl, "CREATE TRIGGER IF NOT EXISTS") {
			t.Fatalf("%s no deberia conservar triggers SQLite en este render minimo", driver)
		}
		if !strings.Contains(ddl, "GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY") {
			t.Fatalf("%s deberia usar columnas identity", driver)
		}
		if strings.Contains(ddl, " DATETIME") {
			t.Fatalf("%s no deberia conservar DATETIME", driver)
		}
		if !strings.Contains(ddl, " TIMESTAMP") {
			t.Fatalf("%s deberia convertir DATETIME a TIMESTAMP", driver)
		}
		if !strings.Contains(ddl, "CREATE TABLE IF NOT EXISTS proyectos") {
			t.Fatalf("%s deberia conservar las tablas base", driver)
		}
		if !strings.Contains(ddl, "CREATE OR REPLACE FUNCTION orquesta_set_updated_at()") {
			t.Fatalf("%s deberia declarar la funcion de updated_at", driver)
		}
		if count := strings.Count(ddl, "EXECUTE FUNCTION orquesta_set_updated_at()"); count != len(updatedAtTables()) {
			t.Fatalf("%s deberia declarar %d triggers updated_at; obtuvo %d", driver, len(updatedAtTables()), count)
		}
		if !strings.Contains(ddl, "BEFORE UPDATE ON tareas") {
			t.Fatalf("%s deberia recrear trigger updated_at para tareas", driver)
		}
		if strings.Contains(ddl, "UPDATE tareas SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;") {
			t.Fatalf("%s no deberia conservar el cuerpo de trigger SQLite", driver)
		}
	}
}

func TestSchemaBaseDDLNoIncluyeDDLAuxiliarEmbebido(t *testing.T) {
	t.Parallel()

	base := schemaBaseDDL
	if base == "" {
		t.Fatalf("schema base vacio")
	}
	if strings.Contains(base, "CREATE TRIGGER IF NOT EXISTS trig_tareas_updated") {
		t.Fatalf("el schema base no deberia conservar triggers updated_at embebidos")
	}
	if !strings.Contains(base, "CREATE TABLE IF NOT EXISTS reglas") {
		t.Fatalf("el schema base deberia conservar el resto del DDL")
	}
	if strings.Contains(base, "CREATE UNIQUE INDEX IF NOT EXISTS idx_locks_scope_activo") {
		t.Fatalf("el schema base no deberia conservar indices auxiliares embebidos")
	}
}

func TestSchemaBaseDDLComponeSeccionesSemanticas(t *testing.T) {
	t.Parallel()

	sections := []struct {
		name string
		ddl  string
		want string
	}{
		{name: "bootstrap", ddl: schemaBootstrapDDL, want: "CREATE TABLE IF NOT EXISTS agentes"},
		{name: "workflow", ddl: schemaWorkflowDDL, want: "CREATE TABLE IF NOT EXISTS sesiones"},
		{name: "coordination", ddl: schemaCoordinationDDL, want: "CREATE TABLE IF NOT EXISTS worktrees"},
		{name: "runtime", ddl: schemaRuntimeDDL, want: "CREATE TABLE IF NOT EXISTS runtime_orders"},
		{name: "capacity", ddl: schemaCapacityDDL, want: "CREATE TABLE IF NOT EXISTS politicas_modelo"},
		{name: "knowledge", ddl: schemaKnowledgeDDL, want: "CREATE TABLE IF NOT EXISTS workflows"},
	}

	for _, section := range sections {
		if strings.TrimSpace(section.ddl) == "" {
			t.Fatalf("seccion %s vacia", section.name)
		}
		if !strings.Contains(section.ddl, section.want) {
			t.Fatalf("seccion %s no contiene %q", section.name, section.want)
		}
	}

	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS agentes",
		"CREATE TABLE IF NOT EXISTS sesiones",
		"CREATE TABLE IF NOT EXISTS worktrees",
		"CREATE TABLE IF NOT EXISTS runtime_orders",
		"CREATE TABLE IF NOT EXISTS politicas_modelo",
		"CREATE TABLE IF NOT EXISTS workflows",
	} {
		if !strings.Contains(schemaBaseDDL, required) {
			t.Fatalf("schemaBaseDDL no contiene %q", required)
		}
	}
}

func TestSchemaBaseSectionsMantieneOrdenEsperado(t *testing.T) {
	t.Parallel()

	sections := schemaBaseSections()
	if len(sections) != 6 {
		t.Fatalf("schemaBaseSections deberia exponer 6 secciones; obtuvo %d", len(sections))
	}
	if sections[0] != schemaBootstrapDDL {
		t.Fatalf("la primera seccion deberia ser bootstrap")
	}
	if sections[1] != schemaWorkflowDDL {
		t.Fatalf("la segunda seccion deberia ser workflow")
	}
	if sections[5] != schemaKnowledgeDDL {
		t.Fatalf("la ultima seccion deberia ser knowledge")
	}
}

func TestSchemaBaseSectionRenderersMantieneContratoBase(t *testing.T) {
	t.Parallel()

	renderers := schemaBaseSectionRenderers()
	if len(renderers) != 6 {
		t.Fatalf("schemaBaseSectionRenderers deberia exponer 6 secciones; obtuvo %d", len(renderers))
	}
	if renderers[0].sqliteDDL != schemaBootstrapDDL {
		t.Fatalf("la primera seccion renderizada deberia ser bootstrap")
	}
	if got := renderers[0].renderFor("postgres"); !strings.Contains(got, "ultima_sesion TIMESTAMP") {
		t.Fatalf("bootstrap postgres deberia salir del spec renderizado; obtuvo: %s", got)
	}
	if got := renderers[1].renderFor("postgres"); !strings.Contains(got, "CREATE TABLE IF NOT EXISTS tareas") {
		t.Fatalf("workflow postgres deberia conservar tareas")
	}
	if got := renderers[3].renderFor("postgres"); !strings.Contains(got, "GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY") {
		t.Fatalf("runtime postgres deberia salir del spec renderizado; obtuvo: %s", got)
	}
	if got := renderers[4].renderFor("postgres"); !strings.Contains(got, "UNIQUE(pool_id, model_slug)") {
		t.Fatalf("capacity postgres deberia salir del spec renderizado; obtuvo: %s", got)
	}
}

func TestBootstrapPlanForDriver(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"sqlite", "sqlite3", "postgres", "postgresql", "mysql"} {
		plan, ok := bootstrapPlanForDriver(driver)
		if !ok {
			t.Fatalf("%s deberia tener bootstrap plan", driver)
		}
		if strings.TrimSpace(plan.DDL) == "" {
			t.Fatalf("%s deberia tener DDL", driver)
		}
		if strings.TrimSpace(plan.Seed) == "" {
			t.Fatalf("%s deberia tener semillas", driver)
		}
	}
}

func TestSchemaDDLPartsForDriver(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"sqlite", "postgres"} {
		parts := schemaDDLPartsForDriver(driver)
		if len(parts) != 2 {
			t.Fatalf("%s deberia exponer 2 partes de DDL; obtuvo %d", driver, len(parts))
		}
		if strings.TrimSpace(parts[0]) == "" {
			t.Fatalf("%s deberia tener DDL base", driver)
		}
		if strings.TrimSpace(parts[1]) == "" {
			t.Fatalf("%s deberia tener DDL auxiliar", driver)
		}
	}
}

func TestRenderDriverColumnSyntaxPostgres(t *testing.T) {
	t.Parallel()

	got := renderDriverColumnSyntax("postgres", "id INTEGER PRIMARY KEY AUTOINCREMENT,\ncreated_at DATETIME NOT NULL")
	if strings.Contains(got, "AUTOINCREMENT") {
		t.Fatalf("postgres no deberia conservar AUTOINCREMENT: %s", got)
	}
	if strings.Contains(got, " DATETIME") {
		t.Fatalf("postgres no deberia conservar DATETIME: %s", got)
	}
	if !strings.Contains(got, "GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY") {
		t.Fatalf("postgres deberia convertir la PK identity: %s", got)
	}
	if !strings.Contains(got, " TIMESTAMP NOT NULL") {
		t.Fatalf("postgres deberia convertir DATETIME a TIMESTAMP: %s", got)
	}
}

func TestRenderBaseDDLForDriverComponeSecciones(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"sqlite", "postgres"} {
		ddl := renderBaseDDLForDriver(driver)
		if strings.TrimSpace(ddl) == "" {
			t.Fatalf("%s deberia producir DDL base", driver)
		}
		for _, required := range []string{
			"CREATE TABLE IF NOT EXISTS agentes",
			"CREATE TABLE IF NOT EXISTS sesiones",
			"CREATE TABLE IF NOT EXISTS worktrees",
			"CREATE TABLE IF NOT EXISTS runtime_orders",
			"CREATE TABLE IF NOT EXISTS politicas_modelo",
			"CREATE TABLE IF NOT EXISTS workflows",
		} {
			if !strings.Contains(ddl, required) {
				t.Fatalf("%s no contiene %q", driver, required)
			}
		}
	}
}
