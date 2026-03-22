package db

import "testing"

func TestTableExistsQueryPorDriver(t *testing.T) {
	t.Parallel()

	cases := []struct {
		driver string
		want   string
	}{
		{"sqlite", `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`},
		{"sqlite3", `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`},
		{"mysql", `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`},
		{"postgres", `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?`},
		{"postgresql", `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?`},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.driver, func(t *testing.T) {
			t.Parallel()
			if got := tableExistsQuery(tc.driver); got != tc.want {
				t.Fatalf("query inesperada:\n%s", got)
			}
		})
	}
}
