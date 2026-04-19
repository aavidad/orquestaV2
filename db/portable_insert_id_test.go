package db

import "testing"

func mustInsertID(t *testing.T, query string, args ...any) int64 {
	t.Helper()

	id, err := insertReturningID(query, args...)
	if err != nil {
		t.Fatalf("insert returning id: %v", err)
	}
	return id
}
