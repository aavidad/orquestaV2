package db

import (
	"database/sql"
	"fmt"
	"strings"
)

const schemaInitialDataMarker = "-- ─── Datos iniciales"

func schemaDDL() string {
	parts := strings.SplitN(Schema, schemaInitialDataMarker, 2)
	return strings.TrimSpace(parts[0])
}

func schemaSeedData() string {
	parts := strings.SplitN(Schema, schemaInitialDataMarker, 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(schemaInitialDataMarker + parts[1])
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
