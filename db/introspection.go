package db

import "fmt"

func TableExists(table string) (bool, error) {
	if DB == nil {
		return false, fmt.Errorf("db no inicializada")
	}
	var count int
	if err := DB.QueryRow(tableExistsQuery(DriverName()), table).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func ColumnExists(table, column string) (bool, error) {
	if DB == nil {
		return false, fmt.Errorf("db no inicializada")
	}
	var count int
	if err := DB.QueryRow(columnExistsQuery(DriverName()), table, column).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func SchemaObjectExists(kind, name string) (bool, error) {
	if DB == nil {
		return false, fmt.Errorf("db no inicializada")
	}
	var count int
	query, args, err := schemaObjectExistsQuery(DriverName(), kind, name)
	if err != nil {
		return false, err
	}
	if err := DB.QueryRow(query, args...).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func tableExistsQuery(driver string) string {
	switch normalizedDriverName(driver) {
	case "postgres":
		return `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?`
	case "mysql":
		return `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`
	default:
		return `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	}
}

func columnExistsQuery(driver string) string {
	switch normalizedDriverName(driver) {
	case "postgres":
		return `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`
	case "mysql":
		return `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`
	default:
		return `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`
	}
}

func schemaObjectExistsQuery(driver, kind, name string) (string, []any, error) {
	driver = normalizedDriverName(driver)
	switch driver {
	case "postgres":
		switch kind {
		case "table":
			return tableExistsQuery(driver), []any{name}, nil
		case "index":
			return `SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = ?`, []any{name}, nil
		case "trigger":
			return `SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema = current_schema() AND trigger_name = ?`, []any{name}, nil
		}
	case "mysql":
		switch kind {
		case "table":
			return tableExistsQuery(driver), []any{name}, nil
		case "index":
			return `SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND index_name = ?`, []any{name}, nil
		case "trigger":
			return `SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema = DATABASE() AND trigger_name = ?`, []any{name}, nil
		}
	default:
		switch kind {
		case "table", "index", "trigger":
			return `SELECT COUNT(*) FROM sqlite_master WHERE type = ? AND name = ?`, []any{kind, name}, nil
		}
	}
	return "", nil, fmt.Errorf("tipo de objeto de schema no soportado: %s", kind)
}
