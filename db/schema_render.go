package db

import (
	"database/sql"
	"fmt"
	"strings"
)

func schemaDDL() string {
	return schemaDDLForDriver(DriverName())
}

func schemaDDLForDriver(driver string) string {
	driver = normalizedDriverName(driver)
	statements := schemaStatements(Schema)
	out := make([]string, 0, len(statements))
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		upper := strings.ToUpper(stripLeadingSQLComments(trimmed))
		if strings.HasPrefix(upper, "INSERT ") || strings.HasPrefix(upper, "UPDATE ") {
			continue
		}
		out = append(out, trimmed)
	}
	return joinDDLParts(out...)
}

func schemaSeedData() string {
	return schemaSeedDataForDriver(DriverName())
}

func schemaSeedDataForDriver(driver string) string {
	driver = normalizedDriverName(driver)
	statements := schemaStatements(Schema)
	out := make([]string, 0, len(statements))
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		upper := strings.ToUpper(stripLeadingSQLComments(trimmed))
		if strings.HasPrefix(upper, "INSERT ") || strings.HasPrefix(upper, "UPDATE ") {
			out = append(out, renderSeedStatementForDriver(driver, trimmed))
		}
	}
	return joinDDLParts(out...)
}

func renderSeedStatementForDriver(driver, stmt string) string {
	if normalizedDriverName(driver) == "mysql" {
		stmt = strings.Replace(stmt, "INSERT OR IGNORE INTO", "INSERT IGNORE INTO", 1)
	}
	return stmt
}

func joinDDLParts(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, "\n\n")
}

func stripLeadingSQLComments(stmt string) string {
	stmt = strings.TrimSpace(stmt)
	for {
		switch {
		case stmt == "":
			return ""
		case strings.HasPrefix(stmt, "--"):
			if idx := strings.IndexByte(stmt, '\n'); idx >= 0 {
				stmt = strings.TrimSpace(stmt[idx+1:])
				continue
			}
			return ""
		default:
			return stmt
		}
	}
}

func schemaStatements(schema string) []string {
	schema = strings.TrimSpace(schema)
	if schema == "" {
		return nil
	}
	var (
		statements []string
		current    strings.Builder
		inSingle   bool
		inDouble   bool
	)
	flush := func() {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
		current.Reset()
	}
	for i := 0; i < len(schema); i++ {
		ch := schema[i]
		switch ch {
		case '\'':
			if !inDouble {
				if inSingle && i+1 < len(schema) && schema[i+1] == '\'' {
					current.WriteByte(ch)
					current.WriteByte(schema[i+1])
					i++
					continue
				}
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case ';':
			current.WriteByte(ch)
			if !inSingle && !inDouble {
				flush()
				continue
			}
		}
		current.WriteByte(ch)
	}
	flush()
	return statements
}

func aplicarDDL(db *sql.DB, ddl string) error {
	ddl = strings.TrimSpace(ddl)
	if ddl == "" {
		return fmt.Errorf("schema DDL vacio")
	}
	_, err := db.Exec(ddl)
	return err
}

func aplicarSeedSQL(db *sql.DB, seeds string) error {
	seeds = strings.TrimSpace(seeds)
	if seeds == "" {
		return nil
	}
	_, err := db.Exec(seeds)
	return err
}
