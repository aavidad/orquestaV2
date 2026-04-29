package db

import (
	"fmt"
	"regexp"
	"strings"
)

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
	driver = normalizedDriverName(driver)
	if driver == "postgres" || driver == "postgresql" {
		return renderPostgresBaseDDLFromLegacySchema()
	}
	return joinDDLParts(
		renderBootstrapSectionDDLForDriver(driver),
		renderWorkflowDDLForDriver(driver),
		renderCoordinationDDLForDriver(driver),
		renderRuntimeSectionDDLForDriver(driver),
		renderCapacitySectionDDLForDriver(driver),
		renderKnowledgeDDLForDriver(driver),
	)
}

var postgresInlineReferencePattern = regexp.MustCompile(`\s+REFERENCES\s+([A-Za-z0-9_]+)\(([^)]+)\)(?:\s+ON\s+DELETE\s+([A-Z\s]+))?`)

func renderPostgresBaseDDLFromLegacySchema() string {
	statements := schemaStatements(Schema)
	createTables := make([]string, 0, len(statements))
	constraints := make([]string, 0, len(statements))
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		upper := strings.ToUpper(stripLeadingSQLComments(trimmed))
		if !strings.HasPrefix(upper, "CREATE TABLE IF NOT EXISTS ") {
			continue
		}
		createStmt, alters := splitPostgresCreateTableReferences(trimmed)
		createTables = append(createTables, createStmt)
		constraints = append(constraints, alters...)
	}
	return joinDDLParts(append(createTables, constraints...)...)
}

func splitPostgresCreateTableReferences(stmt string) (string, []string) {
	stmt = stripLeadingSQLComments(stmt)
	stmt = renderDriverColumnSyntax("postgres", stmt)
	lines := strings.Split(stmt, "\n")
	if len(lines) == 0 {
		return stmt, nil
	}
	table := parseCreateTableName(lines[0])
	if table == "" {
		return stmt, nil
	}
	alters := make([]string, 0, 4)
	for i, line := range lines {
		if !strings.Contains(line, " REFERENCES ") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "UNIQUE(") || strings.HasPrefix(trimmed, "CHECK (") {
			continue
		}
		matches := postgresInlineReferencePattern.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		column := parseColumnNameFromLine(line)
		if column == "" {
			continue
		}
		refTable := strings.TrimSpace(matches[1])
		refColumn := strings.TrimSpace(matches[2])
		onDelete := strings.TrimSpace(matches[3])
		lines[i] = postgresInlineReferencePattern.ReplaceAllString(line, "")
		lines[i] = strings.ReplaceAll(lines[i], " ,", ",")
		lines[i] = strings.TrimRight(lines[i], " ")
		alters = append(alters, postgresForeignKeyAlter(table, column, refTable, refColumn, onDelete))
	}
	return strings.Join(lines, "\n"), alters
}

func parseCreateTableName(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "CREATE TABLE IF NOT EXISTS ")
	line = strings.TrimSpace(strings.TrimSuffix(line, "("))
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSpace(fields[0])
}

func parseColumnNameFromLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSpace(fields[0]), ",")
}

func postgresForeignKeyAlter(table, column, refTable, refColumn, onDelete string) string {
	name := fmt.Sprintf("fk_%s_%s", table, column)
	stmt := fmt.Sprintf(
		"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
		table,
		name,
		column,
		refTable,
		refColumn,
	)
	if onDelete != "" {
		stmt += " ON DELETE " + onDelete
	}
	return postgresEnsureConstraintBlock(name, stmt)
}

func postgresEnsureConstraintBlock(name, stmt string) string {
	return fmt.Sprintf(`DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = '%s') THEN
		%s;
	END IF;
END
$$;`, postgresQuoteLiteral(name), stmt)
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
			parts = append(parts, postgresEnsureUpdatedAtTrigger(table))
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

func postgresEnsureUpdatedAtTrigger(table string) string {
	name := "trig_" + table + "_updated"
	return fmt.Sprintf(`DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = '%s' AND NOT tgisinternal) THEN
		CREATE TRIGGER %s BEFORE UPDATE ON %s FOR EACH ROW EXECUTE FUNCTION orquesta_set_updated_at();
	END IF;
END
$$;`, postgresQuoteLiteral(name), name, table)
}

func postgresQuoteLiteral(value string) string {
	return strings.ReplaceAll(value, `'`, `''`)
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
		canonical := strings.TrimSpace(stripLeadingSQLComments(trimmed))
		for _, prefix := range prefixes {
			if strings.HasPrefix(canonical, prefix) {
				out = append(out, canonical)
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
