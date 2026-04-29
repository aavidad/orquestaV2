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

func TestColumnExistsQueryPorDriver(t *testing.T) {
	t.Parallel()

	cases := []struct {
		driver string
		want   string
	}{
		{"sqlite", `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`},
		{"sqlite3", `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`},
		{"mysql", `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`},
		{"postgres", `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`},
		{"postgresql", `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.driver, func(t *testing.T) {
			t.Parallel()
			if got := columnExistsQuery(tc.driver); got != tc.want {
				t.Fatalf("query inesperada:\n%s", got)
			}
		})
	}
}

func TestSchemaObjectExistsQueryPorDriver(t *testing.T) {
	t.Parallel()

	cases := []struct {
		driver string
		kind   string
		want   string
		args   []any
	}{
		{"sqlite", "trigger", `SELECT COUNT(*) FROM sqlite_master WHERE type = ? AND name = ?`, []any{"trigger", "trig_demo"}},
		{"postgres", "index", `SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = ?`, []any{"idx_demo"}},
		{"mysql", "trigger", `SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema = DATABASE() AND trigger_name = ?`, []any{"trig_demo"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.driver+"_"+tc.kind, func(t *testing.T) {
			t.Parallel()
			gotQuery, gotArgs, err := schemaObjectExistsQuery(tc.driver, tc.kind, tc.args[len(tc.args)-1].(string))
			if err != nil {
				t.Fatalf("schemaObjectExistsQuery: %v", err)
			}
			if gotQuery != tc.want {
				t.Fatalf("query inesperada:\n%s", gotQuery)
			}
			if len(gotArgs) != len(tc.args) {
				t.Fatalf("args inesperados: %+v", gotArgs)
			}
		})
	}

	if _, _, err := schemaObjectExistsQuery("sqlite", "vista", "v_demo"); err == nil {
		t.Fatalf("esperaba error para tipo de objeto no soportado")
	}
}

func TestSchemaObjectExistsCacheSeInvalidaConDDL(t *testing.T) {
	prepararDBTemporal(t)

	exists, err := SchemaObjectExists("table", "review_gates")
	if err != nil {
		t.Fatalf("SchemaObjectExists inicial: %v", err)
	}
	if !exists {
		t.Fatalf("review_gates deberia existir tras bootstrap")
	}

	if _, err := DB.Exec(`DROP TABLE review_gates`); err != nil {
		t.Fatalf("drop review_gates: %v", err)
	}

	exists, err = SchemaObjectExists("table", "review_gates")
	if err != nil {
		t.Fatalf("SchemaObjectExists tras drop: %v", err)
	}
	if exists {
		t.Fatalf("review_gates no deberia seguir cacheada tras DDL")
	}
}
