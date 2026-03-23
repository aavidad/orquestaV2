package db

import (
	"errors"
	"strings"
	"testing"
)

func TestEsErrorMigracionIgnorable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "sqlite duplicate column", err: errors.New("duplicate column name: proyecto_id"), want: true},
		{name: "sqlite already exists", err: errors.New("table proyectos already exists"), want: true},
		{name: "mysql duplicate key", err: errors.New("Error 1061: duplicate key name 'idx_runtime_handles_agente_activo'"), want: true},
		{name: "real failure", err: errors.New("syntax error near FROM"), want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := esErrorMigracionIgnorable(tc.err)
			if got != tc.want {
				t.Fatalf("esErrorMigracionIgnorable(%v)=%v want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestPostMigrationStatementsNoRecreanDDL(t *testing.T) {
	t.Parallel()

	stmts := postMigrationStatements()
	if len(stmts) == 0 {
		t.Fatalf("postMigrationStatements vacio")
	}

	for _, stmt := range stmts {
		upper := strings.ToUpper(strings.TrimSpace(stmt))
		for _, forbidden := range []string{
			"CREATE TABLE",
			"CREATE TRIGGER",
			"CREATE UNIQUE INDEX",
			"CREATE INDEX",
		} {
			if strings.HasPrefix(upper, forbidden) {
				t.Fatalf("postMigrationStatements no deberia contener %q: %s", forbidden, stmt)
			}
		}
	}
}

func TestPostMigrationStatementsIncluyenAlterYAjustesDatos(t *testing.T) {
	t.Parallel()

	stmts := strings.Join(postMigrationStatements(), "\n")
	for _, required := range []string{
		"ALTER TABLE agentes ADD COLUMN estado_sesion",
		"ALTER TABLE propuestas ADD COLUMN proyecto_id",
		"ALTER TABLE sesiones ADD COLUMN conector_id",
		"UPDATE reglas",
		"UPDATE workflows",
	} {
		if !strings.Contains(stmts, required) {
			t.Fatalf("postMigrationStatements deberia incluir %q", required)
		}
	}
}

func TestPostMigrationStatementsForDriverPostgresVacioPorAhora(t *testing.T) {
	t.Parallel()

	if got := postMigrationStatementsForDriver("postgres"); len(got) != 0 {
		t.Fatalf("postgres no deberia reutilizar post-migraciones sqlite-first por ahora: %v", got)
	}
}
