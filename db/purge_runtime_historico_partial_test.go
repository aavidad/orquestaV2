package db

import "testing"

func TestPurgarRuntimeHistoricoNoBloqueaOrdersPorHandlesNoPurgables(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := ConfigSet("runtime_handles_retention_minutes", "30"); err != nil {
		t.Fatalf("config runtime_handles_retention_minutes: %v", err)
	}
	if err := ConfigSet("runtime_orders_retention_minutes", "30"); err != nil {
		t.Fatalf("config runtime_orders_retention_minutes: %v", err)
	}

	handleRes, err := DB.Exec(`INSERT INTO runtime_handles
		(agente, transporte, handle_kind, handle_ref, estado, created_at, updated_at)
		VALUES (?,?,?,?, 'cerrado', datetime('now','-2 hours'), datetime('now','-2 hours'))`,
		"Codex1", "tmux", "session", "orq-codex1")
	if err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	handleID, err := handleRes.LastInsertId()
	if err != nil {
		t.Fatalf("last insert handle: %v", err)
	}

	if _, err := DB.Exec(`INSERT INTO runtime_orders
		(agente, handle_id, tipo, estado, payload_json, created_at, available_at, updated_at)
		VALUES (?,?,?,?,?,datetime('now','-90 minutes'),datetime('now','-90 minutes'),datetime('now','-90 minutes'))`,
		"Codex1", handleID, "start", "pendiente", `{}`); err != nil {
		t.Fatalf("insert live order linked to handle: %v", err)
	}

	oldOrderRes, err := DB.Exec(`INSERT INTO runtime_orders
		(agente, tipo, estado, payload_json, created_at, available_at, finished_at, updated_at)
		VALUES (?,?,?,?,datetime('now','-2 hours'),datetime('now','-2 hours'),datetime('now','-2 hours'),datetime('now','-2 hours'))`,
		"Codex1", "nudge", "completada", `{}`)
	if err != nil {
		t.Fatalf("insert historical order: %v", err)
	}
	oldOrderID, err := oldOrderRes.LastInsertId()
	if err != nil {
		t.Fatalf("last insert historical order: %v", err)
	}

	resultado, err := PurgarRuntimeHistorico()
	if err != nil {
		t.Fatalf("PurgarRuntimeHistorico: %v", err)
	}
	if resultado == nil || resultado.Orders == nil || resultado.Orders.Deleted != 1 {
		t.Fatalf("purga de orders inesperada: %+v", resultado)
	}
	if resultado.Handles == nil || resultado.Handles.Deleted != 0 {
		t.Fatalf("no deberia purgar handle bloqueada: %+v", resultado)
	}

	var historicalOrderCount int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_orders WHERE id=?`, oldOrderID).Scan(&historicalOrderCount); err != nil {
		t.Fatalf("count historical order: %v", err)
	}
	if historicalOrderCount != 0 {
		t.Fatalf("la order historica deberia purgarse, count=%d", historicalOrderCount)
	}

	var liveOrderCount int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_orders WHERE handle_id=? AND estado='pendiente'`, handleID).Scan(&liveOrderCount); err != nil {
		t.Fatalf("count live order: %v", err)
	}
	if liveOrderCount != 1 {
		t.Fatalf("la order viva ligada al handle deberia quedar intacta, count=%d", liveOrderCount)
	}
}
