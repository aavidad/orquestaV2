package db

import (
	"os"
	"strings"
)

var (
	schemaBootstrapDDL = renderBootstrapSectionDDLForDriver("sqlite")
	schemaRuntimeDDL   = renderRuntimeSectionDDLForDriver("sqlite")
	schemaCapacityDDL  = renderCapacitySectionDDLForDriver("sqlite")
	schemaSeedGroups   = parseSchemaSeedGroups(schemaSeedDataForDriver("sqlite"))
)

type schemaSeedGroup struct {
	table string
	rows  [][]string
}

func BootstrapSchemaEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("ORQUESTA_DB_BOOTSTRAP")))
	return v == "1" || v == "true" || v == "yes"
}

func updatedAtTables() []string {
	return []string{
		"tareas",
		"propuestas",
		"votos",
		"proyectos",
		"asignaciones",
		"locks",
		"worktrees",
		"runtime_instances",
		"runtime_handles",
		"runtime_orders",
		"conectores",
		"git_merges",
		"pools_capacidad",
		"pool_modelos",
		"politicas_modelo",
		"decisiones_proyecto",
		"documentos_externos",
		"memoria_proyectos",
		"memoria_derivas",
		"fases_proyecto",
		"avance_tareas",
	}
}

func parseSchemaSeedGroups(seedSQL string) []schemaSeedGroup {
	statements := schemaStatements(seedSQL)
	groups := make([]schemaSeedGroup, 0, len(statements))
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if !strings.HasPrefix(strings.ToUpper(stmt), "INSERT INTO ") {
			continue
		}
		table := parseSeedTable(stmt)
		if table == "" {
			continue
		}
		count := countSeedRows(stmt)
		rows := make([][]string, count)
		for i := 0; i < count; i++ {
			rows[i] = []string{}
		}
		groups = append(groups, schemaSeedGroup{table: table, rows: rows})
	}
	return groups
}

func parseSeedTable(stmt string) string {
	upper := strings.ToUpper(stmt)
	prefix := "INSERT INTO "
	idx := strings.Index(upper, prefix)
	if idx < 0 {
		return ""
	}
	rest := stmt[idx+len(prefix):]
	end := strings.Index(rest, " ")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

func countSeedRows(stmt string) int {
	upper := strings.ToUpper(stmt)
	idx := strings.Index(upper, "VALUES")
	if idx < 0 {
		return 0
	}
	segment := stmt[idx+len("VALUES"):]
	inSingle := false
	count := 0
	depth := 0
	for i := 0; i < len(segment); i++ {
		ch := segment[i]
		if ch == '\'' {
			if inSingle && i+1 < len(segment) && segment[i+1] == '\'' {
				i++
				continue
			}
			inSingle = !inSingle
			continue
		}
		if inSingle {
			continue
		}
		switch ch {
		case '(':
			depth++
			if depth == 1 {
				count++
			}
		case ')':
			if depth > 0 {
				depth--
			}
		}
	}
	return count
}
