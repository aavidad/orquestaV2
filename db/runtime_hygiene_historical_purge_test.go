package db

import (
	"testing"

	"orquesta/internal/controlruntime"
)

func TestProcesarHigieneRuntimesAutonomosBatchIncluyePurgaHistorica(t *testing.T) {
	prepararDBTemporal(t)

	prevCommand := runtimeTMUXCommandPathFn
	prevList := runtimeTMUXListSessionsFn
	prevKill := runtimeTMUXKillSessionFn
	t.Cleanup(func() {
		runtimeTMUXCommandPathFn = prevCommand
		runtimeTMUXListSessionsFn = prevList
		runtimeTMUXKillSessionFn = prevKill
	})
	runtimeTMUXCommandPathFn = func() (string, error) { return "", nil }
	runtimeTMUXListSessionsFn = func(string) ([]controlruntime.TMUXSessionInfo, error) { return nil, nil }
	runtimeTMUXKillSessionFn = func(string, string) error { return nil }

	if err := ConfigSet("runtime_orders_retention_minutes", "30"); err != nil {
		t.Fatalf("config runtime_orders_retention_minutes: %v", err)
	}

	res, err := DB.Exec(`INSERT INTO runtime_orders
		(agente, proyecto_id, tipo, payload_json, estado, created_at, updated_at, finished_at)
		VALUES ('Codex1', 1, 'nudge', '{}', 'completada',
		        datetime('now','-2 hours'),
		        datetime('now','-2 hours'),
		        datetime('now','-2 hours'))`)
	if err != nil {
		t.Fatalf("insert runtime order historica: %v", err)
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert runtime order: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene runtimes autonomos: %v", err)
	}
	if n < 1 {
		t.Fatalf("la higiene deberia contar al menos una purga historica, got=%d", n)
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_orders WHERE id=?`, orderID).Scan(&count); err != nil {
		t.Fatalf("count runtime order: %v", err)
	}
	if count != 0 {
		t.Fatalf("la runtime order historica deberia purgarse, count=%d", count)
	}
}
