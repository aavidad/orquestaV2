package db

import (
	"testing"
	"time"
)

func TestPurgarDatosOperacionalesPurgaMailboxTerminalVieja(t *testing.T) {
	prepararDBTemporal(t)

	old := time.Now().UTC().Add(-25 * time.Hour)
	recent := time.Now().UTC().Add(-2 * time.Hour)
	casos := []struct {
		estado string
		ts     time.Time
	}{
		{estado: "consumido", ts: old},
		{estado: "cancelado", ts: old},
		{estado: "expirado", ts: old},
		{estado: "consumido", ts: recent},
		{estado: "pendiente", ts: old},
		{estado: "entregado", ts: old},
	}
	for _, c := range casos {
		deliveredAt := any(nil)
		consumedAt := any(nil)
		switch c.estado {
		case "consumido", "cancelado":
			deliveredAt = c.ts
			consumedAt = c.ts
		case "expirado":
			deliveredAt = c.ts
		}
		res, err := DB.Exec(`INSERT INTO runtime_mailbox (from_agente, to_agente, kind, estado, created_at, delivered_at, consumed_at) VALUES (?,?,?,?,?,?,?)`,
			"server", "Codex1", "autonomia", c.estado, c.ts, deliveredAt, consumedAt)
		if err != nil {
			t.Fatalf("insert mailbox %s: %v", c.estado, err)
		}
		if _, err := res.RowsAffected(); err != nil {
			t.Fatalf("rows affected insert mailbox %s: %v", c.estado, err)
		}
	}

	n, err := PurgarDatosOperacionales()
	if err != nil {
		t.Fatalf("purgar datos operacionales: %v", err)
	}
	if n < 3 {
		t.Fatalf("deberia purgar al menos 3 mensajes terminales viejos, got=%d", n)
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_mailbox`).Scan(&count); err != nil {
		t.Fatalf("count runtime_mailbox: %v", err)
	}
	if count != 3 {
		t.Fatalf("runtime_mailbox restante=%d want 3", count)
	}

	rows, err := DB.Query(`SELECT estado FROM runtime_mailbox ORDER BY id`)
	if err != nil {
		t.Fatalf("listar runtime_mailbox: %v", err)
	}
	defer rows.Close()
	got := []string{}
	for rows.Next() {
		var estado string
		if err := rows.Scan(&estado); err != nil {
			t.Fatalf("scan estado: %v", err)
		}
		got = append(got, estado)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iter runtime_mailbox: %v", err)
	}
	want := []string{"consumido", "pendiente", "entregado"}
	if len(got) != len(want) {
		t.Fatalf("estados restantes=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("estado[%d]=%s want %s all=%v", i, got[i], want[i], got)
		}
	}
}
