package db

import (
	"fmt"
	"strings"
)

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
