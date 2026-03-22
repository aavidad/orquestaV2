package db

import (
	"database/sql"
	"fmt"
	"strings"
)

type schemaSeedGroup struct {
	table   string
	columns []string
	rows    [][]string
}

var schemaSeedGroups = []schemaSeedGroup{
	{
		table:   "agentes",
		columns: []string{"nombre", "rol"},
		rows: [][]string{
			{"alberto", "admin"},
			{"claude", "programador"},
			{"codex1", "programador"},
			{"codex2", "programador"},
			{"antigravity", "documentador"},
		},
	},
	{
		table:   "config",
		columns: []string{"clave", "valor"},
		rows: [][]string{
			{"distribuidor", "claude"},
			{"version", "1.0.0"},
			{"pool_handoff_threshold_seconds", "1800"},
			{"pool_handoff_threshold_ratio", "0.10"},
			{"pool_default_budget_source", "manual"},
			{"model_policy_default_profile", "implementacion"},
			{"model_policy_default_reasoning", "high"},
		},
	},
}

func schemaDDL() string {
	return strings.TrimSpace(Schema)
}

func schemaSeedData() string {
	var b strings.Builder
	for _, group := range schemaSeedGroups {
		b.WriteString(renderSchemaSeedGroup(group))
	}
	return strings.TrimSpace(b.String())
}

func renderSchemaSeedGroup(group schemaSeedGroup) string {
	if len(group.columns) == 0 || len(group.rows) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("INSERT OR IGNORE INTO ")
	b.WriteString(group.table)
	b.WriteString(" (")
	b.WriteString(strings.Join(group.columns, ", "))
	b.WriteString(") VALUES\n")
	for i, row := range group.rows {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("    (")
		for j, value := range row {
			if j > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteSchemaSeedValue(value))
		}
		b.WriteString(")")
	}
	b.WriteString(";\n\n")
	return b.String()
}

func quoteSchemaSeedValue(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func aplicarSchema(db *sql.DB) error {
	ddl := schemaDDL()
	if ddl == "" {
		return fmt.Errorf("schema DDL vacio")
	}
	_, err := db.Exec(ddl)
	return err
}

func aplicarSemillasSchema(db *sql.DB) error {
	seeds := schemaSeedData()
	if seeds == "" {
		return nil
	}
	_, err := db.Exec(seeds)
	return err
}
