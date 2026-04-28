package db

import "database/sql"

func rebuildSQLiteTable(
	db *sql.DB,
	tableName string,
	indexes []string,
	createTableSQL string,
	createIndexes []string,
	copyBuilder func(legacy string, legacyCols map[string]bool) (string, []any),
) error {
	return rebuildLegacyTable(db, tableName, indexes, createTableSQL, createIndexes, copyBuilder)
}
