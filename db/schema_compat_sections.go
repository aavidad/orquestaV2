package db

import "strings"

var (
	schemaWorkflowDDL     = renderWorkflowDDLForDriver("sqlite")
	schemaCoordinationDDL = renderCoordinationDDLForDriver("sqlite")
	schemaKnowledgeDDL    = renderKnowledgeDDLForDriver("sqlite")
	schemaBaseDDL         = renderBaseDDLForDriver("sqlite")
)

type schemaBaseSectionRenderer struct {
	sqliteDDL string
	renderFn  func(driver string) string
}

func (r schemaBaseSectionRenderer) renderFor(driver string) string {
	return r.renderFn(driver)
}

func schemaBaseSections() []string {
	return []string{
		schemaBootstrapDDL,
		schemaWorkflowDDL,
		schemaCoordinationDDL,
		schemaRuntimeDDL,
		schemaCapacityDDL,
		schemaKnowledgeDDL,
	}
}

func schemaBaseSectionRenderers() []schemaBaseSectionRenderer {
	return []schemaBaseSectionRenderer{
		{sqliteDDL: schemaBootstrapDDL, renderFn: renderBootstrapSectionDDLForDriver},
		{sqliteDDL: schemaWorkflowDDL, renderFn: renderWorkflowDDLForDriver},
		{sqliteDDL: schemaCoordinationDDL, renderFn: renderCoordinationDDLForDriver},
		{sqliteDDL: schemaRuntimeDDL, renderFn: renderRuntimeSectionDDLForDriver},
		{sqliteDDL: schemaCapacityDDL, renderFn: renderCapacitySectionDDLForDriver},
		{sqliteDDL: schemaKnowledgeDDL, renderFn: renderKnowledgeDDLForDriver},
	}
}

func schemaDDLPartsForDriver(driver string) []string {
	return []string{
		renderBaseDDLForDriver(driver),
		renderAuxDDLForDriver(driver),
	}
}

func renderBaseDDLForDriver(driver string) string {
	return joinDDLParts(
		renderBootstrapSectionDDLForDriver(driver),
		renderWorkflowDDLForDriver(driver),
		renderCoordinationDDLForDriver(driver),
		renderRuntimeSectionDDLForDriver(driver),
		renderCapacitySectionDDLForDriver(driver),
		renderKnowledgeDDLForDriver(driver),
	)
}

func renderAuxDDLForDriver(driver string) string {
	driver = normalizedDriverName(driver)
	if driver == "postgres" || driver == "postgresql" {
		parts := []string{`CREATE OR REPLACE FUNCTION orquesta_set_updated_at() RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = CURRENT_TIMESTAMP;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;`}
		for _, table := range updatedAtTables() {
			parts = append(parts, "CREATE TRIGGER trig_"+table+"_updated BEFORE UPDATE ON "+table+" FOR EACH ROW EXECUTE FUNCTION orquesta_set_updated_at();")
		}
		parts = append(parts,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_locks_scope_activo ON locks(scope_type, scope_key) WHERE estado = 'activa';`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_handles_agente_activo ON runtime_handles(agente, estado, id);`,
		)
		return joinDDLParts(parts...)
	}

	return joinDDLParts(
		extractSchemaObjects(
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_locks_scope_activo",
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_handles_agente_activo",
			"CREATE TRIGGER IF NOT EXISTS trig_tareas_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_propuestas_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_votos_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_proyectos_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_asignaciones_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_locks_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_worktrees_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_runtime_instances_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_runtime_handles_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_runtime_orders_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_conectores_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_git_merges_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_pools_capacidad_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_pool_modelos_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_politicas_modelo_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_decisiones_proyecto_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_documentos_externos_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_memoria_proyectos_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_memoria_derivas_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_fases_proyecto_updated",
			"CREATE TRIGGER IF NOT EXISTS trig_avance_tareas_updated",
		)...,
	)
}

func renderWorkflowDDLForDriver(driver string) string {
	return renderSchemaTablesForDriver(driver, "tareas", "propuestas", "votos", "bloqueos", "sesiones")
}

func renderCoordinationDDLForDriver(driver string) string {
	return renderSchemaTablesForDriver(driver, "proyectos", "asignaciones", "locks", "worktrees")
}

func renderKnowledgeDDLForDriver(driver string) string {
	return renderSchemaTablesForDriver(driver, "reglas", "skills", "workflows")
}

func renderSchemaTablesForDriver(driver string, tables ...string) string {
	statements := schemaStatements(Schema)
	selected := make([]string, 0, len(tables))
	for _, stmt := range statements {
		upper := strings.ToUpper(stripLeadingSQLComments(stmt))
		if !strings.HasPrefix(upper, "CREATE TABLE IF NOT EXISTS ") {
			continue
		}
		for _, table := range tables {
			if strings.Contains(stmt, "CREATE TABLE IF NOT EXISTS "+table+" ") {
				selected = append(selected, renderDriverColumnSyntax(driver, stmt))
				break
			}
		}
	}
	return joinDDLParts(selected...)
}

func extractSchemaObjects(prefixes ...string) []string {
	statements := schemaStatements(Schema)
	var out []string
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		for _, prefix := range prefixes {
			if strings.HasPrefix(trimmed, prefix) {
				out = append(out, trimmed)
				break
			}
		}
	}
	return out
}

func renderDriverColumnSyntax(driver, ddl string) string {
	driver = normalizedDriverName(driver)
	if driver != "postgres" && driver != "postgresql" {
		return ddl
	}
	replacer := strings.NewReplacer(
		"INTEGER PRIMARY KEY AUTOINCREMENT", "INTEGER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY",
		" DATETIME", " TIMESTAMP",
	)
	return replacer.Replace(ddl)
}
