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

func TestPostMigrationStatementsForDriverPostgresRenderizaMigracionesIncrementales(t *testing.T) {
	t.Parallel()

	stmts := strings.Join(postMigrationStatementsForDriver("postgres"), "\n")
	for _, required := range []string{
		"ALTER TABLE agentes ADD COLUMN estado_sesion TEXT DEFAULT NULL",
		"ALTER TABLE sesiones ADD COLUMN heartbeat_at TIMESTAMP",
		"ALTER TABLE runtime_orders ADD COLUMN available_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
		"UPDATE reglas",
		"UPDATE workflows",
	} {
		if !strings.Contains(stmts, required) {
			t.Fatalf("postgres deberia incluir %q", required)
		}
	}
	for _, forbidden := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_tareas_proyecto_blueprint_key",
		"CREATE INDEX IF NOT EXISTS idx_sesiones_activa_id",
	} {
		if strings.Contains(stmts, forbidden) {
			t.Fatalf("postgres no deberia reutilizar DDL complementario sqlite %q", forbidden)
		}
	}
}

func TestPostMigrationStatementsForDriverSQLiteConservaDDLComplementario(t *testing.T) {
	t.Parallel()

	stmts := strings.Join(postMigrationStatementsForDriver("sqlite"), "\n")
	if !strings.Contains(stmts, "CREATE UNIQUE INDEX IF NOT EXISTS idx_tareas_proyecto_blueprint_key") {
		t.Fatalf("sqlite deberia conservar el DDL complementario para upgrades heredados")
	}
	for _, required := range []string{
		"CREATE INDEX IF NOT EXISTS idx_sesiones_activa_id",
		"CREATE INDEX IF NOT EXISTS idx_sesiones_agente_activa_id",
		"CREATE INDEX IF NOT EXISTS idx_sesiones_agente_id",
		"CREATE INDEX IF NOT EXISTS idx_asignaciones_agente_estado_proyecto_id",
		"CREATE INDEX IF NOT EXISTS idx_asignaciones_proyecto_estado_agente_id",
		"CREATE INDEX IF NOT EXISTS idx_presupuestos_sesion_sesion_checked_id",
		"CREATE INDEX IF NOT EXISTS idx_presupuestos_sesion_sesion_fuente_checked_id",
		"CREATE INDEX IF NOT EXISTS idx_propuestas_estado_proyecto_id",
		"CREATE INDEX IF NOT EXISTS idx_runtime_mailbox_estado_id",
		"CREATE INDEX IF NOT EXISTS idx_tareas_agente_estado_id",
		"CREATE INDEX IF NOT EXISTS idx_tareas_estado_id",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_accion_id",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_agente_id",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_entidad_entidadid_id",
		"CREATE INDEX IF NOT EXISTS idx_runtime_instances_agente_updated_id",
		"CREATE INDEX IF NOT EXISTS idx_votos_agente_posicion_propuesta",
	} {
		if !strings.Contains(stmts, required) {
			t.Fatalf("sqlite deberia conservar el indice complementario %q", required)
		}
	}
}
