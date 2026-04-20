package db

import "testing"

func TestPurgarRuntimeHistoricoEliminaMailboxTerminalHuerfana(t *testing.T) {
	prepararDBTemporal(t)

	if err := ConfigSet("runtime_orders_retention_minutes", "30"); err != nil {
		t.Fatalf("config runtime_orders_retention_minutes: %v", err)
	}

	res, err := DB.Exec(`INSERT INTO runtime_mailbox
		(from_agente, to_agente, kind, payload_json, estado, created_at, delivered_at, consumed_at)
		VALUES ('server','Codex1','autonomia','{}','consumido',
		        datetime('now','-2 hours'),
		        datetime('now','-2 hours'),
		        datetime('now','-2 hours'))`)
	if err != nil {
		t.Fatalf("insert mailbox huerfana: %v", err)
	}
	mailboxID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert mailbox: %v", err)
	}

	resultado, err := PurgarRuntimeHistorico()
	if err != nil {
		t.Fatalf("PurgarRuntimeHistorico: %v", err)
	}
	if resultado == nil {
		t.Fatal("resultado nulo")
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_mailbox WHERE id=?`, mailboxID).Scan(&count); err != nil {
		t.Fatalf("count mailbox: %v", err)
	}
	if count != 0 {
		t.Fatalf("la mailbox terminal huerfana deberia purgarse, count=%d", count)
	}
}
