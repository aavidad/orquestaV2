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

func tableExistsQuery(driver string) string {
	switch driver {
	case "postgres":
		return `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?`
	case "mysql":
		return `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`
	default:
		return `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	}
}
