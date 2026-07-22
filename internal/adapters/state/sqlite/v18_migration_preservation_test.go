package sqlite

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var v18MigrationHistoryTables = []string{
	"goals", "work_items", "executions", "workspace_bindings", "change_sets",
	"attestations", "artifacts", "artifact_occurrences", "integration_receipts", "outbox",
}

type v18MigrationTableSnapshot struct {
	Columns []string
	Rows    []string
}

type v18MigrationHistorySnapshot map[string]v18MigrationTableSnapshot

func snapshotV18MigrationHistoryPath(t *testing.T, path string) v18MigrationHistorySnapshot {
	t.Helper()
	database, err := sql.Open(driverName, path)
	sqliteTestNoError(t, err)
	defer database.Close()
	return snapshotV18MigrationHistory(t, database, nil)
}

func snapshotV18MigrationHistory(t *testing.T, database *sql.DB,
	contract v18MigrationHistorySnapshot,
) v18MigrationHistorySnapshot {
	t.Helper()
	result := make(v18MigrationHistorySnapshot, len(v18MigrationHistoryTables))
	for _, table := range v18MigrationHistoryTables {
		columns := v18SnapshotColumns(t, database, table)
		if expected, found := contract[table]; found {
			columns = append([]string(nil), expected.Columns...)
		}
		quoted := make([]string, len(columns))
		for index, column := range columns {
			quoted[index] = `"` + strings.ReplaceAll(column, `"`, `""`) + `"`
		}
		rows, err := database.Query(`SELECT ` + strings.Join(quoted, ",") + ` FROM "` +
			strings.ReplaceAll(table, `"`, `""`) + `"`)
		sqliteTestNoError(t, err)
		values := make([]string, 0)
		for rows.Next() {
			cells := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for index := range cells {
				pointers[index] = &cells[index]
			}
			sqliteTestNoError(t, rows.Scan(pointers...))
			encoded := make([]string, len(cells))
			for index, cell := range cells {
				encoded[index] = encodeV18SnapshotCell(cell)
			}
			values = append(values, strings.Join(encoded, "\x00"))
		}
		sqliteTestNoError(t, rows.Close())
		sort.Strings(values)
		result[table] = v18MigrationTableSnapshot{Columns: columns, Rows: values}
	}
	return result
}

func v18SnapshotColumns(t *testing.T, database *sql.DB, table string) []string {
	t.Helper()
	rows, err := database.Query(`PRAGMA table_info("` + strings.ReplaceAll(table, `"`, `""`) + `")`)
	sqliteTestNoError(t, err)
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue any
		sqliteTestNoError(t, rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey))
		columns = append(columns, name)
	}
	if len(columns) == 0 {
		t.Fatalf("V18 migration history table %s missing", table)
	}
	return columns
}

func encodeV18SnapshotCell(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case int64:
		return fmt.Sprintf("int:%d", typed)
	case float64:
		return fmt.Sprintf("float:%g", typed)
	case []byte:
		return "bytes:" + hex.EncodeToString(typed)
	case string:
		return "text:" + typed
	default:
		return fmt.Sprintf("%T:%v", typed, typed)
	}
}

func assertPopulatedV18MigrationHistory(t *testing.T, snapshot v18MigrationHistorySnapshot) {
	t.Helper()
	for _, table := range v18MigrationHistoryTables {
		if len(snapshot[table].Rows) == 0 {
			t.Fatalf("V18 migration fixture lacks populated %s history", table)
		}
	}
}

func assertV18MigrationHistoryEqual(t *testing.T, before, after v18MigrationHistorySnapshot) {
	t.Helper()
	if !reflect.DeepEqual(after, before) {
		for _, table := range v18MigrationHistoryTables {
			if !reflect.DeepEqual(after[table], before[table]) {
				t.Fatalf("V18 migration changed %s history\nbefore=%+v\nafter=%+v", table, before[table], after[table])
			}
		}
		t.Fatal("V18 migration changed durable V17 history")
	}
}
