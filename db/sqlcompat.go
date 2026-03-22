package db

import (
	"database/sql"
	"fmt"
	"strings"
)

type upsertAssignment struct {
	Column string
	Expr   string
}

func insertIgnoreValuesSQL(table string, columns, conflictColumns []string) string {
	return buildInsertIgnoreValuesSQL(DriverName(), table, columns, conflictColumns)
}

func insertIgnoreSelectSQL(table string, columns, conflictColumns []string, selectSQL string) string {
	return buildInsertIgnoreSelectSQL(DriverName(), table, columns, conflictColumns, selectSQL)
}

func buildInsertIgnoreValuesSQL(driver, table string, columns, conflictColumns []string) string {
	values := make([]string, len(columns))
	for i := range columns {
		values[i] = "?"
	}
	return buildInsertIgnoreSQL(driver, table, columns, conflictColumns, "VALUES ("+strings.Join(values, ",")+")")
}

func buildInsertIgnoreSelectSQL(driver, table string, columns, conflictColumns []string, selectSQL string) string {
	return buildInsertIgnoreSQL(driver, table, columns, conflictColumns, strings.TrimSpace(selectSQL))
}

func buildInsertIgnoreSQL(driver, table string, columns, conflictColumns []string, tail string) string {
	head := "INSERT INTO"
	if strings.EqualFold(strings.TrimSpace(driver), "mysql") {
		head = "INSERT IGNORE INTO"
	}
	stmt := fmt.Sprintf("%s %s (%s) %s", head, table, strings.Join(columns, ","), strings.TrimSpace(tail))
	if head == "INSERT IGNORE INTO" {
		return stmt
	}
	return stmt + " ON CONFLICT(" + strings.Join(conflictColumns, ",") + ") DO NOTHING"
}

func upsertValuesSQL(table string, insertColumns, conflictColumns []string, updateAssignments []upsertAssignment) string {
	return buildUpsertValuesSQL(DriverName(), table, insertColumns, conflictColumns, updateAssignments)
}

func buildUpsertValuesSQL(driver, table string, insertColumns, conflictColumns []string, updateAssignments []upsertAssignment) string {
	values := make([]string, len(insertColumns))
	for i := range insertColumns {
		values[i] = "?"
	}
	stmt := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(insertColumns, ","),
		strings.Join(values, ","),
	)
	updates := buildUpsertAssignments(driver, updateAssignments)
	if len(updates) == 0 {
		return stmt
	}
	if strings.EqualFold(strings.TrimSpace(driver), "mysql") {
		return stmt + " ON DUPLICATE KEY UPDATE " + strings.Join(updates, ",")
	}
	return stmt + " ON CONFLICT(" + strings.Join(conflictColumns, ",") + ") DO UPDATE SET " + strings.Join(updates, ",")
}

func buildUpsertAssignments(driver string, updateAssignments []upsertAssignment) []string {
	assignments := make([]string, 0, len(updateAssignments))
	isMySQL := strings.EqualFold(strings.TrimSpace(driver), "mysql")
	for _, assignment := range updateAssignments {
		column := strings.TrimSpace(assignment.Column)
		if column == "" {
			continue
		}
		expr := strings.TrimSpace(assignment.Expr)
		if expr == "" {
			if isMySQL {
				expr = "VALUES(" + column + ")"
			} else {
				expr = "excluded." + column
			}
		}
		assignments = append(assignments, column+"="+expr)
	}
	return assignments
}

type queryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

func insertReturningID(query string, args ...any) (int64, error) {
	return insertReturningIDWith(DB, query, args...)
}

func insertReturningIDWith(q queryRower, query string, args ...any) (int64, error) {
	stmt := strings.TrimRight(strings.TrimSpace(query), ";")
	if !strings.Contains(strings.ToUpper(stmt), "RETURNING") {
		stmt += "\nRETURNING id"
	}
	var id int64
	if err := q.QueryRow(stmt, args...).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
