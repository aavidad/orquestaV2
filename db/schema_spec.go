package db

import "strings"

// schema spec mínimo para secciones que ya queremos renderizar por backend sin
// depender de reemplazos de texto sobre un DDL SQLite-first.
type schemaColumnType string

const (
	schemaColumnTypeText     schemaColumnType = "text"
	schemaColumnTypeInteger  schemaColumnType = "integer"
	schemaColumnTypeDateTime schemaColumnType = "datetime"
)

type schemaColumnSpec struct {
	Name       string
	Type       schemaColumnType
	NotNull    bool
	DefaultSQL string
	PrimaryKey bool
	CheckSQL   string
}

type schemaTableSpec struct {
	Name    string
	Columns []schemaColumnSpec
}

func bootstrapSchemaSpecs() []schemaTableSpec {
	return []schemaTableSpec{
		{
			Name: "config",
			Columns: []schemaColumnSpec{
				{Name: "clave", Type: schemaColumnTypeText, PrimaryKey: true},
				{Name: "valor", Type: schemaColumnTypeText, NotNull: true},
			},
		},
		{
			Name: "agentes",
			Columns: []schemaColumnSpec{
				{Name: "nombre", Type: schemaColumnTypeText, PrimaryKey: true},
				{Name: "rol", Type: schemaColumnTypeText, NotNull: true, CheckSQL: "rol IN ('programador','documentador','admin')"},
				{Name: "activo", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "0"},
				{Name: "habilitado", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "1"},
				{Name: "estado_sesion", Type: schemaColumnTypeText, DefaultSQL: "NULL"},
				{Name: "ultima_sesion", Type: schemaColumnTypeDateTime},
			},
		},
	}
}

func renderBootstrapSectionDDLForDriver(driver string) string {
	parts := make([]string, 0, len(bootstrapSchemaSpecs()))
	for _, table := range bootstrapSchemaSpecs() {
		parts = append(parts, renderTableSpecForDriver(driver, table))
	}
	return joinDDLParts(parts...)
}

func renderTableSpecForDriver(driver string, table schemaTableSpec) string {
	var b strings.Builder
	b.WriteString("CREATE TABLE IF NOT EXISTS ")
	b.WriteString(table.Name)
	b.WriteString(" (\n")
	for i, column := range table.Columns {
		b.WriteString("    ")
		b.WriteString(renderColumnSpecForDriver(driver, column))
		if i < len(table.Columns)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(");")
	return b.String()
}

func renderColumnSpecForDriver(driver string, column schemaColumnSpec) string {
	parts := []string{column.Name, renderColumnTypeForDriver(driver, column.Type)}
	if column.NotNull {
		parts = append(parts, "NOT NULL")
	}
	if column.DefaultSQL != "" {
		parts = append(parts, "DEFAULT "+column.DefaultSQL)
	}
	if column.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if column.CheckSQL != "" {
		parts = append(parts, "CHECK ("+column.CheckSQL+")")
	}
	return strings.Join(parts, " ")
}

func renderColumnTypeForDriver(driver string, columnType schemaColumnType) string {
	switch columnType {
	case schemaColumnTypeInteger:
		return "INTEGER"
	case schemaColumnTypeDateTime:
		if normalizedDriverName(driver) == "postgres" {
			return "TIMESTAMP"
		}
		return "DATETIME"
	default:
		return "TEXT"
	}
}
