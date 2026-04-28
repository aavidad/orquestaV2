package db

import "strings"

// schema spec mínimo para secciones que ya queremos renderizar por backend sin
// depender de reemplazos de texto sobre un DDL base del dialecto heredado.
type schemaColumnType string

const (
	schemaColumnTypeText     schemaColumnType = "text"
	schemaColumnTypeInteger  schemaColumnType = "integer"
	schemaColumnTypeDateTime schemaColumnType = "datetime"
	schemaColumnTypeReal     schemaColumnType = "real"
)

type schemaReferenceSpec struct {
	Table    string
	Column   string
	OnDelete string
}

type schemaColumnSpec struct {
	Name          string
	Type          schemaColumnType
	NotNull       bool
	DefaultSQL    string
	PrimaryKey    bool
	AutoIncrement bool
	Unique        bool
	CheckSQL      string
	References    *schemaReferenceSpec
}

type schemaUniqueConstraintSpec struct {
	Columns []string
}

type schemaTableSpec struct {
	Name              string
	Columns           []schemaColumnSpec
	UniqueConstraints []schemaUniqueConstraintSpec
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

func runtimeSchemaSpecs() []schemaTableSpec {
	return []schemaTableSpec{
		{
			Name: "runtime_handles",
			Columns: []schemaColumnSpec{
				{Name: "id", Type: schemaColumnTypeInteger, PrimaryKey: true, AutoIncrement: true},
				{Name: "agente", Type: schemaColumnTypeText, NotNull: true, References: &schemaReferenceSpec{Table: "agentes", Column: "nombre"}},
				{Name: "sesion_id", Type: schemaColumnTypeInteger, References: &schemaReferenceSpec{Table: "sesiones", Column: "id", OnDelete: "SET NULL"}},
				{Name: "proyecto_id", Type: schemaColumnTypeInteger, References: &schemaReferenceSpec{Table: "proyectos", Column: "id", OnDelete: "SET NULL"}},
				{Name: "transporte", Type: schemaColumnTypeText, NotNull: true},
				{Name: "handle_kind", Type: schemaColumnTypeText, NotNull: true},
				{Name: "handle_ref", Type: schemaColumnTypeText, NotNull: true},
				{Name: "estado", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("activo"), CheckSQL: "estado IN ('activo','pausado','cerrado','fallido')"},
				{Name: "metadata_json", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("{}")},
				{Name: "created_at", Type: schemaColumnTypeDateTime, NotNull: true, DefaultSQL: "CURRENT_TIMESTAMP"},
				{Name: "updated_at", Type: schemaColumnTypeDateTime, NotNull: true, DefaultSQL: "CURRENT_TIMESTAMP"},
			},
		},
		{
			Name: "runtime_orders",
			Columns: []schemaColumnSpec{
				{Name: "id", Type: schemaColumnTypeInteger, PrimaryKey: true, AutoIncrement: true},
				{Name: "agente", Type: schemaColumnTypeText, NotNull: true, References: &schemaReferenceSpec{Table: "agentes", Column: "nombre"}},
				{Name: "proyecto_id", Type: schemaColumnTypeInteger, References: &schemaReferenceSpec{Table: "proyectos", Column: "id", OnDelete: "SET NULL"}},
				{Name: "tipo", Type: schemaColumnTypeText, NotNull: true, CheckSQL: "tipo IN ('enviar_instruccion','pausar','continuar','handoff')"},
				{Name: "payload_json", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("{}")},
				{Name: "estado", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("pendiente"), CheckSQL: "estado IN ('pendiente','ejecutando','completada','fallida')"},
				{Name: "error_text", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("")},
				{Name: "created_at", Type: schemaColumnTypeDateTime, NotNull: true, DefaultSQL: "CURRENT_TIMESTAMP"},
				{Name: "started_at", Type: schemaColumnTypeDateTime},
				{Name: "finished_at", Type: schemaColumnTypeDateTime},
			},
		},
	}
}

func capacitySchemaSpecs() []schemaTableSpec {
	return []schemaTableSpec{
		{
			Name: "pools_capacidad",
			Columns: []schemaColumnSpec{
				{Name: "id", Type: schemaColumnTypeInteger, PrimaryKey: true, AutoIncrement: true},
				{Name: "slug", Type: schemaColumnTypeText, NotNull: true, Unique: true},
				{Name: "proveedor", Type: schemaColumnTypeText, NotNull: true},
				{Name: "runtime", Type: schemaColumnTypeText, NotNull: true},
				{Name: "plan", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("")},
				{Name: "es_de_pago", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "0"},
				{Name: "capacidad_total", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "1"},
				{Name: "capacidad_reservada", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "0"},
				{Name: "permite_hijos", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "1"},
				{Name: "permite_modelos_multi", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "1"},
				{Name: "permite_sobrecoste", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "0"},
				{Name: "politica_handoff", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("preventivo")},
				{Name: "fuente_telemetria", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("manual")},
				{Name: "metadata_json", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("{}")},
				{Name: "activo", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "1"},
				{Name: "created_at", Type: schemaColumnTypeDateTime, NotNull: true, DefaultSQL: "CURRENT_TIMESTAMP"},
				{Name: "updated_at", Type: schemaColumnTypeDateTime, NotNull: true, DefaultSQL: "CURRENT_TIMESTAMP"},
			},
		},
		{
			Name: "pool_modelos",
			Columns: []schemaColumnSpec{
				{Name: "id", Type: schemaColumnTypeInteger, PrimaryKey: true, AutoIncrement: true},
				{Name: "pool_id", Type: schemaColumnTypeInteger, NotNull: true, References: &schemaReferenceSpec{Table: "pools_capacidad", Column: "id", OnDelete: "CASCADE"}},
				{Name: "model_slug", Type: schemaColumnTypeText, NotNull: true},
				{Name: "activo", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "1"},
				{Name: "prioridad", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "100"},
				{Name: "coste_relativo", Type: schemaColumnTypeReal, NotNull: true, DefaultSQL: "1.0"},
				{Name: "limite_conocido_json", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("{}")},
				{Name: "created_at", Type: schemaColumnTypeDateTime, NotNull: true, DefaultSQL: "CURRENT_TIMESTAMP"},
				{Name: "updated_at", Type: schemaColumnTypeDateTime, NotNull: true, DefaultSQL: "CURRENT_TIMESTAMP"},
			},
			UniqueConstraints: []schemaUniqueConstraintSpec{
				{Columns: []string{"pool_id", "model_slug"}},
			},
		},
		{
			Name: "politicas_modelo",
			Columns: []schemaColumnSpec{
				{Name: "id", Type: schemaColumnTypeInteger, PrimaryKey: true, AutoIncrement: true},
				{Name: "scope_tipo", Type: schemaColumnTypeText, NotNull: true, CheckSQL: "scope_tipo IN ('global','perfil','proyecto','fase','tarea')"},
				{Name: "scope_ref", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("")},
				{Name: "perfil_tarea", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("*")},
				{Name: "pool_slug", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("")},
				{Name: "model_slug", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("")},
				{Name: "reasoning_effort", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("")},
				{Name: "prioridad", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "100"},
				{Name: "activa", Type: schemaColumnTypeInteger, NotNull: true, DefaultSQL: "1"},
				{Name: "metadata_json", Type: schemaColumnTypeText, NotNull: true, DefaultSQL: quoteSchemaSeedValue("{}")},
				{Name: "created_at", Type: schemaColumnTypeDateTime, NotNull: true, DefaultSQL: "CURRENT_TIMESTAMP"},
				{Name: "updated_at", Type: schemaColumnTypeDateTime, NotNull: true, DefaultSQL: "CURRENT_TIMESTAMP"},
			},
		},
	}
}

func renderSectionDDLForDriver(driver string, tables []schemaTableSpec) string {
	parts := make([]string, 0, len(tables))
	for _, table := range tables {
		parts = append(parts, renderTableSpecForDriver(driver, table))
	}
	return joinDDLParts(parts...)
}

func renderBootstrapSectionDDLForDriver(driver string) string {
	return renderSectionDDLForDriver(driver, bootstrapSchemaSpecs())
}

func renderRuntimeSectionDDLForDriver(driver string) string {
	return renderSectionDDLForDriver(driver, runtimeSchemaSpecs())
}

func renderCapacitySectionDDLForDriver(driver string) string {
	return renderSectionDDLForDriver(driver, capacitySchemaSpecs())
}

func renderTableSpecForDriver(driver string, table schemaTableSpec) string {
	var b strings.Builder
	b.WriteString("CREATE TABLE IF NOT EXISTS ")
	b.WriteString(table.Name)
	b.WriteString(" (\n")
	lineCount := len(table.Columns) + len(table.UniqueConstraints)
	lineNo := 0
	for _, column := range table.Columns {
		b.WriteString("    ")
		b.WriteString(renderColumnSpecForDriver(driver, column))
		lineNo++
		if lineNo < lineCount {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	for _, unique := range table.UniqueConstraints {
		b.WriteString("    UNIQUE(")
		b.WriteString(strings.Join(unique.Columns, ", "))
		b.WriteString(")")
		lineNo++
		if lineNo < lineCount {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(");")
	return b.String()
}

func renderColumnSpecForDriver(driver string, column schemaColumnSpec) string {
	if column.AutoIncrement && column.PrimaryKey {
		switch normalizedDriverName(driver) {
		case "postgres":
			return column.Name + " INTEGER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY"
		default:
			return column.Name + " INTEGER PRIMARY KEY AUTOINCREMENT"
		}
	}

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
	if column.Unique {
		parts = append(parts, "UNIQUE")
	}
	if column.CheckSQL != "" {
		parts = append(parts, "CHECK ("+column.CheckSQL+")")
	}
	if column.References != nil {
		ref := "REFERENCES " + column.References.Table + "(" + column.References.Column + ")"
		if column.References.OnDelete != "" {
			ref += " ON DELETE " + column.References.OnDelete
		}
		parts = append(parts, ref)
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
	case schemaColumnTypeReal:
		return "REAL"
	default:
		return "TEXT"
	}
}
