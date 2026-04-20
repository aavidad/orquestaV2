package db

import (
	"testing"
	"time"
)

func TestPurgarRuntimeHistoricoEliminaMailboxTerminalLigadaAOrdenPurgada(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := ConfigSet("runtime_orders_retention_minutes", "30"); err != nil {
		t.Fatalf("config runtime_orders_retention_minutes: %v", err)
	}

	old := time.Now().UTC().Add(-2 * time.Hour)
	res, err := DB.Exec(`INSERT INTO runtime_orders
		(agente, tipo, estado, payload_json, created_at, available_at, finished_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		"Codex1", "nudge", "completada", `{}`, old, old, old, old)
	if err != nil {
		t.Fatalf("insert runtime_order: %v", err)
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert order: %v", err)
	}

	if _, err := DB.Exec(`INSERT INTO runtime_mailbox
		(from_agente, to_agente, runtime_order_id, kind, payload_json, estado, created_at, delivered_at, consumed_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		"server", "Codex1", orderID, "autonomia", `{}`, "consumido", old, old, old); err != nil {
		t.Fatalf("insert mailbox consumido: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO runtime_mailbox
		(from_agente, to_agente, runtime_order_id, kind, payload_json, estado, created_at, delivered_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		"server", "Codex1", orderID, "autonomia", `{}`, "entregado", old, old); err != nil {
		t.Fatalf("insert mailbox entregado: %v", err)
	}

	resultado, err := PurgarRuntimeHistorico()
	if err != nil {
		t.Fatalf("PurgarRuntimeHistorico: %v", err)
	}
	if resultado == nil || resultado.Orders == nil || resultado.Orders.Deleted != 1 {
		t.Fatalf("purga orders inesperada: %+v", resultado)
	}

	var ordersCount int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_orders WHERE id=?`, orderID).Scan(&ordersCount); err != nil {
		t.Fatalf("count runtime_orders: %v", err)
	}
	if ordersCount != 0 {
		t.Fatalf("runtime_order deberia desaparecer, count=%d", ordersCount)
	}

	rows, err := DB.Query(`SELECT estado, runtime_order_id FROM runtime_mailbox ORDER BY id`)
	if err != nil {
		t.Fatalf("listar runtime_mailbox: %v", err)
	}
	defer rows.Close()
	type mailboxState struct {
		estado string
		order  any
	}
	var got []mailboxState
	for rows.Next() {
		var (
			estado string
			order  any
		)
		if err := rows.Scan(&estado, &order); err != nil {
			t.Fatalf("scan runtime_mailbox: %v", err)
		}
		got = append(got, mailboxState{estado: estado, order: order})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iter runtime_mailbox: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("mailbox restante inesperada: %+v", got)
	}
	if got[0].estado != "entregado" {
		t.Fatalf("deberia conservar solo la mailbox no terminal, got=%+v", got)
	}
	if got[0].order != nil {
		t.Fatalf("mailbox restante deberia quedar desacoplada de la order purgada, got=%+v", got)
	}
}
